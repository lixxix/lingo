package gate

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/lixxix/lingo/utils"
)

// 创建一个tcp接收的连接
type TcpConn struct {
	Id            uint32
	conn          net.Conn
	connect       int32
	Recv          atomic.Int64
	send          chan []byte
	first_package bool
	recv_buffer   *utils.Buffer
	send_buffer   *utils.Buffer
	processer     utils.IClientProcesser
	Links         map[uint32]uint32 //保存对应的server的id
}

func (c *TcpConn) SetId(id uint32) { c.Id = id }

func (c *TcpConn) GetId() uint32 { return c.Id }

func (c *TcpConn) GetServerId(sId uint32) uint32 {
	if _, ok := c.Links[sId]; ok {
		return c.Links[sId]
	}
	return 0
}

func (c *TcpConn) GetRecv() int64 {
	return c.Recv.Load()
}
func (c *TcpConn) ClearRecv() {
	c.Recv.Store(0)
}

func (c *TcpConn) SetLocation(server uint32, client utils.IServerConn) {

} // 设置位置--相当于游戏的连接

func (c *TcpConn) GetLocation(server uint32) utils.IServerConn {
	return nil
}

func (c *TcpConn) LeaveLocation(conn utils.IServerConn) bool {
	return true
}

func (c *TcpConn) AllLocation() []utils.IServerConn {
	return make([]utils.IServerConn, 0)
}

func (c *TcpConn) RemoteAddr() string {
	addr := c.conn.RemoteAddr()
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		fmt.Println("Unable to get TCP address:", addr)
		return ""
	}
	return tcpAddr.IP.String()
}

// 传递0的时候删除对应的serverid
func (c *TcpConn) SetServerId(sId uint32, id uint32) {
	if id == 0 {
		delete(c.Links, sId)
	} else {
		c.Links[sId] = id
	}
}

func (c *TcpConn) Send(target uint32, data []byte) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r, c.connect)
		}
	}()
	if c.connect == 1 {
		// c.send <- utils.PackData(target, 0, data) tcp 效率会比较差
		if _, err := c.conn.Write(utils.PackClientData(target, data)); err != nil {
			c.connect = 0
			fmt.Println(err)
			if c.conn != nil {
				c.conn.Close()
			}
		}
	}
}

func (c *TcpConn) Close() {
	if c.connect == 1 {
		c.connect = 0
		c.recv_buffer.Clear()
		c.conn.Close()
	}
}

func (c *TcpConn) Reading() {
	defer func() {
		c.Close()
		close(c.send)
		if c.processer != nil {
			c.processer.UnRegister(c)
		}
	}()

	buf := make([]byte, 4096)
	sz := 0

	for {
		index, err := c.conn.Read(buf)
		if err != nil {
			return
		}

		for index >= len(buf) {
			buf = append(buf, make([]byte, utils.BUFFER_SIZE)...)
			sz, err = c.conn.Read(buf[index:])
			index += sz
			if err != nil {
				fmt.Println("read error:", err.Error())
				return
			}
		}

		if c.first_package {
			if index == utils.PACKAGE_HEAD_LEN {
				var singal uint16 = 0
				reader := bytes.NewReader(buf)
				binary.Read(reader, binary.LittleEndian, &singal)
				if singal == 0 {
					c.first_package = false
					c.processer.Register(c)
				} else {
					c.conn.Close()
				}
			}
			continue
		}

		c.recv_buffer.PushData(buf[0:index])
		for !c.recv_buffer.Empty() {
			c.Recv.Add(1)
			server, msgid, data, err := c.recv_buffer.PopClientData()
			if err != nil {
				fmt.Println("client package error:", err.Error())
				return
			}
			if c.processer != nil {
				c.processer.OnClientMessage(server, msgid, data, c)
			}
		}
	}
}

// 这种方式也不是安全的方式
func (c *TcpConn) Writing() {
	defer func() {
		c.Close()
	}()
	for buf := range c.send {
		c.send_buffer.PushData(buf)
		if len(c.send) == 0 {
			c.conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
			if _, err := c.conn.Write(c.send_buffer.Buffer()); err != nil {
				return
			}
		}
	}
}

func CreateTcp(conn net.Conn, procsser utils.IClientProcesser) *TcpConn {
	con := &TcpConn{
		conn:          conn,
		Id:            0,
		connect:       1,
		first_package: true,
		send:          make(chan []byte, 10),
		recv_buffer:   utils.CreateBuffer(),
		send_buffer:   utils.CreateBuffer(),
		processer:     procsser,
		Links:         make(map[uint32]uint32),
	}

	return con
}
