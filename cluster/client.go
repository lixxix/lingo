package cluster

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/utils"
)

type TcpClient struct {
	Id        uint32
	Server    uint32
	conn      net.Conn
	buffer    *utils.Buffer
	recv      bool
	processer IProcesser
}

func (c *TcpClient) SetId(id uint32)         { c.Id = id }
func (c *TcpClient) GetId() uint32           { return c.Id }
func (c *TcpClient) SetServer(server uint32) { c.Server = server }
func (c *TcpClient) GetServer() uint32       { return c.Server }

func (c *TcpClient) Send(target uint32, data []byte) {
	if c.conn != nil {
		c.conn.Write(c.buffer.PackGateData(target, data))
	}
}

func (c *TcpClient) HoldBack(server uint32, id uint32) {
	if c.conn != nil {
		c.conn.Write(c.buffer.PackServerData(server, id))
	}
}

func (c *TcpClient) Reading() {
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
			fmt.Println("read error:", err.Error())
			return
		}

		for index >= len(buff) {
			buff = append(buff, make([]byte, utils.BUFFER_SIZE)...)
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
				target, data, err := c.buffer.PopGateData()
				if err != nil {
					fmt.Println("package error:", err.Error())
					return
				}
				if c.processer != nil {
					c.processer.OnClientMessage(target, data, c)
				}
			}
		}

	}
}

func CreateClient(conn net.Conn, procsser IProcesser) *TcpClient {
	return &TcpClient{
		conn:      conn,
		Id:        0,
		buffer:    utils.CreateBuffer(),
		processer: procsser,
	}
}
