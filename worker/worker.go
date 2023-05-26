package worker

import (
	"fmt"
	"sync"
	"time"

	"github.com/lixxix/lingo/logger"
)

// 创建工人， 可以共同处理异步事件，并调用回调方法！ -- 可以创建一些异步方法
/*
	Process 返回的数据作为callback的回调数据
*/
type WorkData struct {
	Data     interface{}
	Process  func() (interface{}, error)
	CallBack func(data interface{}, err error)
}

var mutex sync.Mutex

var MutexMain *sync.Mutex = nil

var WorkerMissions *Queue = &Queue{}

type Worker struct {
	ID uint32
}

func (w *Worker) Run() {
	for {
		mutex.Lock()
		top := WorkerMissions.Get()
		mutex.Unlock()
		if top == nil {
			time.Sleep(time.Millisecond * 100)
		} else {
			data := top.(*WorkData)
			logger.LOG.Debug(fmt.Sprintf("Worker Id :%d start!", w.ID))
			rt, err := data.Process()
			logger.LOG.Debug(fmt.Sprintf("Worker Id :%d finish!", w.ID))
			if MutexMain != nil {
				MutexMain.Lock()
				data.CallBack(rt, err)
				MutexMain.Unlock()
			} else {
				data.CallBack(rt, err)
			}
		}
	}
}

// 设置主线程的锁
func SetMainMutex(mu *sync.Mutex) {
	MutexMain = mu
}

func AddWork(Process func() (interface{}, error), CallBack func(data interface{}, err error)) {
	mutex.Lock()
	WorkerMissions.Put(&WorkData{
		Process:  Process,
		CallBack: CallBack,
	})
	mutex.Unlock()
}

func InitWorker(worker int) {
	for i := 0; i < worker; i++ {
		wk := &Worker{
			ID: uint32(i + 1),
		}
		go wk.Run()
	}
}
