package cluster

type IConnection interface {
	SetId(id uint32)
	GetId() uint32
	Send(target uint32, data []byte)
	HoldBack(server uint32, id uint32)
}

type IProcesser interface {
	OnClientMessage(target uint32, data []byte, client IConnection)
	Register(client IConnection)
	UnRegister(client IConnection)
}
