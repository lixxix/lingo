package message

import (
	"bytes"
	"encoding/binary"
)

func ParserPackage(data []byte) (msgid uint16, buff []byte) {
	if len(data) < 4 {
		return
	}
	reader := bytes.NewReader(data[0:4])
	binary.Read(reader, binary.BigEndian, &msgid)
	buff = data[4:]
	return
}

func PackPackage(msgid uint16, buff []byte) []byte {
	writer := bytes.NewBuffer([]byte{})
	binary.Write(writer, binary.BigEndian, &msgid)
	binary.Write(writer, binary.BigEndian, &buff)
	return writer.Bytes()
}

func PackClient(server uint16, msgid uint16, buff []byte) ([]byte, int) {
	writer := bytes.NewBuffer([]byte{})
	sz := int16(len(buff) + 2)
	binary.Write(writer, binary.BigEndian, &sz)
	binary.Write(writer, binary.BigEndian, &server)
	binary.Write(writer, binary.BigEndian, &msgid)
	binary.Write(writer, binary.BigEndian, &buff)
	return writer.Bytes(), writer.Len()
}
