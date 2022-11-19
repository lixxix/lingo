package gate

import (
	"errors"
	"fmt"

	"github.com/lixxix/lingo/utils"
)

type Server struct {
	Instances map[uint32]utils.IServerConn
}

func (s *Server) GetServerGroup(id uint32) utils.IServerConn {
	if _, ok := s.Instances[id]; ok {
		return s.Instances[id]
	}
	return nil
}

func (s *Server) GetInstance() utils.IServerConn {
	for _, instance := range s.Instances {
		return instance
	}
	return nil
}

func (s *Server) PushServer(server utils.IServerConn) error {
	if _, ok := s.Instances[server.GetId()]; ok {
		return errors.New("server already exists")
	}
	s.Instances[server.GetId()] = server
	return nil
}

func (s *Server) RemoveServer(server utils.IServerConn) error {
	if _, ok := s.Instances[server.GetId()]; ok {
		delete(s.Instances, server.GetId())
		return nil
	}
	return errors.New("remove server errors")
}

type ServerGroup struct {
	Servers map[uint32]*Server
}

func (s *ServerGroup) PushServer(server utils.IServerConn) error {
	if _, ok := s.Servers[server.GetServer()]; ok {
		return s.Servers[server.GetServer()].PushServer(server)
	}

	s.Servers[server.GetServer()] = &Server{
		Instances: make(map[uint32]utils.IServerConn),
	}

	return s.Servers[server.GetServer()].PushServer(server)
}

func (s *ServerGroup) RemoveServer(server utils.IServerConn) error {
	if _, ok := s.Servers[server.GetServer()]; ok {
		return s.Servers[server.GetServer()].RemoveServer(server)
	}
	return errors.New("not exits server : remove error")
}

func (s *ServerGroup) GetServer(sId uint32) utils.IServerConn {
	if _, ok := s.Servers[sId]; ok {
		return s.Servers[sId].GetInstance()
	}
	return nil
}

func (s *ServerGroup) GetServerById(sId uint32, id uint32) utils.IServerConn {
	if _, ok := s.Servers[sId]; ok {
		return s.Servers[sId].GetServerGroup(id)
	}
	return nil
}

func (s *ServerGroup) SendServer(sId uint32, client utils.IGateConn, data []byte) error {
	id := client.GetServerId(sId)
	ser := s.GetServerById(sId, id)
	if ser != nil {
		ser.Send(client.GetId(), data)
		return nil
	} else {
		ser = s.GetServer(sId)
		if ser != nil {
			client.SetServerId(sId, ser.GetId())
			ser.Send(client.GetId(), data)
			return nil
		}
	}
	return fmt.Errorf("no server found : send error server:%d", sId)
}
