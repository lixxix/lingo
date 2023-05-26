package utils

import "github.com/lixxix/lingo/message"

type IConn interface {
	SetId(id uint32)
	GetId() uint32
	Send(target uint32, option int16, data []byte)
	Close()
}

type IConnection interface {
	IConn
	GetServer() uint32
	HoldBack(server uint32, id uint32)
}

// 客户端的连接
type IGateLink interface {
	SetId(id uint32)
	GetId() uint32
	Close()
	Send(target uint32, data []byte)
	GetServerId(sid uint32) uint32
	SetServerId(sid uint32, id uint32)
	RemoteAddr() string
	GetRecv() int64 //用于检测
	ClearRecv()
	// 位置
	SetLocation(server uint32, link IServerConn) // 设置位置--相当于游戏的连接
	GetLocation(server uint32) IServerConn
	LeaveLocation(conn IServerConn) bool
	AllLocation() []IServerConn
}

// 服务器管理客户连接
type IServerLink interface {
	IConnection
	SetServer(server uint32)
	GetServer() uint32
	Ready() bool
	RemoteAddr() string
	SetIpPort(ip uint32, port uint16)
	GetRegister() *message.TranServer
}

// 连接的服务器
type IServerConn interface {
	IConn
	SetServer(server uint32)
	GetServer() uint32
	GetLinkServer() uint32
	GetLinkID() uint32
	Connect(addr string) error
	ReConnect()
}

// server-client <- server-server
type IServerSink interface {
	OnServerMessage(target uint32, option uint16, msgid uint16, data []byte, server IServerConn)
	OnServerDisconect(server IServerConn)
	OnServerConnected(server IServerConn)
}

type IClientProcesser interface {
	OnClientMessage(server uint16, msgid uint16, data []byte, client IGateLink)
	OnClientDisconect(client IGateLink)
	OnClientConnected(client IGateLink)
	Register(client IGateLink)
	UnRegister(client IGateLink)
}

// 服务器管理客户连接
type IClientManager interface {
	Register(client IConnection)
	UnRegister(client IConnection)
	OnClientMessage(session uint32, option uint16, msgid uint16, data []byte, client IConnection)
}

type IServerManager interface {
	Register(client IServerLink)
	UnRegister(client IServerLink)
	OnClientMessage(session uint32, option uint16, msgid uint16, data []byte, client IServerLink)
}
