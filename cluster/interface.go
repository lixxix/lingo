package cluster

import "github.com/lixxix/lingo/utils"

type IConnSink interface {
	OnClientMessage(target uint32, msgid uint16, data []byte, client utils.IConnection)
	OnClientConnected(client utils.IConnection)
	OnClientDisconect(client utils.IConnection)
	OnCenterConnected(server utils.IServerConn) //是server的连接
	OnServerConnected(server utils.IServerConn)
	OnServerDisconnect(server utils.IServerConn)
	OnServerSinkMessage(target uint32, msgid uint16, data []byte, server utils.IServerConn)
	OnClientLink(session uint32, client utils.IConnection)
	OnClientShut(session uint32, client utils.IConnection)
}
