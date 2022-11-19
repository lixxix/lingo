package gate

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lixxix/lingo/logger"
	"github.com/lixxix/lingo/utils"
)

type ITranSink interface {
	OnTranMessage(target uint32, data []byte, client *TranConn)
	OnTranDisconect(client *TranConn)
	OnTranConnected(client *TranConn)
}

// 数据转发机制
type TranConn struct {
	Id        uint32
	Server    uint32
	conn      net.Conn
	buffer    *utils.Buffer
	recv      bool //第一次接受
	processer ITranSink
	mutex     sync.Mutex
}

func (c *TranConn) SetId(id uint32)         { c.Id = id }
func (c *TranConn) GetId() uint32           { return c.Id }
func (c *TranConn) SetServer(server uint32) { c.Server = server }
func (c *TranConn) GetServer() uint32       { return c.Server }
func (c *TranConn) Send(clientid uint32, data []byte) {
	c.mutex.Lock()
	if c.conn != nil {
		c.conn.Write(c.buffer.PackGateData(clientid, data))
	}
	c.mutex.Unlock()
}

func (c *TranConn) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *TranConn) ReConnect() {

}

func (c *TranConn) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		c.conn = nil
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
			logger.LOG.Debug("中转 hold back")
			c.recv = true
			if index == 8 {
				reader := bytes.NewReader(buff)
				binary.Read(reader, binary.BigEndian, &c.Server)
				binary.Read(reader, binary.BigEndian, &c.Id)
				if c.processer != nil {
					c.processer.OnTranConnected(c)
				}
			} else {
				fmt.Println("Hold Failed!")
				return
			}
		} else {
			c.buffer.PushData(buff[0:index])
			for !c.buffer.Empty() {
				target, buf, err := c.buffer.PopGateData()
				if err != nil {
					fmt.Println("TcpTrans data error: ", err.Error())
					return
				} else if buf == nil {
					break
				}
				if c.processer != nil {
					c.processer.OnTranMessage(target, buf, c)
				}
			}
		}

	}
}

func (c *TranConn) Connnect(address string) {
	logger.LOG.Debug(fmt.Sprintf("中转数据: tarn link %s", address))
	if c.conn == nil {
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
		c.Hold()
	}
}

// 当连接到中间层的服务器时告知服务器数据--中层服务器也会将数据发送回来
func (c *TranConn) Hold() {
	logger.LOG.Debug("中转：hold info")
	if c.conn != nil {
		c.conn.Write(c.buffer.PackServerData(c.GetServer(), c.GetId()))
	}
}

func CreateTranConn(processer ITranSink) *TranConn {
	return &TranConn{
		conn:      nil,
		buffer:    utils.CreateBuffer(),
		processer: processer,
	}
}
