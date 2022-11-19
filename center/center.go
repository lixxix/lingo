package center

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/utils"
)

type IProcesser interface {
	Register(client utils.IServerLink)
	UnRegister(client utils.IServerLink)
	OnClientMessage(data []byte, client utils.IServerLink)
}

/*
	package

中心服务：
主要用于管理所有服务器设备，并分配服务器设备数据。整体调度服务器的使用
*/
type Center struct {
	// 接受的tcp连接器
	clients   map[utils.IServerLink]bool
	servers   *Servers
	reg_mutex sync.Mutex
}

func (c *Center) OnClientMessage(data []byte, client utils.IServerLink) {
	m, s, b := message.ParserPackage(data)
	if m == message.M_Center {
		switch s {
		case message.S_Register:
			reg := message.ServerRegister{}
			reg.Parser(b)
			// 这个服务器的连接地址是
			add := strings.Split(client.RemoteAddr(), ":")[0] + ":" + strconv.Itoa(int(reg.Port))
			logger.LOG.Debug(fmt.Sprintf("link addr :%s", add))
			client.SetIpPort(reg.Ip, reg.Port)
			serv := c.servers.GetServer(client.GetServer())
			if serv != nil {
				cc := serv.GetServer(client.GetId())
				if cc == client {
					// 说明没有问题
					logger.LOG.Debug("OK Right Server Register")
				} else {
					logger.LOG.Debug("Wront Server ")
				}
			}
			c.servers.SendServer(utils.GateServer, message.PackPackage(m, s, client.GetRegister().Bytes()))
		}
	}
}

func (c *Center) Register(client utils.IServerLink) {
	c.reg_mutex.Lock()
	sId := c.servers.PushServer(client)
	client.SetId(sId)
	client.HoldBack(utils.CenterServer, sId)
	c.clients[client] = true
	if client.GetServer() == utils.GateServer {
		regs := c.servers.GetOtherServer()
		for _, v := range regs {
			client.Send(0, message.PackPackage(0, 1, v.Bytes()))
		}
	}
	c.reg_mutex.Unlock()
}

func (c *Center) UnRegister(client utils.IServerLink) {
	c.reg_mutex.Lock()
	delete(c.clients, client)
	if !c.servers.RemoveServer(client) {
		logger.LOG.Warn("remove sever failed")
	}
	c.reg_mutex.Unlock()
}

func (c *Center) Serve(port uint16) {
	address := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", address)
	if err != nil {
		logger.LOG.Warn(fmt.Sprintf("listen failed :%s", err.Error()))
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.LOG.Error(fmt.Sprintf("accept failed:%s", err.Error()))
			return
		}
		c.CreateTcp(conn)
	}
}

func (c *Center) CreateTcp(conn net.Conn) {
	CreateTcp(conn, c)
}

func CreateCenter() *Center {
	return &Center{
		clients: make(map[utils.IServerLink]bool),
		servers: CreateServers(),
	}
}
