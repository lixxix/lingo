package utils

import (
	"fmt"
	"net"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/packer"
)

type ServerLink struct {
	Server uint32
	Id     uint32
	Ip     uint32
	port   uint16
	conn   net.Conn // connection
	recv   bool
	ready  bool
	buffer *Buffer
	sink   IServerManager
}

func (c *ServerLink) SetId(id uint32)         { c.Id = id }
func (c *ServerLink) GetId() uint32           { return c.Id }
func (c *ServerLink) SetServer(server uint32) { c.Server = server }
func (c *ServerLink) GetServer() uint32       { return c.Server }
func (c *ServerLink) Send(target uint32, option int16, data []byte) {
	if c.conn != nil {
		c.conn.Write(PackData(target, option, data))
	}
}

func (c *ServerLink) HoldBack(server uint32, id uint32) {
	if c.conn != nil {
		holdback := &message.ServerHoldSuccess{
			LinkServer: int16(server),
			LinkId:     int16(c.Id),
			MyId:       int16(id),
		}
		c.conn.Write(packer.PackMessage(holdback))
	}
}

func (c *ServerLink) RemoteAddr() string {
	return c.conn.LocalAddr().String()
}

func (c *ServerLink) Close() {
	c.conn.Close()
}

func (c *ServerLink) SetIpPort(ip uint32, port uint16) {
	c.Ip = ip
	c.port = port
	c.ready = true
}

func (c *ServerLink) Ready() bool {
	return c.ready
}

func (c *ServerLink) GetRegister() *message.TranServer {
	return &message.TranServer{
		Ip:     c.Ip,
		Port:   c.port,
		Server: c.Server,
		Id:     c.Id,
	}
}

func (c *ServerLink) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		if c.sink != nil {
			c.sink.UnRegister(c)
		}
	}()

	buff := make([]byte, 4096)
	sz := 0

	for {
		index, err := c.conn.Read(buff)
		if err != nil {
			logger.LOG.Debug(fmt.Sprintf("read error:%s", err.Error()))
			return
		}

		for index >= len(buff) {
			buff = append(buff, make([]byte, BUFFER_SIZE)...)
			sz, err = c.conn.Read(buff[index:])
			index += sz
			if err != nil {
				logger.LOG.Debug(fmt.Sprintf("read error:%s", err.Error()))
				return
			}
		}
		if !c.recv {
			c.recv = true
			if index == 4 { //告知服务器的属性
				hold := &message.ServerHold{}
				packer.ParserData(buff, hold)
				c.Server = uint32(hold.Server)
				c.Id = uint32(hold.Id)
				fmt.Println("server link ", c.Server, c.Id)
				if c.sink != nil {
					c.sink.Register(c)
				}
			} else {
				logger.LOG.Debug(fmt.Sprintf("Hold Failed! %d", index))
				return
			}
		} else {
			c.buffer.PushData(buff[0:index])
			for !c.buffer.Empty() {
				sess, option, msgid, data, err := c.buffer.PopServerData()
				if err != nil {
					logger.LOG.Debug(fmt.Sprintf("package error:%s", err.Error()))
					return
				}
				if c.sink != nil {
					c.sink.OnClientMessage(sess, option, msgid, data, c)
				}
			}
		}

	}
}

func CreateServerLink(conn net.Conn, sink IServerManager) *ServerLink {
	cl := &ServerLink{
		conn:   conn,
		buffer: CreateBuffer(),
		sink:   sink,
	}
	go cl.Reading()
	return cl
}
