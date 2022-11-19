package gate

import (
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/lixxix/lingo/utils"
)

type WSConn struct {
	Id        uint32
	buffer    *utils.Buffer
	conn      *websocket.Conn
	processer utils.IConnProcesser
}

func (c *WSConn) SetId(id uint32) { c.Id = id }

func (c *WSConn) GetId() uint32 { return c.Id }

func (c *WSConn) GetServerId(sId uint32) uint32 {
	return 0
}

func (c *WSConn) SetServerId(sId uint32, id uint32) {

}

func (c *WSConn) Send(target uint32, data []byte) {
	if c.conn != nil {
		c.conn.WriteMessage(websocket.BinaryMessage, c.buffer.PackGateData(target, data))
	}
}

func (c *WSConn) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *WSConn) Reading() {
	defer func() {
		c.conn.Close()
		c.buffer.Clear()
		if c.processer != nil {
			c.processer.UnRegister(c)
		}

	}()

	for {
		tp, buf, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Println("client read error:", err.Error())
			return
		}

		if tp == websocket.BinaryMessage {
			fmt.Println(buf)
			c.buffer.PushData(buf)
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

		// index, err := c.conn.Read(buff)
		// if err != nil {
		// 	fmt.Println("client read error:", err.Error())
		// 	return
		// }

		// for index >= len(buff) {
		// 	buff = append(buff, make([]byte, utils.BUFFER_SIZE)...)
		// 	sz, err = c.conn.Read(buff[index:])
		// 	index += sz
		// 	if err != nil {
		// 		fmt.Println("read error:", err.Error())
		// 		return
		// 	}
		// }
		// c.buffer.PushData(buff[0:index])
		// for !c.buffer.Empty() {
		// 	head, data, err := c.buffer.PopGateData()
		// 	if err != nil {
		// 		fmt.Println("client package error:", err.Error())
		// 		return
		// 	}
		// 	if c.processer != nil {
		// 		fmt.Println(head)
		// 		c.processer.OnClientMessage(head, data, c)
		// 	}
		// }
	}
}

func CreateWS(conn *websocket.Conn, procsser utils.IConnProcesser) *WSConn {
	con := &WSConn{
		conn:      conn,
		Id:        0,
		buffer:    utils.CreateBuffer(),
		processer: procsser,
	}
	go con.Reading()
	return con
}
