package gate

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lixxix/lingo/utils"
	"nhooyr.io/websocket"
)

var Recv int64
var Send int64

type WSConn struct {
	Id            uint32
	SendChan      chan []byte
	connect       atomic.Int32
	first_package bool
	Recv          atomic.Int64
	recv_buffer   *utils.Buffer
	send_buffer   *utils.Buffer
	conn          *websocket.Conn
	processer     utils.IClientProcesser
	Context       context.Context
	Links         map[uint32]uint32 //保存对应的server的id
	Conns         map[uint32]utils.IServerConn
	IP            string
}

func (c *WSConn) SetId(id uint32) { c.Id = id }

func (c *WSConn) GetId() uint32 { return c.Id }

func (c *WSConn) GetServerId(sId uint32) uint32 {
	if _, ok := c.Links[sId]; ok {
		return c.Links[sId]
	}
	return 0
}

func (c *WSConn) GetRecv() int64 {
	return c.Recv.Load()
}
func (c *WSConn) ClearRecv() {
	c.Recv.Store(0)
}

func (c *WSConn) SetLocation(server uint32, link utils.IServerConn) {
	// 设置连接
	c.Conns[server] = link
} // 设置位置--相当于游戏的连接

func (c *WSConn) GetLocation(server uint32) utils.IServerConn {
	return c.Conns[server]
}

func (c *WSConn) LeaveLocation(conn utils.IServerConn) bool {
	ser := c.Conns[conn.GetServer()]
	if ser == conn {
		fmt.Println("删除连接", conn.GetServer())
		delete(c.Conns, conn.GetServer())
	}
	return true
}

func (c *WSConn) AllLocation() []utils.IServerConn {
	lks := make([]utils.IServerConn, 0)
	for _, v := range c.Conns {
		lks = append(lks, v)
	}
	return lks
}

func (c *WSConn) RemoteAddr() string {
	return c.IP
}

func (c *WSConn) SetServerId(sId uint32, id uint32) {
	if id == 0 {
		delete(c.Links, sId)
	} else {
		c.Links[sId] = id
	}
}

func (c *WSConn) Send(session uint32, data []byte) {
	if c.connect.Load() == 1 {
		select {
		case c.SendChan <- utils.PackClientData(session, data):
		default:
			return
		}
	}
}

func (c *WSConn) Close() {
	if c.connect.Load() == 1 {
		c.recv_buffer.Clear()
		c.send_buffer.Clear()
		c.connect.Store(0)
		c.conn.Close(websocket.StatusNormalClosure, "")
	}
}

func (c *WSConn) write(data []byte) error {
	if c.conn == nil {
		return errors.New("conn is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	return c.conn.Write(ctx, websocket.MessageBinary, data)
}

func (c *WSConn) Writing() {
	for buf := range c.SendChan {
		atomic.AddInt64(&Send, 1)
		c.send_buffer.PushData(buf)
		if len(c.SendChan) == 0 {
			err := c.write(c.send_buffer.Buffer())
			if err != nil {
				fmt.Println(c.Id, err.Error())
				if c.connect.Load() == 1 {
					c.connect.Store(0)
					c.conn.Close(websocket.StatusGoingAway, "")
				}
				return
			}
		}
	}
}

func (c *WSConn) Reading() {
	defer func() {
		c.Close()
		c.connect.Store(0)
		close(c.SendChan)
		c.processer.UnRegister(c)
	}()

	for {
		tp, buf, err := c.conn.Read(c.Context)
		if err != nil {
			fmt.Println(c.Id, err.Error())
			return
		}

		if tp == websocket.MessageText {
			fmt.Println(string(buf))
			c.conn.Write(c.Context, websocket.MessageText, buf)
			continue

			// c.write(c.send_buffer.Buffer())
		}

		if c.first_package {
			if len(buf) == utils.PACKAGE_HEAD_LEN {
				var singal uint16 = 0
				reader := bytes.NewReader(buf)
				binary.Read(reader, binary.LittleEndian, &singal)
				if singal == 0 {
					c.first_package = false
					c.processer.Register(c)
				} else {
					return
				}
			}
			continue
		}
		if tp == websocket.MessageBinary {
			c.recv_buffer.PushData(buf)
			atomic.AddInt64(&Recv, 1)
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
}

func CreateWS(conn *websocket.Conn, ctx context.Context, procsser utils.IClientProcesser) *WSConn {
	con := &WSConn{
		conn:          conn,
		Id:            0,
		first_package: true,
		SendChan:      make(chan []byte, 256),
		recv_buffer:   utils.CreateBuffer(),
		send_buffer:   utils.CreateBuffer(),
		processer:     procsser,
		Context:       ctx,
		Links:         make(map[uint32]uint32),
		Conns:         make(map[uint32]utils.IServerConn),
	}
	con.connect.Store(1)
	return con
}
