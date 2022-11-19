package main

import (
	"time"

	"github.com/lixxix/lingo/center"
	"github.com/lixxix/lingo/gate"
	"github.com/lixxix/lingo/logger"
)

func main() {
	center := center.CreateCenter()
	go center.Serve(8787)
	time.Sleep(time.Second)

	// for i := 0; i < 100; i++ {
	// cls := cluster.CreateCluster(20)
	// cls.Start("127.0.0.1:8787")
	// go cls.Serve()
	// }

	// cls = cluster.CreateCluster(20)
	// cls.Start("127.0.0.1:8787")
	// go cls.Serve()

	// cls = cluster.CreateCluster(20)
	// cls.Start("127.0.0.1:8787")
	// go cls.Serve()

	// cls = cluster.CreateCluster(20)
	// cls.Start("127.0.0.1:8787")
	// go cls.Serve()

	time.Sleep(time.Second)
	gt := gate.CreateGate()
	gt.Start()
	go gt.ServeWS(6888)
	gt.ServeTcp(6565)
	defer logger.LOG.Sync()
}
