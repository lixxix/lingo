package utils

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/packer"
)

/*
	服务器之间传递消息使用
	Id  区分相同server类型的不同
	Server 服务器类型
	第一次连接发送本身的数据信息 发送本身的属性信息
	第一次返回数据包返回id信息
*/
//连接到center的类
type ConnectionServer struct {
	Id         uint32
	Server     uint32
	LinkID     uint32
	LinkServer uint32
	connected  int32 //连接
	conn       net.Conn
	buffer     *Buffer
	address    string
	recv       bool //第一次接受
	processer  IServerSink
}

func (c *ConnectionServer) GetLinkID() uint32       { return c.LinkID }
func (c *ConnectionServer) GetLinkServer() uint32   { return c.LinkServer }
func (c *ConnectionServer) SetId(id uint32)         { c.Id = id }
func (c *ConnectionServer) GetId() uint32           { return c.Id }
func (c *ConnectionServer) SetServer(server uint32) { c.Server = server }
func (c *ConnectionServer) GetServer() uint32       { return c.Server }
func (c *ConnectionServer) Send(target uint32, option int16, data []byte) {
	// 是否需要进行缓存发送，这样在压力大的时候进行堆积一起发。
	if c.conn != nil && atomic.LoadInt32(&c.connected) == 1 {
		c.conn.Write(PackData(target, option, data))
	}
}

func (c *ConnectionServer) Close() {
	if c.conn != nil {
		fmt.Println("主动关闭")
		c.conn.Close()
	}
}

func (c *ConnectionServer) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		atomic.StoreInt32(&c.connected, 0)
		c.recv = false
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
				fmt.Printf("Tcp Error 2 %s Server:%d Address:%s Len:%d", err.Error(), c.Server, c.conn.RemoteAddr().String(), len(buff))
				return
			}
		}
		if !c.recv {
			logger.LOG.Debug("连接： hold back")
			c.recv = true
			if index == 6 {
				logger.LOG.Debug(fmt.Sprintf("Hold %v", buff[0:index]))

				holdback := message.ServerHoldSuccess{}
				packer.ParserData(buff, &holdback)

				c.LinkServer = uint32(holdback.LinkServer)
				c.Id = uint32(holdback.MyId)

				c.LinkServer = uint32(holdback.LinkServer)
				c.LinkID = uint32(holdback.LinkId)
				c.Id = uint32(holdback.MyId)

				if c.processer != nil {
					c.processer.OnServerConnected(c)
				}
			} else {
				logger.LOG.Debug(fmt.Sprintf("Hold Failed! %d %v", index, buff[0:index]))
				return
			}
		} else {
			c.buffer.PushData(buff[0:index])
			for !c.buffer.Empty() {
				target, option, msgid, buf, err := c.buffer.PopServerData()
				if err != nil {
					logger.LOG.Error(fmt.Sprintf("TcpTrans data error: %s", err.Error()))
					return
				} else if buf == nil {
					break
				}
				if c.processer != nil {
					c.processer.OnServerMessage(target, option, msgid, buf, c)
				}
			}
		}

	}
}

func (c *ConnectionServer) ReConnect() {
	go func() {
		for atomic.LoadInt32(&c.connected) == 0 {
			time.Sleep(time.Second * 2)
			if c.Connect(c.address) == nil {
				return
			}
		}
	}()
}

func (c *ConnectionServer) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *ConnectionServer) Connect(address string) error {
	if c.conn == nil || atomic.LoadInt32(&c.connected) == 0 {
		c.address = address
		raddr, err := net.ResolveTCPAddr("tcp", address)
		if err != nil {
			logger.LOG.Error(err.Error())
			return err
		}

		conn, err := net.DialTCP("tcp", nil, raddr)
		if err != nil {
			logger.LOG.Error(err.Error())
			return err
		}

		err = conn.SetKeepAlive(true)
		if err != nil {
			logger.LOG.Error(err.Error())
			return err
		}

		err = conn.SetKeepAlivePeriod(time.Second * 30)
		if err != nil {
			logger.LOG.Error(err.Error())
			return err
		}

		c.conn = conn

		go c.Reading()
		// 握手 -- 发送数据

		atomic.StoreInt32(&c.connected, 1)
		c.Hold()

		// go func() {
		// 	time.Sleep(time.Second * 5)
		// 	if atomic.LoadInt32(&c.connected) == 1 {
		// 		c.conn.Close()
		// 	}
		// }()
	}
	return nil
}

// 第一次数据进行握手自报家门
func (c *ConnectionServer) Hold() {
	if c.conn != nil {
		c.conn.Write(packer.PackMessage(&message.ServerHold{
			Server: int16(c.Server),
			Id:     int16(c.Id),
		}))
	}
}

func (c *ConnectionServer) CreateBuffer() {
	if c.buffer == nil {
		c.buffer = CreateBuffer()
	}
}

func (c *ConnectionServer) HoldBack(server uint32, sid uint32) {
	fmt.Println("Hold bak")
}

func (c *ConnectionServer) SetSink(sink IServerSink) {
	c.processer = sink
}

func CreateClientConnection(server uint32, id uint32, processer IServerSink) *ConnectionServer {
	return &ConnectionServer{
		Id:        id,
		Server:    server,
		connected: 0,
		conn:      nil,
		buffer:    CreateBuffer(),
		processer: processer,
	}
}
