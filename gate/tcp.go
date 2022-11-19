package gate

import (
	"fmt"
	"net"

	"github.com/lixxix/lingo/utils"
)

// 创建一个tcp接收的连接
type TcpConn struct {
	Id        uint32
	conn      net.Conn
	buffer    *utils.Buffer
	processer utils.IConnProcesser
	Links     map[uint32]uint32 //保存对应的server的id
}

func (c *TcpConn) SetId(id uint32) { c.Id = id }

func (c *TcpConn) GetId() uint32 { return c.Id }

func (c *TcpConn) GetServerId(sId uint32) uint32 {
	if _, ok := c.Links[sId]; ok {
		return c.Links[sId]
	}
	return 0
}

// 传递0的时候删除对应的serverid
func (c *TcpConn) SetServerId(sId uint32, id uint32) {
	fmt.Println(sId, id)
	if id == 0 {
		delete(c.Links, sId)
	} else {
		c.Links[sId] = id
	}
}

func (c *TcpConn) Send(target uint32, data []byte) {
	if c.conn != nil {
		c.conn.Write(c.buffer.PackGateData(target, data))
	}
}

func (c *TcpConn) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *TcpConn) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		if c.processer != nil {
			c.processer.UnRegister(c)
		}
		fmt.Println("close conn")
	}()

	buff := make([]byte, 4096)
	sz := 0

	for {
		index, err := c.conn.Read(buff)
		if err != nil {
			fmt.Println("client read error:", err.Error())
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
		c.buffer.PushData(buff[0:index])
		for !c.buffer.Empty() {
			head, data, err := c.buffer.PopGateData()
			if err != nil {
				fmt.Println("client package error:", err.Error())
				return
			}
			if c.processer != nil {
				c.processer.OnClientMessage(head, data, c)
			}
		}
	}
}

func CreateTcp(conn net.Conn, procsser utils.IConnProcesser) *TcpConn {
	con := &TcpConn{
		conn:      conn,
		Id:        0,
		buffer:    utils.CreateBuffer(),
		processer: procsser,
		Links:     make(map[uint32]uint32),
	}
	go con.Reading()
	return con
}
