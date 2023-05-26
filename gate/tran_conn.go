package gate

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/utils"
)

type ITranSink interface {
	GetServer() uint32
	GetId() uint32
	OnTranMessage(target uint32, option uint16, msgid uint16, data []byte, client *TranConn)
	OnTranDisconect(client *TranConn)
	OnTranConnected(client *TranConn)
}

// 数据转发机制
type TranConn struct {
	Id         uint32
	Server     uint32
	LinkID     uint32
	LinkServer uint32
	address    string
	connect    atomic.Int32
	bufChan    chan []byte
	conn       net.Conn
	buffer     *utils.Buffer
	recv       bool //第一次接受
	processer  ITranSink
}

func (c *TranConn) GetLinkID() uint32       { return c.LinkID }
func (c *TranConn) GetLinkServer() uint32   { return c.LinkServer }
func (c *TranConn) SetId(id uint32)         { c.Id = id }
func (c *TranConn) GetId() uint32           { return c.Id }
func (c *TranConn) SetServer(server uint32) { c.Server = server }
func (c *TranConn) GetServer() uint32       { return c.Server }

// 转发接口应该发送的精简数据，不应该包含option操作，gate只负责转发
func (c *TranConn) Send(clientid uint32, option int16, data []byte) {
	if c.connect.Load() == 1 {
		c.conn.Write(utils.PackData(clientid, option, data))
	}
}

func (c *TranConn) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *TranConn) ReConnect() {
	go func() {
		if c.conn == nil {
			time.Sleep(time.Second * 2)
			c.Connect(c.address)
		}
	}()
}

func (c *TranConn) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		c.conn = nil
		c.recv = false
		close(c.bufChan)
		if c.processer != nil {
			c.processer.OnTranDisconect(c)
		}
	}()
	buff := make([]byte, 4096)
	sz := 0
	for {
		index, err := c.conn.Read(buff)
		if err != nil {
			fmt.Printf("Tcp Error %s Server:%d Len:%d\n", err.Error(), c.Server, len(buff))
			return
		}

		for index >= len(buff) {
			buff = append(buff, make([]byte, 4096)...)
			fmt.Println("TcpTrans扩充buffer到:", len(buff))
			sz, err = c.conn.Read(buff[index:])
			index += sz
			if err != nil {
				fmt.Printf("Tcp Error 2 %s Server:%d Address:%s Len:%d", err.Error(), c.Server, c.conn.RemoteAddr().String(), len(buff))
				return
			}
		}
		if !c.recv {
			c.recv = true
			if index == 8 {
				reader := bytes.NewReader(buff)
				binary.Read(reader, binary.BigEndian, &c.Server)
				binary.Read(reader, binary.BigEndian, &c.Id)
				logger.LOG.Debug(fmt.Sprintf("中转 hold back :%d :%d", c.Server, c.Id))
				if c.processer != nil {
					c.processer.OnTranConnected(c)
				}

			} else {
				fmt.Println("Hold Failed!")
				return
			}
		} else {
			select {
			case c.bufChan <- utils.CopyBytes(buff[:index]):
			default:
				continue
			}
		}

	}
}

func (c *TranConn) Processing() {
	fmt.Println("processer")
	defer func() {
		fmt.Println("--------------------------------")
	}()
	for buf := range c.bufChan {
		c.buffer.PushData(buf)
		if len(c.bufChan) == 0 {
			pks, err := c.buffer.PopTranGroup()
			if err != nil {
				fmt.Println("package error:", err.Error())
				return
			}
			if c.processer != nil {
				for _, pk := range pks {
					c.processer.OnTranMessage(pk.Session, pk.Option, pk.MsgId, pk.Buff, c)
				}
			}
		}
	}
}

func (c *TranConn) Connect(address string) error {
	c.address = address
	logger.LOG.Debug(fmt.Sprintf("中转数据: tarn link %s", address))
	if c.conn == nil {
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
		c.connect.Store(1)
		go c.Reading()
		go c.Processing()
		c.Hold()
	}
	return nil
}

// 当连接到中间层的服务器时告知服务器数据--中层服务器也会将数据发送回来
func (c *TranConn) Hold() {
	if c.processer != nil {
		c.SetId(c.processer.GetId())
		c.SetServer(c.processer.GetServer())
	}
	if c.conn != nil {
		c.conn.Write(utils.PackServerData(c.GetServer(), c.GetId()))
	}
}

func CreateTranConn(processer ITranSink) *TranConn {
	return &TranConn{
		conn:      nil,
		buffer:    utils.CreateBuffer(),
		processer: processer,
		bufChan:   make(chan []byte, 1024),
	}
}
