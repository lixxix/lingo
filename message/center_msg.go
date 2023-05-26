package message

const (
	M_Center int16 = 0
)

const (
	S_Register    uint16 = 1 //注册数据
	C_GetServer   uint16 = 2 //获取服务器连接   --- 数组反馈
	S_GetServer   uint16 = 3 //返回给服务器的数据
	C_QueryServer uint16 = 4 //请求连接服务器数据
	S_QueryServer uint16 = 5 //返回
)

type ServerHold struct {
	Server int16
	Id     int16
}

type ServerHoldSuccess struct {
	LinkServer int16
	LinkId     int16 //连接着的id
	MyId       int16 //我的id
}

type ServerRegister struct {
	Port uint16
	Ip   uint32
}

// 中转的信息
type TranServer struct {
	Ip     uint32
	Port   uint16
	Server uint32
	Id     uint32
}

type GetServer struct {
	Server uint32
}

type QueryServer struct {
	Server uint16
}

// 监听的数据
type ListenServer struct {
	Servers []uint32
}

func (l *ListenServer) PushServer(server uint32) {
	for _, v := range l.Servers {
		if v == server {
			return
		}
	}
	l.Servers = append(l.Servers, server)
}

func (l *ListenServer) RemoveServer(server uint32) {
	for i, v := range l.Servers {
		if v == server {
			l.Servers = append(l.Servers[0:i], l.Servers[i+1:]...)
			return
		}
	}
}
