package center

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/utils"
)

type ClientLink struct {
	Server    uint32
	Id        uint32
	Ip        int64
	port      uint16
	conn      net.Conn // connection
	recv      bool
	buffer    *utils.Buffer
	processer IProcesser
}

func (c *ClientLink) SetId(id uint32)         { c.Id = id }
func (c *ClientLink) GetId() uint32           { return c.Id }
func (c *ClientLink) SetServer(server uint32) { c.Server = server }
func (c *ClientLink) GetServer() uint32       { return c.Server }
func (c *ClientLink) Send(target uint32, data []byte) {
	if c.conn != nil {
		c.conn.Write(c.buffer.PackGateData(target, data))
	}
}

func (c *ClientLink) HoldBack(server uint32, id uint32) {
	if c.conn != nil {
		c.conn.Write(c.buffer.PackServerData(server, id))
	}
}

func (c *ClientLink) RemoteAddr() string {
	return c.conn.LocalAddr().String()
}

func (c *ClientLink) Close() {
	c.conn.Close()
}

func (c *ClientLink) SetIpPort(ip int64, port uint16) {
	c.Ip = ip
	c.port = port
}

func (c *ClientLink) GetRegister() *message.TranServer {
	return &message.TranServer{
		Ip:     c.Ip,
		Port:   c.port,
		Server: c.Server,
		Id:     c.Id,
	}
}

func (c *ClientLink) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		if c.processer != nil {
			c.processer.UnRegister(c)
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
			buff = append(buff, make([]byte, utils.BUFFER_SIZE)...)
			sz, err = c.conn.Read(buff[index:])
			index += sz
			if err != nil {
				logger.LOG.Debug(fmt.Sprintf("read error:%s", err.Error()))
				return
			}
		}
		if !c.recv {
			c.recv = true
			if index == 4 {
				reader := bytes.NewReader(buff)
				binary.Read(reader, binary.BigEndian, &c.Server)
				if c.processer != nil {
					c.processer.Register(c)
				}
			} else {
				logger.LOG.Debug("Hold Failed!")
				return
			}
		} else {
			c.buffer.PushData(buff[0:index])
			for !c.buffer.Empty() {
				_, data, err := c.buffer.PopGateData()
				if err != nil {
					logger.LOG.Debug(fmt.Sprintf("package error:%s", err.Error()))
					return
				}
				if c.processer != nil {
					c.processer.OnClientMessage(data, c)
				}
			}
		}

	}
}

func CreateTcp(conn net.Conn, processer IProcesser) *ClientLink {
	cl := &ClientLink{
		conn:      conn,
		buffer:    utils.CreateBuffer(),
		processer: processer,
	}
	go cl.Reading()
	return cl
}
