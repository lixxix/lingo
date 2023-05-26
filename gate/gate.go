package gate

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/packer"
	"github.com/lixxix/lingo/utils"
	"nhooyr.io/websocket"
)

type IGateSink interface {
	ClientProcess(client utils.IGateLink) //处理客户端返送消息
	// ServerProcess(msgid uint16, data []byte) //处理返回消息
	ClientConnected(client utils.IGateLink) bool //连接的ip和id
	ClientRegister(client utils.IGateLink)       //客户端注册好了
	ClientDisconect(client utils.IGateLink)      //关闭连接
}

type Gate struct {
	sync.Mutex //
	ID         uint32
	ServerID   uint32
	sink       IGateSink
	mapClients sync.Map
	Group      *ServerGroup //zuhe

	LockTranMutex sync.Mutex
	// ws
	serveMux http.ServeMux
	/*
		提供其他后端服务器进行转发 所以前端的数据打包需要新增数据包。解析包的时候需要解析多个位数，
		2 byte size 4 byte target  -- 数据通过client进行发送， 然后转发到后端的数据。
		后端处理完数据之后，将 target 替换成client的id，进行转发。
		EP: sz+server1(clientid)+data  => server1 ... server1 = sz+clientid+data => gate  ... gate => client
	*/
}

func (g *Gate) GetServer() uint32 { return utils.GateServer }
func (g *Gate) GetId() uint32     { return g.ServerID }

func (g *Gate) GetClient(clientid uint32) utils.IGateLink {
	cli, exists := g.mapClients.Load(clientid)
	if !exists {
		return nil
	}
	return cli.(utils.IGateLink)
}

func (g *Gate) SetSink(sink IGateSink) {
	g.sink = sink
}

func (g *Gate) Register(client utils.IGateLink) {
	atomic.AddUint32(&g.ID, 1)
	client.SetId(g.ID*0x100 + g.ServerID)
	g.mapClients.Store(client.GetId(), client)
}

func (g *Gate) UnRegister(client utils.IGateLink) {
	if g.sink != nil {
		g.sink.ClientDisconect(client)
	} else {
		g.RemoveClientLink(client)
	}
}

func (g *Gate) RemoveClientLink(client utils.IGateLink) {
	g.mapClients.Delete(client.GetId()) //如果没有绑定回调，可能存在异常短线的情况
	lks := client.AllLocation()
	for _, s := range lks {
		if s != nil {
			s.Send(client.GetId(), utils.SERVER, message.PackPackage(message.ClientBreak, []byte{}))
		}
	}
}

// 先连接中心服务器
func (g *Gate) Start(ip string) {
	conn := utils.CreateClientConnection(utils.GateServer, 0, g)
	if nil != conn.Connect(ip) {
		conn.ReConnect()
	}
}

// 告知cluster，客户端head绑定
func (g *Gate) SendClientLink(server uint32, client utils.IGateLink) {
	sv := g.Group.GetServer(server)
	if sv != nil {
		sv.Send(client.GetId(), utils.SERVER, utils.PackMessage(message.ClientLink, []byte{}))
	}
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

		// 尚未分配到id
		if g.ServerID == 0 {
			conn.Close()
			return
		}

		client := g.CreateTcp(conn)
		if g.sink != nil {
			if g.sink.ClientConnected(client) {
				continue
			}
		}
		if client != nil {
			go client.Reading()
			g.Register(client)
		}
	}
}

func (g *Gate) CreateWS(conn *websocket.Conn, ctx context.Context) *WSConn {
	c := CreateWS(conn, ctx, g)
	if c != nil {
		g.Register(c)
	}
	return c
}

func (g *Gate) ServeWS(port uint16) {
	g.serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			logger.LOG.Info(err.Error())
			return
		}

		// 尚未分配到id
		if g.ServerID == 0 {
			c.Close(websocket.StatusAbnormalClosure, "client not ready")
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		ws := g.CreateWS(c, r.Context())
		if err == nil {
			ws.IP = ip
		}
		if g.sink != nil {
			if g.sink.ClientConnected(ws) {
				return
			}
		}
		go ws.Writing()
		ws.Reading()
	})
	address := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err.Error())
	}
	s := &http.Server{
		Handler:      &g.serveMux,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	s.Serve(ln)
}

func (g *Gate) OnClientConnected(client utils.IGateLink) {
	logger.LOG.Debug(fmt.Sprintf("client linked %v", client))
}

func (g *Gate) OnClientDisconect(client utils.IGateLink) {
	logger.LOG.Debug(fmt.Sprintf("disconnected %v", client))
}

func (g *Gate) OnClientMessage(server uint16, msgid uint16, data []byte, client utils.IGateLink) {
	// 判断并转换data的数据，
	// logger.LOG.Debug(fmt.Sprintf("客户端信息 %v", data))
	if g.sink != nil {
		g.sink.ClientProcess(client)
	}
	if msgid == 0 { //定义的心跳包
		return
	}
	// 进行发送转发， 包含了绑定上次的数据。size-
	g.LockTranMutex.Lock()
	ser := client.GetLocation(uint32(server))
	if ser == nil {
		get_ser := g.Group.GetServer(uint32(server))
		if get_ser != nil {
			ip_data := message.GateUserIP{
				Session: client.GetId(),
				Address: client.RemoteAddr(),
			}
			get_ser.Send(client.GetId(), utils.SERVER, message.PackPackage(message.ClientLink, packer.PackMessage(&ip_data)))
			get_ser.Send(client.GetId(), utils.CLIENT, data)
			client.SetLocation(uint32(server), get_ser)
		}
	} else {
		ser.Send(client.GetId(), 0, data)
	}
	g.LockTranMutex.Unlock()
	// 发送并转发给后端
}

func (g *Gate) TickInfo() {
	ticker := time.NewTicker(time.Millisecond * 500)
	for {
		<-ticker.C
		fmt.Println(Send, Recv, Recv-Send)
	}
}

// region gate_msg
func (g *Gate) GateMessage(session uint32, msgid uint16, data []byte) {
	switch msgid {
	case message.NetShutDown:
		client := g.GetClient(session)
		if client != nil {
			client.Close()
		}
	}
}

// endregion gate_msg

func (g *Gate) SendClient(session uint32, data []byte) {
	c := g.GetClient(session)
	if c != nil {
		c.Send(session, data)
	}
}

// 转发层的数据 -- 应该包含转发的数据，比如是给gate发还是给玩家发
func (g *Gate) OnTranMessage(session uint32, option uint16, msgid uint16, data []byte, client *TranConn) {
	// logger.LOG.Debug(fmt.Sprintf("tran data %d ,%d, %v", session, data, client.GetServer()))
	if option == uint16(utils.SERVER) {
		g.GateMessage(session, msgid, data)
	} else {
		g.SendClient(session, data)
	}
}

func (g *Gate) OnTranDisconect(tran_server *TranConn) {
	go func() { //只进行一次重新连接
		time.Sleep(time.Second * 2)
		tran_server.ReConnect()
	}()
	logger.LOG.Warn(fmt.Sprintf("Server disconnected: %v\n", tran_server))
	g.LockTranMutex.Lock()
	err := g.Group.RemoveServer(tran_server)
	if err != nil {
		logger.LOG.Error(err.Error())
	}
	// 将所有用户的连接进行更新
	g.mapClients.Range(func(key, value any) bool {
		lk, ok := value.(utils.IGateLink)
		if ok {
			lk.LeaveLocation(tran_server)
		}
		return true
	})
	g.LockTranMutex.Unlock()
}

func (g *Gate) OnTranConnected(client *TranConn) {
	logger.LOG.Debug(fmt.Sprintf("Gate Link : %d, %d\n", client.GetServer(), client.GetId()))
	err := g.Group.PushServer(client)
	if err != nil {
		logger.LOG.Error(err.Error())
	}
}

func (g *Gate) OnServerMessage(target uint32, option uint16, msgid uint16, data []byte, client utils.IServerConn) {
	logger.LOG.Debug(fmt.Sprintf("server message:%d , %d ", target, data))

	switch msgid {
	case message.S_Register:
		reg := &message.TranServer{}
		packer.ParserData(data, reg)
		if reg.Port == 0 || reg.Ip == 0 {
			fmt.Println("register server , ", reg, client)
			return
		}
		ser_ip := fmt.Sprintf("%s:%d", utils.InetItoa(reg.Ip), reg.Port)
		logger.LOG.Debug(fmt.Sprintf("连接其他服务器:%d,%v", g.ServerID, ser_ip))
		tran := CreateTranConn(g)
		if nil != tran.Connect(ser_ip) {
			tran.ReConnect()
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
	if server.GetLinkServer() == utils.CenterServer {
		g.ServerID = server.GetLinkID()
	}
	fmt.Println("link server connected", server)
}

func (g *Gate) CreateTcp(conn net.Conn) *TcpConn {
	return CreateTcp(conn, g)
}

func CreateGate() *Gate {
	return &Gate{
		mapClients: sync.Map{},
		Group: &ServerGroup{
			Servers: make(map[uint32]*Server),
		},
	}
}
