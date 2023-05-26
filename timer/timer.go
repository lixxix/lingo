package timer

import (
	"sync"
	"time"
)

type Timer struct {
	id       int
	interval int64
	pass     int64
	repeat   int
}

type BaseTimer struct {
	precision int64
	timers    map[int]*Timer
	callback  func(id int)
	start     bool
	to_queue  TimerQueue
	mux       sync.Mutex
	stop      chan bool
}

func MakeTimer(precision time.Duration) BaseTimer {
	return BaseTimer{
		timers:   make(map[int]*Timer),
		stop:     make(chan bool),
		callback: nil,
		to_queue: TimerQueue{
			items: make([]*TimeFunc, 0),
			lock:  sync.Mutex{},
		},
		precision: int64(precision),
		start:     false,
	}
}

func (t *BaseTimer) SetTimerCallBack(call func(id int)) {
	t.callback = call
}

func (t *BaseTimer) KillTimer(id int) {
	go func() {
		t.mux.Lock()
		delete(t.timers, id)
		t.mux.Unlock()
	}()
}

func (t *BaseTimer) StopTimer() {
	t.stop <- true
}

func (t *BaseTimer) SetTimeOut(duration time.Duration, call func()) {
	if !t.start {
		t.timers = make(map[int]*Timer)
		go t.Ticking()
		t.start = true
	}

	t.to_queue.Push(&TimeFunc{
		delay: int64(duration),
		call:  call,
	})
}

func (r *BaseTimer) Ticking() {
	ticker := time.NewTicker(time.Duration(r.precision))
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			r.mux.Lock()
			for key, value := range r.timers {
				r.timers[key].pass -= r.precision
				if r.timers[key].pass <= 0 {
					r.timers[key].pass += value.interval
					if value.repeat != -1 {
						r.timers[key].repeat -= 1
					}
					if r.callback != nil {
						r.callback(key)
					}
					if value.repeat != -1 && value.repeat <= 0 {
						delete(r.timers, key)
					}
				}
			}

			if r.to_queue.Len() > 0 {
				call := 0
				for i := 0; i < r.to_queue.Len(); i++ {
					tick := r.to_queue.At(i)
					if tick != nil {
						tick.delay -= r.precision
						if tick.delay <= 0 {
							call++
						}
					}
				}
				for call > 0 {
					time_call := r.to_queue.Pop()
					if time_call != nil {
						time_call.call()
					}
					call--
				}
			}
			r.mux.Unlock()
		case <-r.stop:
			r.start = false
			return
		}
	}
}

func (t *BaseTimer) SetTimer(id int, interval time.Duration, repeat int) {

	if !t.start {
		t.timers = make(map[int]*Timer)
		go t.Ticking()
		t.start = true
	}

	go func() {
		t.mux.Lock()
		t.timers[id] = &Timer{
			id:       id,
			interval: int64(interval),
			pass:     int64(interval),
			repeat:   repeat,
		}
		t.mux.Unlock()
	}()
}
