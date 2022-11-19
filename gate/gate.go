package gate

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/utils"
)

type Gate struct {
	ID        uint32
	ServerID  uint32
	clients   map[utils.IGateConn]bool
	idClients map[uint32]utils.IGateConn
	Group     *ServerGroup
	reg_mutex sync.Mutex
	/*
		提供其他后端服务器进行转发 所以前端的数据打包需要新增数据包。解析包的时候需要解析多个位数，
		2 byte size 4 byte target  -- 数据通过client进行发送， 然后转发到后端的数据。
		后端处理完数据之后，将 target 替换成client的id，进行转发。
		EP: sz+server1(clientid)+data  => server1 ... server1 = sz+clientid+data => gate  ... gate => client
	*/
}

func (g *Gate) GetClient(clientid uint32) utils.IGateConn {
	if _, ok := g.idClients[clientid]; ok {
		return g.idClients[clientid]
	}
	return nil
}

func (g *Gate) Register(client utils.IGateConn) {
	g.reg_mutex.Lock()
	g.ID++
	g.clients[client] = true
	client.SetId(g.ID*0xFF + g.ServerID)
	g.idClients[client.GetId()] = client
	g.reg_mutex.Unlock()
}

func (g *Gate) UnRegister(client utils.IGateConn) {
	g.reg_mutex.Lock()
	delete(g.clients, client)
	delete(g.idClients, client.GetId())
	g.reg_mutex.Unlock()
}

// 先连接中心服务器
func (g *Gate) Start() {
	conn := utils.CreateClientConnection(utils.GateServer, g)
	conn.Connnect("127.0.0.1:8787")
}

func (g *Gate) ServeTcp(port uint16) {
	address := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		logger.LOG.Error(fmt.Sprintf("listen failed :%s", err.Error()))
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.LOG.Error(fmt.Sprintf("accept failed: %s", err.Error()))
			return
		}
		g.CreateTcp(conn)
	}
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (g *Gate) BindWS(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	g.CreateWS(conn)
}

func (g *Gate) CreateWS(conn *websocket.Conn) {
	c := CreateWS(conn, g)
	if c != nil {
		g.Register(c)
	}
}

func (g *Gate) ServeWS(port uint16) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		g.BindWS(w, r)
	})
	address := fmt.Sprintf(":%d", port)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (g *Gate) OnClientConnected(client utils.IGateConn) {
	logger.LOG.Debug(fmt.Sprintf("client linked %v", client))
}

func (g *Gate) OnClientDisconect(client utils.IGateConn) {
	logger.LOG.Debug(fmt.Sprintf("disconnected %v", client))
}

func (g *Gate) OnClientMessage(target uint32, data []byte, client utils.IGateConn) {
	// 判断并转换data的数据，
	logger.LOG.Debug(fmt.Sprintf("客户端信息 %v", data))
	err := g.Group.SendServer(target, client, data)
	if err != nil {
		fmt.Println(err.Error())
		client.Send(0, data)
	}
	// 发送并转发给后端
}

// 转发层的数据
func (g *Gate) OnTranMessage(target uint32, data []byte, client *TranConn) {
	logger.LOG.Debug(fmt.Sprintf("tran data %d ,%d, %v", target, data, client.GetServer()))
	c := g.GetClient(target)
	if c != nil {
		c.Send(client.GetServer(), data)
	}
}

func (g *Gate) OnTranDisconect(client *TranConn) {
	err := g.Group.RemoveServer(client)
	if err != nil {
		logger.LOG.Error(err.Error())
	}
}
func (g *Gate) OnTranConnected(client *TranConn) {
	logger.LOG.Debug(fmt.Sprintf("Gate Link : %d, %d ", client.GetServer(), client.GetId()))
	err := g.Group.PushServer(client)
	if err != nil {
		logger.LOG.Error(err.Error())
	}
}

func (g *Gate) OnServerMessage(target uint32, data []byte, client utils.IServerConn) {
	logger.LOG.Debug(fmt.Sprintf("server message:%d , %d ", target, data))
	main, sub, buf := message.ParserPackage(data)
	if main == message.M_Center {
		switch sub {
		case message.S_Register:
			reg := &message.TranServer{}
			reg.Parser(buf)
			if reg.Port == 0 || reg.Ip == 0 {
				return
			}
			ser_ip := fmt.Sprintf("%s:%d", utils.InetItoa(reg.Ip), reg.Port)
			logger.LOG.Debug(fmt.Sprintf("连接其他服务器:%d,%v", g.ServerID, ser_ip))
			tran := CreateTranConn(g)
			tran.Connnect(ser_ip)
		}
	}

}

func (g *Gate) OnServerDisconect(server utils.IServerConn) {
	go func() {
		time.Sleep(time.Second * 2)
		logger.LOG.Info("Relink center")
		server.ReConnect()
	}()
}

func (g *Gate) OnServerConnected(server utils.IServerConn) {
	if server.GetServer() == utils.CenterServer {
		g.ServerID = server.GetId()
	}
}

func (g *Gate) CreateTcp(conn net.Conn) {
	c := CreateTcp(conn, g)
	if c != nil {
		g.Register(c)
	}
}

func CreateGate() *Gate {
	return &Gate{
		clients:   make(map[utils.IGateConn]bool),
		idClients: make(map[uint32]utils.IGateConn),
		Group: &ServerGroup{
			Servers: make(map[uint32]*Server),
		},
	}
}
