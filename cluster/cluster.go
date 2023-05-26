package cluster

import (
	"fmt"
	"net"
	"sync"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/packer"
	"github.com/lixxix/lingo/utils"
)

// 作为普通服务器，需要新增一个连接
type Cluster struct {
	Server     uint32
	ServerID   uint32
	Port       int
	Session    sync.Map
	mapClients sync.Map          //gate的连接 存储key link , value : id
	centerConn utils.IServerConn // 中心协调
	sink       IConnSink         //逻辑回调
	reg_mutex  sync.Mutex
	Mutex      sync.Mutex
}

type ServerUserSession struct {
	Conn utils.IConnection
	Addr string
}

func (s *ServerUserSession) Send(session uint32, option uint16, data []byte) {
	if s.Conn != nil {
		s.Conn.Send(session, int16(option), data)
	}
}

// func (s *Cluster) GetSessionClient(session uint32) utils.IGateLink {
// 	id := session % 0x100
// 	client, exists := s.mapClients.Load(id)
// 	if exists {
// 		return client.(utils.IGateLink)
// 	}
// 	return nil
// }

func (c *Cluster) SetSink(sink IConnSink) {
	c.sink = sink
}

func (clus *Cluster) Register(client utils.IConnection) {
	clus.reg_mutex.Lock()
	clus.mapClients.Store(client, client.GetId())
	client.HoldBack(clus.Server, clus.ServerID)
	clus.reg_mutex.Unlock()

	if clus.sink != nil {
		clus.sink.OnClientConnected(client)
	}
}

func (clus *Cluster) UnRegister(client utils.IConnection) {
	clus.reg_mutex.Lock()
	clus.mapClients.Delete(client)
	// 删除用户
	clus.Session.Range(func(key, value any) bool {
		if value == client {
			fmt.Println("删除连接 gate", key)
			clus.Session.Delete(key)
		}
		return true
	})
	clus.reg_mutex.Unlock()
	if clus.sink != nil {
		clus.sink.OnClientDisconect(client) //处理短线网关节点
	}
}

func (clus *Cluster) GetSessionClient(id uint32) *ServerUserSession {

	if con, exists := clus.Session.Load(id); exists {
		return con.(*ServerUserSession)
	}
	return nil
}

func (clus *Cluster) Start(ip string) {
	conn := utils.CreateClientConnection(clus.Server, 0, clus)
	if nil != conn.Connect(ip) {
		conn.ReConnect()
	}
}

func (clus *Cluster) Serve() {
	rand_port, err := utils.GetFreePort()
	if err != nil {
		logger.LOG.Error(err.Error())
		return
	}

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
	clus.centerConn.Send(clus.ServerID, utils.SERVER, buf)
}

func (clus *Cluster) ShutDownClient(session uint32) {
	client := clus.GetSessionClient(session)
	if client != nil {
		client.Conn.Send(session, utils.SERVER, message.PackPackage(message.NetShutDown, nil))
	} else {
		fmt.Println("no session ", session)
	}
}

func (clus *Cluster) ServeClient(conn net.Conn) {
	c := utils.CreateClientLink(conn, clus)
	go c.Reading()
	go c.Processing()
}

func (clus *Cluster) OnServerMessage(target uint32, option uint16, msgid uint16, data []byte, server utils.IServerConn) {
	// clus.Mutex.Lock()
	// defer clus.Mutex.Unlock()
	if option == uint16(utils.SERVER) {
		switch msgid {
		case message.S_GetServer:
			reg := message.TranServer{}
			packer.ParserData(data, &reg)
			ser_ip := fmt.Sprintf("%s:%d", utils.InetItoa(reg.Ip), reg.Port)
			logger.LOG.Info(fmt.Sprintf("Link %s for server :%v", ser_ip, reg))

			conn := utils.CreateClientConnection(clus.Server, clus.ServerID, clus)
			if nil != conn.Connect(ser_ip) {
				conn.ReConnect()
			}
		case message.S_QueryServer:
			if len(data) == 0 {
				fmt.Println("there is no server user query")
			} else {
				link_server := message.TranServer{}
				packer.ParserData(data, &link_server)
			}
		}
	} else {
		if clus.sink != nil {
			clus.sink.OnServerSinkMessage(target, msgid, data, server)
		}
	}
}

func (c *Cluster) OnServerDisconect(server utils.IServerConn) {
	if server.GetLinkServer() == utils.CenterServer {
		logger.LOG.Info("Relink center")
		server.ReConnect()
	} else {
		fmt.Println("server disconnected", server.GetLinkServer())
		// 去和中心请求下一个地址，并连接
		// c.LinkServer(server.GetLinkServer())
		//服务器断开，需要进行回调
		if c.sink != nil {
			c.sink.OnServerDisconnect(server)
		}
	}

}

func (clus *Cluster) OnServerConnected(server utils.IServerConn) {
	fmt.Println("cluster center connected", server.GetServer(), server.GetId())
	if server.GetLinkServer() == utils.CenterServer {
		// 将我的ID进行赋值，id数据由中心服务器下发完成 -- 并赋值
		clus.ServerID = server.GetLinkID()
		server.SetId(clus.ServerID)
		// 通知给服务器告知，自己的位置
		clus.centerConn = server
		// 发送注册
		reg := message.ServerRegister{
			Port: uint16(clus.Port),
			Ip:   utils.LocalIP(),
		}
		// 是底层使用的数据打包，PackPackage 主消息0
		server.Send(999999, utils.SERVER, message.PackPackage(message.S_Register, packer.PackMessage(&reg)))
		if clus.sink != nil {
			// 连接到中心服务器，这样就可以跟这个服务器进行通信  --可以通过这个进行调整
			clus.sink.OnCenterConnected(server)
		}
	} else {
		// 如果是其他服务器的哈， GetLinkServer()  GetLinkID() 是对应的其他服务器的数据
		clus.sink.OnServerConnected(server)
	}
}

// 进行连接服务器，可以不知道服务器数据信息
func (c *Cluster) LinkServer(server uint32) {
	if c.centerConn != nil {
		query := &message.GetServer{
			Server: server,
		}
		c.centerConn.Send(0, utils.SERVER, message.PackPackage(message.C_GetServer, packer.PackMessage(query)))
	} else {
		logger.LOG.Error("centerserver not connected")
	}
}

// 由Gate发过来的消息 op
func (clus *Cluster) OnClientMessage(session uint32, option uint16, msgid uint16, data []byte, client utils.IConnection) {
	if clus.sink != nil {
		if option == uint16(utils.SERVER) {
			switch msgid {
			case message.ClientLink:
				ip_data := message.GateUserIP{}
				packer.ParserData(data, &ip_data)
				fmt.Println("绑定连接", session, client, ip_data)
				clus.Session.Store(session, &ServerUserSession{
					Conn: client,
					Addr: ip_data.Address,
				})
				clus.sink.OnClientLink(session, client)
			case message.ClientBreak:
				clus.Session.Delete(session)
				fmt.Println("连接断开", session, client)
				clus.sink.OnClientShut(session, client)
			}
		} else {
			// clus.Mutex.Lock()
			clus.sink.OnClientMessage(session, msgid, data, client)
			// clus.Mutex.Unlock()
		}

	}
}

func CreateCluster(Sid uint32) *Cluster {
	return &Cluster{
		centerConn: nil,
		Server:     Sid,
	}
}
