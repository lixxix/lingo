package main

import (
	"flag"
	"fmt"

	_ "net/http/pprof"

	"github.com/lixxix/lingo/gate"
	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/utils"
)

var stype *string = flag.String("stype", "all", "server type ")

func main() {
	flag.Parse()
	fmt.Println("Starting", *stype)

	// worker 异步，主要是解决http请求的问题
	// worker.InitWorker(runtime.NumCPU())
	// for i := 0; i < 50; i++ {
	// 	worker.AddWork(func() (interface{}, error) {
	// 		time.Sleep(time.Second)
	// 		return nil, nil
	// 	}, func(b interface{}, err error) {
	// 		if err == nil {
	// 			fmt.Println("mutext end")
	// 		}
	// 		fmt.Println("processer")
	// 	})
	// }

	// center := center.CreateCenter()
	// go center.Serve(8787)

	// clust := cluster.CreateCluster(6)
	// clust.Start("127.0.0.1:8787")
	// go clust.Serve()

	gt := gate.CreateGate()
	gt.Start("127.0.0.1:8787")
	go gt.ServeWS(6888)
	go gt.ServeTcp(6565)
	// go gt.TickInfo()
	defer logger.LOG.Sync()

	// go func() {
	// 	// if true {
	// 	runtime.SetBlockProfileRate(1)     // 开启对阻塞操作的跟踪，block
	// 	runtime.SetMutexProfileFraction(1) // 开启对锁调用的跟踪，mutex
	// 	// }

	// 	//http://127.0.0.1:7001/debug/pprof/
	// 	// mybase.I("http debug port 7011")
	// 	err := http.ListenAndServe(":7011", nil)
	// 	if err != nil {
	// 		fmt.Println("ListenAndServe: ", err)
	// 	}
	// }()

	// good := make(chan bool, 3)
	// go func() {
	// 	defer func() {
	// 		fmt.Println("close chane")
	// 	}()
	// 	for c := range good {
	// 		fmt.Println(c)
	// 	}
	// }()
	// go func() {
	// 	defer func() {
	// 		fmt.Println("close chane11")
	// 		if err := recover(); err != nil {
	// 			fmt.Println(err)
	// 		}
	// 	}()
	// 	for {
	// 		select {
	// 		case good <- true:
	// 		default:
	// 			return
	// 		}
	// 	}
	// }()
	// time.Sleep(time.Second)

	// close(good)
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	// sig := <-c

	sig := utils.WaitSignal()
	fmt.Printf("sig:%v\n", sig)
}
