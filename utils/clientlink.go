package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/lixxix/lingo/logger"
)

type ClientLink struct {
	Id      uint32
	Server  uint32
	conn    net.Conn
	bufChan chan []byte
	Recv    atomic.Int64
	buffer  *Buffer
	recv    bool
	sink    IClientManager
}

func (c *ClientLink) SetId(id uint32)         { c.Id = id }
func (c *ClientLink) GetId() uint32           { return c.Id }
func (c *ClientLink) SetServer(server uint32) { c.Server = server }
func (c *ClientLink) GetServer() uint32       { return c.Server }

func (c *ClientLink) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *ClientLink) Send(target uint32, option int16, data []byte) {
	if c.conn != nil {
		if _, err := c.conn.Write(PackData(target, option, data)); err != nil {
			fmt.Println(err.Error())
		}
	}
}

func (c *ClientLink) HoldBack(server uint32, id uint32) {
	if c.conn != nil {
		c.conn.Write(PackServerData(server, id))
	}
}

func (c *ClientLink) Processing() {
	for buf := range c.bufChan {
		c.buffer.PushData(buf)
		if len(c.bufChan) == 0 {
			pks, err := c.buffer.PopServerGroup()
			if err != nil {
				fmt.Println("package error:", err.Error())
				return
			}
			c.Recv.Add(int64(len(pks)))
			if c.sink != nil {
				for _, pk := range pks {
					c.sink.OnClientMessage(pk.Session, pk.Option, pk.MsgId, pk.Buff, c)
				}
			}
		}
	}
}

func (c *ClientLink) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		c.recv = false
		c.conn = nil
		close(c.bufChan)
		if c.sink != nil {
			c.sink.UnRegister(c)
		}
	}()

	buff := make([]byte, 4096)
	sz := 0

	for {
		index, err := c.conn.Read(buff)
		if err != nil {
			fmt.Println("read error:", err.Error())
			return
		}

		for index >= len(buff) {
			buff = append(buff, make([]byte, BUFFER_SIZE)...)
			sz, err = c.conn.Read(buff[index:])
			index += sz
			if err != nil {
				fmt.Println("read error:", err.Error())
				return
			}
		}

		if !c.recv {
			logger.LOG.Debug("cluster hold back")
			c.recv = true
			if index == 8 {
				reader := bytes.NewReader(buff)
				binary.Read(reader, binary.BigEndian, &c.Server)
				binary.Read(reader, binary.BigEndian, &c.Id)
				if c.sink != nil {
					c.sink.Register(c)
				}
			} else {
				logger.LOG.Debug("Hold Failed!")
				return
			}
		} else {
			// fmt.Println("zhuanfa", index, buff[0:index])
			select {
			case c.bufChan <- CopyBytes(buff[0:index]):
			default:
				fmt.Println("full")
				continue
			}
		}
	}
}

func CreateClientLink(conn net.Conn, sink IClientManager) *ClientLink {
	return &ClientLink{
		conn:    conn,
		Id:      0,
		bufChan: make(chan []byte, 2048),
		buffer:  CreateBuffer(),
		sink:    sink,
	}
}
