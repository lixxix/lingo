package timer

import (
	"fmt"
	"sync"
)

type TimeFunc struct {
	delay int64
	call  func()
}

type TimerQueue struct {
	items []*TimeFunc
	lock  sync.Mutex
}

func (t *TimerQueue) Push(data *TimeFunc) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.items) == 0 {
		t.items = append(t.items, data)
	} else {
		bigindex := -1
		for i := 0; i < len(t.items); i++ {
			if data.delay < t.items[i].delay {
				bigindex = i
				break
			}
		}
		t.items = append(t.items, data)
		if bigindex != -1 {
			length := len(t.items)
			copy(t.items[bigindex+1:length], t.items[bigindex:length-1])
			t.items[bigindex] = data
		}
	}

}

func (t *TimerQueue) Pop() *TimeFunc {
	t.lock.Lock()
	defer func() {
		t.items = t.items[1:]
		t.lock.Unlock()
	}()
	return t.items[0]
}

func (t *TimerQueue) Print() {
	for i := 0; i < len(t.items); i++ {
		fmt.Println(t.items[i].delay)
	}
	fmt.Println()
}

func (t *TimerQueue) Len() int {
	t.lock.Lock()
	defer t.lock.Unlock()
	return len(t.items)
}

func (t *TimerQueue) At(i int) *TimeFunc {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.items) <= i {
		return nil
	}
	return t.items[i]
}
