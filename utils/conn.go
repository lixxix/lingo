package utils

import "github.com/lixxix/lingo/message"

type IConn interface {
	SetId(id uint32)
	GetId() uint32
	Send(target uint32, data []byte)
	Close()
}
type IGateConn interface {
	IConn
	GetServerId(sid uint32) uint32
	SetServerId(sid uint32, id uint32)
}

//服务器管理客户连接
type IServerLink interface {
	IConn
	SetServer(server uint32)
	GetServer() uint32
	RemoteAddr() string
	SetIpPort(ip int64, port uint16)
	GetRegister() *message.TranServer
	HoldBack(server uint32, id uint32) //告知这个时什么服务器，并且返回对应的id
}

//连接的服务器
type IServerConn interface {
	IConn
	SetServer(server uint32)
	GetServer() uint32
	Connnect(addr string)
	ReConnect()
}

// server-client <- server-server
type IServerSink interface {
	OnServerMessage(target uint32, data []byte, server IServerConn)
	OnServerDisconect(server IServerConn)
	OnServerConnected(server IServerConn)
}

// server-server <- server-client
type IServerProcesser interface {
	OnServerMessage(target uint32, data []byte, client IServerConn)
	OnServerDisconect(client IServerConn)
	OnServerConnected(client IServerConn)
	Register(client IServerConn)
	UnRegister(client IServerConn)
}

type IConnProcesser interface {
	OnClientMessage(target uint32, data []byte, client IGateConn)
	OnClientDisconect(client IGateConn)
	OnClientConnected(client IGateConn)
	Register(client IGateConn)
	UnRegister(client IGateConn)
}
