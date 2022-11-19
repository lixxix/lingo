package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/lixxix/lingo/logger"
)

/*
	服务器之间传递消息使用
	Id  区分相同server类型的不同
	Server 服务器类型
	第一次连接发送本身的数据信息
	第一次返回数据包返回id信息
*/
//连接到center的类
type ConnectionServer struct {
	Id        uint32
	Server    uint32
	conn      net.Conn
	buffer    *Buffer
	address   string
	recv      bool //第一次接受
	processer IServerSink
}

func (c *ConnectionServer) SetId(id uint32)         { c.Id = id }
func (c *ConnectionServer) GetId() uint32           { return c.Id }
func (c *ConnectionServer) SetServer(server uint32) { c.Server = server }
func (c *ConnectionServer) GetServer() uint32       { return c.Server }
func (c *ConnectionServer) Send(target uint32, data []byte) {
	c.conn.Write(c.buffer.PackGateData(target, data))
}

func (c *ConnectionServer) Close() {
	if c.conn != nil {
		fmt.Println("主动关闭")
		c.conn.Close()
	}
}

// func ()

func (c *ConnectionServer) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		c.recv = false
		c.conn = nil
		if c.processer != nil {
			c.processer.OnServerDisconect(c)
		}
	}()
	buff := make([]byte, 4096)
	sz := 0
	for {
		index, err := c.conn.Read(buff)
		if err != nil {
			logger.LOG.Debug(fmt.Sprintf("Tcp Error %s Server:%d Len:%d\n", err.Error(), c.Server, len(buff)))
			return
		}

		for index >= len(buff) {
			buff = append(buff, make([]byte, 4096)...)
			sz, err = c.conn.Read(buff[index:])
			index += sz
			if err != nil {
				fmt.Println(fmt.Sprintf("Tcp Error 2 %s Server:%d Address:%s Len:%d", err.Error(), c.Server, c.conn.RemoteAddr().String(), len(buff)))
				return
			}
		}
		if !c.recv {
			logger.LOG.Debug("连接： hold back")
			c.recv = true
			if index == 8 {
				reader := bytes.NewReader(buff)
				binary.Read(reader, binary.BigEndian, &c.Server)
				binary.Read(reader, binary.BigEndian, &c.Id)
				if c.processer != nil {
					c.processer.OnServerConnected(c)
				}
			} else {
				logger.LOG.Debug("Hold Failed!")
				return
			}
		} else {
			c.buffer.PushData(buff[0:index])
			for !c.buffer.Empty() {
				target, buf, err := c.buffer.PopGateData()
				if err != nil {
					logger.LOG.Error(fmt.Sprintf("TcpTrans data error: %s", err.Error()))
					return
				} else if buf == nil {
					break
				}
				if c.processer != nil {
					c.processer.OnServerMessage(target, buf, c)
				}
			}
		}

	}
}

func (c *ConnectionServer) ReConnect() {
	if c.conn == nil {
		go func() {
			for c.conn == nil {
				time.Sleep(time.Second * 2)
				c.Connnect(c.address)
			}
		}()
	}
}

func (c *ConnectionServer) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *ConnectionServer) Connnect(address string) {
	if c.conn == nil {
		c.address = address
		raddr, err := net.ResolveTCPAddr("tcp", address)
		if err != nil {
			logger.LOG.Error(err.Error())
			return
		}

		conn, err := net.DialTCP("tcp", nil, raddr)
		if err != nil {
			logger.LOG.Error(err.Error())
			return
		}

		err = conn.SetKeepAlive(true)
		if err != nil {
			logger.LOG.Error(err.Error())
			return
		}

		err = conn.SetKeepAlivePeriod(time.Second * 30)
		if err != nil {
			logger.LOG.Error(err.Error())
			return
		}

		c.conn = conn
		go c.Reading()
		// 握手 -- 发送数据
		c.Hold()
	}
}

// 第一次数据进行握手
func (c *ConnectionServer) Hold() {
	if c.conn != nil {
		c.conn.Write(c.buffer.PackHold(c.Server))
	}
}

func (c *ConnectionServer) HoldBack(server uint32, sid uint32) {

}

func CreateClientConnection(server uint32, processer IServerSink) *ConnectionServer {
	return &ConnectionServer{
		Server:    server,
		conn:      nil,
		buffer:    CreateBuffer(),
		processer: processer,
	}
}
