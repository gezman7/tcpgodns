package tcpgodns

import (
	"errors"
	"fmt"
	"math/rand"
)

type sessionManger struct {
	SessionMap map[uint8]*Session
	fwdPort    string
}

const MaxNum = 256

func NewSessionManger(port string) *sessionManger {

	sessionMap := make(map[uint8]*Session)

	return &sessionManger{

		fwdPort:    port,
		SessionMap: sessionMap,
	}
}

func (sm *sessionManger) EstablishNewSession() (openSlot uint8, err error) {

	openSlot, err = sm.findOpenSlot(uint8(rand.Intn(MaxNum)), 10)
	if err != nil {
		return
	}

	sm.SessionMap[openSlot] = EstablishSession( false, openSlot,0)
	communicator:=DialLocally(sm.fwdPort)
	sm.SessionMap[openSlot].ConnectLocal(communicator)
	return
}

func (sm *sessionManger) findOpenSlot(id uint8, attempts int) (openSlot uint8, err error) {

	if attempts == 0 {
		err = errors.New("could not found an open slot in given attempts")
		return
	}
	if _, occupied := sm.SessionMap[id]; occupied {
		 return sm.findOpenSlot(uint8(rand.Intn(MaxNum)), attempts-1)
	} else {
		openSlot = id
		fmt.Printf("assiging sessionId: %d\n",openSlot)
		return
	}

}
