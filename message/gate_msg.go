package message

import (
	"bytes"
	"encoding/binary"
)

const (
	M_Gate int16 = 0
)

const (
	NetShutDown uint16 = 1 //关闭客户机链接
	NetIsLink   uint16 = 2 //查看客户是否连接
	NetClinetIP uint16 = 3 //查看客户机的ip地址
	ClientLink  uint16 = 4 //客户绑定
	ClientBreak uint16 = 5 //客户端断开
)

func SessionByte(session uint32) []byte {
	writer := bytes.NewBuffer([]byte{})
	binary.Write(writer, binary.BigEndian, &session)
	return writer.Bytes()
}

func GetSessionByte(buf []byte) uint32 {
	Session := uint32(0)
	reader := bytes.NewReader(buf)
	binary.Read(reader, binary.BigEndian, &Session)
	return Session
}

// 用户的连接唯一标志
type GateUserSession struct {
	Session uint32
}

func (s *GateUserSession) Bytes() []byte {
	writer := bytes.NewBuffer([]byte{})
	binary.Write(writer, binary.BigEndian, &s.Session)
	return writer.Bytes()
}

func (s *GateUserSession) Parser(buf []byte) {
	reader := bytes.NewReader(buf)
	binary.Read(reader, binary.BigEndian, &s.Session)
}

type GateUserIP struct {
	Session uint32
	Address string
}
