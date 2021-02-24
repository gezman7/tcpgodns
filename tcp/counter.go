package tcp

import "sync"

type idCounter struct {
	wg sync.WaitGroup
	c uint
}

func (ic idCounter) Create(){
	ic.c =0
	ic.wg.Add(1)
}

func (ic idCounter) GetId() uint{

	ic.wg.Wait()

}