package center

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/packer"
	"github.com/lixxix/lingo/utils"
)

type ICenterSink interface {
	ClientMessage(msgid uint16, data []byte, client utils.IServerLink) //回调 -- 不设置session，主要是客户端没有这个权限
	ServerUnRegister(client utils.IServerLink)                         //断开链接的时候
}

/*
中心服务：
主要用于管理所有服务器设备，并分配服务器设备数据。整体调度服务器的使用
*/
type Center struct {
	// 接受的tcp连接器
	clients   map[utils.IServerLink]bool
	servers   *Servers
	sink      ICenterSink
	reg_mutex sync.Mutex
	Mutex     sync.Mutex
}

func (c *Center) SetSink(sink ICenterSink) {
	c.sink = sink
}

func (c *Center) OnClientMessage(sess uint32, option uint16, msgid uint16, data []byte, client utils.IServerLink) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if option == uint16(utils.SERVER) {
		switch msgid {
		case message.S_Register:
			reg := message.ServerRegister{}
			packer.ParserData(data, &reg)
			fmt.Println(reg, utils.InetItoa(reg.Ip))
			// 这个服务器的连接地址是
			add := strings.Split(client.RemoteAddr(), ":")[0] + ":" + strconv.Itoa(int(reg.Port))
			logger.LOG.Debug(fmt.Sprintf("link addr :%s %s", add, utils.InetItoa(reg.Ip)))
			client.SetIpPort(reg.Ip, reg.Port)
			serv := c.servers.GetServer(client.GetServer())
			if serv != nil {
				cc := serv.GetServer(client.GetId())
				if cc == client {
					logger.LOG.Debug("OK Right Server Register")
				} else {
					logger.LOG.Error("Wront Server ")
				}
			}

			if client.GetServer() > 2 {
				c.servers.SendServer(utils.GateServer, message.PackPackage(msgid, packer.PackMessage(client.GetRegister())))
			}
		case message.C_GetServer:
			query := message.GetServer{}
			// query.Parser(b)
			packer.ParserData(data, &query)
			sers := c.servers.GetServer(query.Server)
			if sers != nil {
				v := sers.GetRandRegister()
				if v != nil {
					client.Send(0, utils.SERVER, message.PackPackage(message.S_GetServer, packer.PackMessage(v)))
				}
			}
		case message.C_QueryServer:
			query := message.QueryServer{}
			packer.ParserData(data, &query)
			ser := c.servers.GetServer(uint32(query.Server))
			if ser != nil {
				fmt.Println(ser, "服务器连接数据")
				servers := ser.GetRandRegister()
				if servers != nil {
					client.Send(0, utils.SERVER, message.PackPackage(message.S_QueryServer, packer.PackMessage(servers)))
				} else {
					client.Send(0, utils.SERVER, message.PackPackage(message.S_QueryServer, packer.PackMessage(servers)))
				}
			} else {
				fmt.Println("并由启动", query.Server)
			}
		}
	} else {
		fmt.Println("客户都安消息", msgid, data, client.GetServer(), client.GetId())
		if c.sink != nil {
			c.sink.ClientMessage(msgid, data, client)
		}
		// type Message struct {
		// 	Name string
		// 	Age  int32
		// }
		// msg := &Message{}
		// packer.ParserData(data, msg)
		// // s, buf := message.ParserPackage(data)
	}
}

func (c *Center) Register(client utils.IServerLink) {
	c.reg_mutex.Lock()
	fmt.Println("Register server : ", client)
	sId := c.servers.PushServer(client)
	client.SetId(sId)
	logger.LOG.Info(fmt.Sprintf("register : %d id：%d ", client.GetServer(), client.GetId()))
	client.HoldBack(utils.CenterServer, sId)
	c.clients[client] = true
	time.Sleep(time.Microsecond)
	if client.GetServer() == utils.GateServer {
		regs := c.servers.GetOtherServer()
		for _, v := range regs {
			client.Send(0, utils.SERVER, message.PackPackage(1, packer.PackMessage(v)))
		}
	}
	c.reg_mutex.Unlock()
}

func (c *Center) UnRegister(client utils.IServerLink) {
	c.reg_mutex.Lock()
	delete(c.clients, client)
	fmt.Println("UnRegister server :", client)
	if c.sink != nil {
		c.sink.ServerUnRegister(client)
	}
	if !c.servers.RemoveServer(client) {
		logger.LOG.Warn(fmt.Sprintf("remove sever failed %d,%d", client.GetServer(), client.GetId()))
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
	utils.CreateServerLink(conn, c)
}

func CreateCenter() *Center {
	return &Center{
		clients: make(map[utils.IServerLink]bool),
		servers: CreateServers(),
	}
}
