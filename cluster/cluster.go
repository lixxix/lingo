package cluster

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/utils"
)

// 作为普通服务器，需要新增一个连接
type Cluster struct {
	Server     uint32
	ServerID   uint32
	Port       int
	clients    map[IConnection]bool // TODO:保存网关连接
	idClients  map[uint32]IConnection
	centerConn utils.IServerConn
	reg_mutex  sync.Mutex
}

func (clus *Cluster) Register(client IConnection) {
	clus.reg_mutex.Lock()
	clus.clients[client] = true
	clus.idClients[client.GetId()] = client
	client.HoldBack(clus.Server, clus.ServerID)
	clus.reg_mutex.Unlock()
}

func (clus *Cluster) UnRegister(client IConnection) {
	clus.reg_mutex.Lock()
	delete(clus.clients, client)
	delete(clus.idClients, client.GetId())
	clus.reg_mutex.Unlock()
}

func (clus *Cluster) Start(ip string) {
	conn := utils.CreateClientConnection(clus.Server, clus)
	conn.Connnect(ip)
}

func (clus *Cluster) Serve() {
	rand_port, err := utils.GetFreePort()
	if err != nil {
		logger.LOG.Error(err.Error())
		return
	}
	fmt.Println(rand_port)
	clus.Port = rand_port
	logger.LOG.Debug(fmt.Sprintf("Listen :%d", clus.Port))
	address := fmt.Sprintf(":%d", clus.Port)
	logger.LOG.Debug(address)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("listen failed :", err.Error())
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("accept failed:", err.Error())
			return
		}
		clus.ServeClient(conn)
	}
}

func (clus *Cluster) SendCenter(buf []byte) {

}

func (clus *Cluster) ServeClient(conn net.Conn) {
	c := CreateClient(conn, clus)
	go c.Reading()
}

func (clus *Cluster) OnServerMessage(target uint32, data []byte, server utils.IServerConn) {}
func (clus *Cluster) OnServerDisconect(server utils.IServerConn) {
	// go func
	go func() {
		time.Sleep(time.Second * 2)
		logger.LOG.Info("Relink center")
		server.ReConnect()
	}()
}
func (clus *Cluster) OnServerConnected(server utils.IServerConn) {
	fmt.Println("cluster center connected", server.GetServer(), server.GetId())
	if server.GetServer() == utils.CenterServer {
		// 将我的ID进行赋值，id数据由中心服务器下发完成
		clus.ServerID = server.GetId()
		// 通知给服务器告知，自己的位置
		clus.centerConn = server
		// 发送注册
		reg := message.ServerRegister{
			Port: uint16(clus.Port),
			Ip:   utils.IpToInt("127.0.0.1"),
		}
		server.Send(0, message.PackPackage(message.M_Center, message.S_Register, reg.Bytes()))
	}
}

// 由Gate发过来的消息
func (clus *Cluster) OnClientMessage(target uint32, data []byte, client IConnection) {
	fmt.Println("client message:", target, data)
	// client.Send()
	client.Send(target, data)
}

func CreateCluster(Sid uint32) *Cluster {
	return &Cluster{
		clients:    make(map[IConnection]bool),
		idClients:  make(map[uint32]IConnection),
		centerConn: nil,
		Server:     Sid,
	}
}
