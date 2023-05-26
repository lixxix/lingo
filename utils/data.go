package utils

import (
	"bytes"
	"encoding/binary"
)

func PackData(session uint32, option int16, buff []byte) []byte {
	sz := int16(len(buff))
	length := 8 + sz
	writer := bytes.NewBuffer(make([]byte, 0, length))
	binary.Write(writer, binary.BigEndian, &sz)
	binary.Write(writer, binary.BigEndian, &session)
	binary.Write(writer, binary.BigEndian, &option)
	binary.Write(writer, binary.BigEndian, buff)
	return writer.Bytes()
}

func PackClientData(session uint32, buff []byte) []byte {
	sz := int16(len(buff))
	length := 6 + sz
	writer := bytes.NewBuffer(make([]byte, 0, length))
	binary.Write(writer, binary.BigEndian, &sz)
	binary.Write(writer, binary.BigEndian, &session)
	binary.Write(writer, binary.BigEndian, buff)
	return writer.Bytes()
}

// 返回握手包
func PackServerData(server uint32, id uint32) []byte {
	writer := bytes.NewBuffer(make([]byte, 0, 4))
	binary.Write(writer, binary.BigEndian, &server)
	binary.Write(writer, binary.BigEndian, &id)
	return writer.Bytes()
}

func PackMessage(msgid uint16, buff []byte) []byte {
	length := 2 + len(buff)
	writer := bytes.NewBuffer(make([]byte, 0, length))
	binary.Write(writer, binary.BigEndian, &msgid)
	binary.Write(writer, binary.BigEndian, buff)
	return writer.Bytes()
}
