package utils

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestGet(t *testing.T) {

	send := make(chan []byte, 20)
	buf := CreateBuffer()
	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		time.Sleep(time.Second)
		for b := range send {
			if b == nil {
				return
			}
			buf.PushData(b)
			time.Sleep(time.Millisecond)
			if len(send) == 0 {
				buf.Buffer()
			}
		}
		fmt.Println("OVER", len(send))
		wait.Done()
	}()

	data := []byte("hello world")
	for i := 0; i < 10; i++ {
		send <- data
	}

	time.Sleep(time.Second)
	for i := 0; i < 20; i++ {
		send <- data
	}

	close(send)
	wait.Wait()
	fmt.Println("end")
}
