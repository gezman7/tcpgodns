package tcpgodns

import (
	"math/rand"
)

const Domain = "tcpgodns.com."

func Query(encoded string) []byte {

	msg := Message{
		Header: Header{
			ID: uint16(rand.Uint32()),
			Response: false, Authoritative: false},
		Questions: []Question{
			{
				Name:  mustNewName(Domain),
				Type:  TypeA,
				Class: ClassINET,
			},
		}}
	buf, err := msg.Pack()
	if err != nil {
		panic(err)
	}
	return buf
}

func mustNewName(name string) Name {
	n, err := NewName(name)
	if err != nil {
		panic(err)
	}
	return n
}
func CreateDnsQuery(encoded []byte) []byte {
	buf := make([]byte, 2, 514)
	header := queryHeader()
	builder := NewBuilder(buf, header)
	builder.StartQuestions()
	q := customQuestion(encoded)
	builder.Question(q)

	binaryMessage, err := builder.Finish()
	if err != nil {
		panic("Could not create DNS message")
	}
	return binaryMessage
}

func CreateDnsResponse(encoded []byte) []byte {
	buf := make([]byte, 2, 514)
	header := queryHeader()
	builder := NewBuilder(buf, header)
	builder.Question(customQuestion(encoded))

	_, err := builder.Finish()
	if err != nil {
		panic("Could not create DNS message")
	}
	return buf
}

func queryHeader() Header {

	return Header{
		ID:                 uint16(rand.Uint32()),
		Response:           false, //QR
		OpCode:             0,
		Authoritative:      false, //AA
		Truncated:          false, //TC
		RecursionDesired:   false, //RD
		RecursionAvailable: false, //RA
		RCode:              0,
	}
}

func responseHeader() Header {

	return Header{
		ID:                 uint16(rand.Uint32()),
		Response:           false, //QR
		OpCode:             0,
		Authoritative:      false, //AA
		Truncated:          false, //TC
		RecursionDesired:   true,  //RD
		RecursionAvailable: false, //RA
		RCode:              0,
	}
}

func customQuestion(encoded []byte) Question {

	var domain [255]byte
	encodedDomain := append(encoded[:], []byte(Domain)[:]...)
	if len(encodedDomain) > 255 {
		panic("Message is to big to send")
	}
	copy(domain[:], encodedDomain)
	return Question{
		Name: Name{
			Data:   domain,
			Length: uint8(len(domain)),
		},
		Type:  TypeA,
		Class: ClassINET,
	}
}
