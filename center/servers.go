package center

import (
	"fmt"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/utils"
)

/*
用于管理和分配服务器连接
*/
type Servers struct {
	servers map[uint32]*Server
}

func (s *Servers) SendServer(server uint32, data []byte) {
	sers := s.GetServer(server)
	if sers != nil {
		sers.Send(data)
	}
}

func (s *Servers) GetServer(sId uint32) *Server {
	if _, ok := s.servers[sId]; ok {
		return s.servers[sId]
	}
	s.servers[sId] = CreateServer()
	return s.servers[sId]
}

func (s *Servers) PushServer(client utils.IServerLink) uint32 {
	server := s.GetServer(client.GetServer())
	if client.GetId() == 0 {
		return server.PushServer(client)
	} else {
		// 获取是否存在
		con := server.GetServer(client.GetId())
		if con != nil {
			logger.LOG.Info(fmt.Sprintf("push server  %d", client.GetId()))
			return 0
		} else {
			server.PushServer(client)
		}
		return client.GetId()
	}
}

func (s *Servers) GetOtherServer() []*message.TranServer {
	regs := make([]*message.TranServer, 0)
	for id, server := range s.servers {
		if id > utils.GateServer {
			regs = append(regs, server.GetRegister()...)
		}
	}
	return regs
}

func (s *Servers) RemoveServer(client utils.IServerLink) bool {
	ser := s.GetServer(client.GetServer())
	if ser != nil {
		return ser.RemoveServer(client.GetId())
	}
	return false
}

type Server struct {
	ID    uint32
	Conns map[uint32]utils.IServerLink
}

func (s *Server) GetRegister() []*message.TranServer {
	reg := make([]*message.TranServer, 0)
	for _, ser := range s.Conns {
		if ser.Ready() {
			reg = append(reg, ser.GetRegister())
		}
	}
	return reg
}

// 利用map的随机性进行返回所需要的连接地址
func (s *Server) GetRandRegister() *message.TranServer {
	for _, v := range s.Conns {
		return v.GetRegister()
	}
	return nil
}

func (s *Server) Send(data []byte) {
	for _, v := range s.Conns {
		v.Send(0, utils.SERVER, data)
	}
}

func (s *Server) GetServer(id uint32) utils.IServerLink {
	if _, ok := s.Conns[id]; ok {
		return s.Conns[id]
	}
	return nil
}

func (s *Server) RemoveServer(id uint32) bool {
	if s.GetServer(id) != nil {
		delete(s.Conns, id)
		return true
	}
	return false
}

// 加入控制并且返回ID
func (s *Server) PushServer(conn utils.IServerLink) uint32 {
	if conn.GetId() == 0 {
		s.ID++
		s.ID %= 256
		s.Conns[s.ID] = conn
		return s.ID
	} else {
		s.Conns[conn.GetId()] = conn
		return conn.GetId()
	}
}

func CreateServers() *Servers {
	return &Servers{
		servers: make(map[uint32]*Server),
	}
}

func CreateServer() *Server {
	return &Server{
		ID:    0,
		Conns: make(map[uint32]utils.IServerLink),
	}
}
