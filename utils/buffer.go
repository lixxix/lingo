package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
)

//4k的数据作为基础缓冲
const BUFFER_SIZE = 4096

type IBuffer interface {
	Clear()
	Empty()
	GetIndex() int
	GetBufferLength() int
	PopGateData() (interface{}, error) //解析包的具体数据
}

type Buffer struct {
	buff  []byte
	index int
	size  int
	mu    sync.Mutex // guards
}

func (b *Buffer) Clear() {
	b.index = 0
}

func (b *Buffer) Empty() bool {
	return b.index == 0
}

func (b *Buffer) GetIndex() int {
	return b.index
}

func (b *Buffer) GetBufferLength() int {
	return b.size
}

func (b *Buffer) PushData(d []byte) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to close buffer", err)
		}
	}()

	b.mu.Lock()
	for b.index+len(d) > b.size {
		b.buff = append(b.buff, make([]byte, BUFFER_SIZE)...)
		b.size += BUFFER_SIZE
	}
	copy(b.buff[b.index:], d)
	b.index += len(d)
	b.mu.Unlock()
}

func (b *Buffer) PackData(buff []byte) []byte {
	writer := bytes.NewBuffer([]byte{})
	sz := int16(len(buff))
	binary.Write(writer, binary.BigEndian, &sz)
	binary.Write(writer, binary.BigEndian, buff)
	return writer.Bytes()
}

func (b *Buffer) PackGateData(target uint32, buff []byte) []byte {
	writer := bytes.NewBuffer([]byte{})
	sz := int16(len(buff))
	binary.Write(writer, binary.BigEndian, &sz)
	binary.Write(writer, binary.BigEndian, &target)
	binary.Write(writer, binary.BigEndian, buff)
	return writer.Bytes()
}

// 2Byte size other message
func (b *Buffer) PopPackage() (buff []byte, err error) {
	if b.index == 0 {
		return nil, nil
	}

	if b.index < 2 {
		return nil, errors.New("data too short")
	}

	reader := bytes.NewReader(b.buff[0:b.index])

	size := int16(0)
	binary.Read(reader, binary.BigEndian, &size)

	if reader.Len() < int(size) {
		return nil, errors.New("package is wrong")
	}

	buff = make([]byte, size)
	copy(buff, b.buff[2:b.index])
	b.index -= int(size + 2)
	copy(b.buff[0:b.index], b.buff[int(size+2):])

	return
}

func (b *Buffer) PopGateData() (head uint32, buff []byte, err error) {
	if b.index == 0 {
		return 0, nil, nil
	}

	if b.index < 6 {
		err = errors.New("data too short")
		return
	}

	reader := bytes.NewReader(b.buff[0:b.index])
	size := int16(0)

	binary.Read(reader, binary.BigEndian, &size)
	binary.Read(reader, binary.BigEndian, &head)

	if reader.Len() < int(size) {
		err = fmt.Errorf("package less len:%d, size:%d", reader.Len(), size)
		return
	}

	buff = make([]byte, size)
	copy(buff, b.buff[6:b.index])
	b.index -= int(size + 6)
	copy(b.buff[0:b.index], b.buff[int(size+6):])

	return
}

// 发送握手包
func (b *Buffer) PackHold(server uint32) []byte {
	writer := bytes.NewBuffer([]byte{})
	binary.Write(writer, binary.BigEndian, &server)
	return writer.Bytes()
}

//返回握手包
func (b *Buffer) PackServerData(server uint32, id uint32) []byte {
	writer := bytes.NewBuffer([]byte{})
	binary.Write(writer, binary.BigEndian, &server)
	binary.Write(writer, binary.BigEndian, &id)
	return writer.Bytes()
}

func CreateBuffer() *Buffer {
	return &Buffer{buff: make([]byte, BUFFER_SIZE), index: 0, size: BUFFER_SIZE}
}
