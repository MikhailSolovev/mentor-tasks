package main

import (
	"fmt"
)

func NewRingBuffer(inCh, outCh chan int) *ringBuffer {
	return &ringBuffer{
		inCh:  inCh,
		outCh: outCh,
	}
}

type ringBuffer struct {
	inCh  chan int
	outCh chan int
}

func (r *ringBuffer) Run() {
	for {
		val, ok := <-r.inCh
		if ok {
			if len(r.outCh) == cap(r.outCh) {
				<-r.outCh
			}
			r.outCh <- val
		} else {
			close(r.outCh)
			break
		}
	}
}

func main() {
	max := 100
	inCh := make(chan int, max)
	outCh := make(chan int, 10)

	for i := 0; i < max; i++ {
		inCh <- i
	}

	rb := NewRingBuffer(inCh, outCh)
	close(inCh)
	rb.Run()

	resSlice := make([]int, 0)
	for res := range outCh {
		resSlice = append(resSlice, res)
	}
	fmt.Println(resSlice)
}
