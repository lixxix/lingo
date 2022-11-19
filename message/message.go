package message

import (
	"bytes"
	"encoding/binary"
)

func ParserPackage(data []byte) (main int16, sub int16, buff []byte) {
	if len(data) < 4 {
		return
	}

	reader := bytes.NewReader(data[0:4])
	binary.Read(reader, binary.BigEndian, &main)
	binary.Read(reader, binary.BigEndian, &sub)
	buff = data[4:]
	return
}

func PackPackage(main int16, sub int16, buff []byte) []byte {
	writer := bytes.NewBuffer([]byte{})
	binary.Write(writer, binary.BigEndian, &main)
	binary.Write(writer, binary.BigEndian, &sub)
	binary.Write(writer, binary.BigEndian, &buff)
	return writer.Bytes()
}

const (
	M_Center int16 = 0
)

const (
	S_Register int16 = 1 //注册数据
)

type ServerRegister struct {
	Port uint16
	Ip   int64
}

func (r *ServerRegister) Bytes() []byte {
	writer := bytes.NewBuffer([]byte{})
	binary.Write(writer, binary.BigEndian, r.Port)
	binary.Write(writer, binary.BigEndian, r.Ip)
	return writer.Bytes()
}

func (r *ServerRegister) Parser(buf []byte) {
	reader := bytes.NewReader(buf)
	binary.Read(reader, binary.BigEndian, &r.Port)
	binary.Read(reader, binary.BigEndian, &r.Ip)
}

type TranServer struct {
	Ip     int64
	Port   uint16
	Server uint32
	Id     uint32
}

func (t *TranServer) Bytes() []byte {
	writer := bytes.NewBuffer([]byte{})
	binary.Write(writer, binary.BigEndian, t.Ip)
	binary.Write(writer, binary.BigEndian, t.Port)
	binary.Write(writer, binary.BigEndian, t.Server)
	binary.Write(writer, binary.BigEndian, t.Id)
	return writer.Bytes()
}

func (t *TranServer) Parser(buf []byte) {
	reader := bytes.NewReader(buf)
	binary.Read(reader, binary.BigEndian, &t.Ip)
	binary.Read(reader, binary.BigEndian, &t.Port)
	binary.Read(reader, binary.BigEndian, &t.Server)
	binary.Read(reader, binary.BigEndian, &t.Id)
}
