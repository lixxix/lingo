package utils

// dbserver的回调
type IDBSink interface {
	OnQueryMessage(target uint32, msgid uint16, data []byte, client IConnection)
	OnDBConnected(client IConnection)
	OnDBDisconnected(client IConnection)
}

// clientdb连接的回调
type IDBConnSink interface {
	OnDBConnected(server IServerConn)
	OnDBDisconnected(server IServerConn)
	OnDBMessage(target uint32, msgid uint16, data []byte, server IServerConn) //是db给的数据
}
