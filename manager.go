package tcpgodns

import (
	"container/heap"
	"encoding/base32"
	"fmt"
	"github.com/miekg/dns"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type PacketManager struct {
	SessionId     uint8
	LastReceived  uint8
	SentHeap      SentHeap
	ReceivedMap   *MsgRcv
	Conuter       int32
	channelWrite  chan []byte
	channelRead   chan []byte
	packetChannel chan UserPacket
}

const MaxNum = 256

func ManagerFactory(cr chan []byte, cw chan []byte) *PacketManager {
	sentHeap := make(SentHeap, 0)
	heap.Init(&sentHeap)
	receivedMap := MessageReceivedFactory()
	sessionId := rand.Intn(MaxNum)
	packetChannel := make(chan UserPacket)
	return &PacketManager{
		SessionId:     uint8(sessionId),
		LastReceived:  0,
		SentHeap:      sentHeap,
		ReceivedMap:   receivedMap,
		Conuter:       0,
		channelRead:   cr,
		channelWrite:  cw,
		packetChannel: packetChannel,
	}
}
func (pm *PacketManager) LocalStreamIncome() {

	for {
		var data = <-pm.channelRead
		packetId := uint8(atomic.AddInt32(&pm.Conuter, 1))

		userPacket := UserPacket{
			Id:          packetId,
			SessionId:   pm.SessionId,
			LastSeenPid: pm.LastReceived,
			Data:        data,
			Flags:       0, //todo: Flags
		}
		item := msgItemFactory(userPacket)

		fmt.Printf("LocalStreamIncome: registred sentPacket: %d\n", userPacket.Id)
		heap.Push(&pm.SentHeap, &item)

		pm.packetChannel <- userPacket // send packet to dns
	}
}

func (pm *PacketManager) ManageIncome(packet UserPacket) []byte {
	pm.LastReceived = packet.LastSeenPid
	pm.ReceivedMap.Add(packet)
	packets := pm.ReceivedMap.PopAvailablePackets()
	return ToBytes(packets)
}

func (pm *PacketManager) ManageResend(lastSeen uint8, c chan UserPacket) {
	flag := true
	curTime := time.Now()
	fmt.Printf("inside ManageResend -lastSeen:%d\n", lastSeen)
	for flag {
		if pm.SentHeap.Len() == 0 {
			fmt.Printf("heap is empty: no unseen messages in heap. lastSeen:%d\n", lastSeen)
			return
		}
		nextPacket := heap.Pop(&pm.SentHeap)

		if nextPacket.(*msgItem).Sent.After(curTime) {
			heap.Push(&pm.SentHeap, nextPacket)
			return
		}
		if nextPacket.(*msgItem).Packet.Id <= lastSeen {
			continue
		}

		if nextPacket.(*msgItem).Packet.Id > lastSeen {
			packetToSend := UserPacket{
				Id:          nextPacket.(*msgItem).Packet.Id,
				SessionId:   nextPacket.(*msgItem).Packet.SessionId,
				LastSeenPid: pm.LastReceived,
				Data:        nextPacket.(*msgItem).Packet.Data,
				Flags:       0,
			}
			fmt.Printf("resending packet:%v\n", packetToSend)
			c <- packetToSend
			msgItem := msgItemFactory(packetToSend)
			heap.Push(&pm.SentHeap, &msgItem)
		}

	}
}

func ToBytes(packets []UserPacket) []byte {
	ByteStream := make([]byte, 0) // todo: adjust constants of slice
	for _, p := range packets {
		slice := p.Data[:]
		ByteStream = append(ByteStream, slice...)
	}

	return ByteStream
}

type MsgRcv struct {
	lastProcessed uint8
	mu            sync.Mutex
	packetMap     map[uint8]UserPacket
}

func MessageReceivedFactory() *MsgRcv {
	return &MsgRcv{
		lastProcessed: 0,
		packetMap:     make(map[uint8]UserPacket),
	}
}

func (mr *MsgRcv) Add(packet UserPacket) {

	// already got ACK for higher PID
	if packet.Id <= mr.lastProcessed {
		return
	}

	// Already accepted the Packet
	if _, ok := mr.packetMap[packet.Id]; ok {
		fmt.Printf("Packet %d already in Packet map", packet.Id)
		return
	}

	fmt.Printf("Adding Packet %d  to Packet map\n", packet.Id)
	mr.packetMap[packet.Id] = packet

}

/*
Pop All packets that directly connected to the last sent Packet in MsgRcv and
updates the last Sent Packet.
*/
func (mr *MsgRcv) PopAvailablePackets() []UserPacket {

	mr.mu.Lock()

	next := mr.lastProcessed + 1

	keys := make([]int, 0, len(mr.packetMap))

	for k := range mr.packetMap {
		keys = append(keys, int(k))
	}

	sort.Ints(keys)
	packets := make([]UserPacket, len(mr.packetMap))
	lastSent := mr.lastProcessed

	for i, k := range keys {
		if next == uint8(k) {
			packets[i] = mr.packetMap[uint8(k)]
			fmt.Printf("Removing Packet %d  from Packet map to the connection\n", packets[i].Id)

			lastSent = next
			next++
			delete(mr.packetMap, uint8(k))
		} else {
			break
		}
	}
	mr.lastProcessed = lastSent
	mr.mu.Unlock()

	return packets

}

func (pm *PacketManager) HandleDNSResponse(w dns.ResponseWriter, req *dns.Msg) {

	var resp dns.Msg
	resp.SetReply(req)
	resp.Response = true
	question := req.Question[0]
	fmt.Printf("recived dns query\n")
	i := strings.Index(question.Name, ".tcpgodns.com")
	encoded := question.Name[0:i]
	str, err := base32.HexEncoding.DecodeString(encoded)
	if err != nil {
		fmt.Println("Error while decoding query")
		return
	}
	userPacket := BytesToPacket(str)
	bytesToTcp := pm.ManageIncome(userPacket)
	fmt.Printf("userPacket Parsed packetId:%d\n ", userPacket.Id)

	msg := []string{"1", "2"}
	fmt.Printf("size of array:%d size of answer:%d\n", uint16(unsafe.Sizeof(msg)), uint16(len(msg[0])))
	for _, q := range req.Question {
		a := dns.TXT{
			Hdr: dns.RR_Header{
				Name:     q.Name,
				Rrtype:   dns.TypeTXT,
				Class:    dns.ClassINET,
				Ttl:      0,
				Rdlength: uint16(1),
			},
			Txt: msg,
		}
		resp.Answer = append(resp.Answer, &a)
		println("respAnswer:%v", resp.Answer)
	}
	w.WriteMsg(&resp)
	pm.channelWrite <- bytesToTcp

}

func (pm *PacketManager) HandleDnsClient() {

	for {
		var data = <-pm.packetChannel // Listen for local incoming packets.

		fmt.Printf("handleDnsClient: packetId:%d\n", data.Id)

		str := encode(data)
		var msg dns.Msg
		query := str + ".tcpgodns.com." // todo: change domain to custom
		msg.SetQuestion(query, dns.TypeTXT)

		fmt.Printf("Question:%v\n", msg)

		in, err := dns.Exchange(&msg, ":5553") // todo: create main parameters
		if in == nil || err != nil {
			fmt.Printf("Error with the exchange error:%s\n", err.Error())
			continue
		}

		fmt.Printf("respnse recived from dns query: %s\n", in.Answer[0].String())

	}
}

func encode(packet UserPacket) string {
	bytesToEncode := packet.PacketToBytes()
	return base32.HexEncoding.EncodeToString(bytesToEncode)
}

func decode(str string) UserPacket {

	bytes, err := base32.HexEncoding.DecodeString(str)
	if err != nil {
		fmt.Printf("Error with income decoding.")
	}

	return BytesToPacket(bytes)

}
