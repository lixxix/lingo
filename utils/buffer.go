package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
)

// 4k的数据作为基础缓冲

const (
	MSG_HEAD_LEN     = 2
	SESSION_LEN      = 4
	SERVER_LEN       = 2
	OPTION_LEN       = 2
	MSG_CLIENT_LEN   = MSG_HEAD_LEN + SERVER_LEN
	PACKAGE_HEAD_LEN = MSG_HEAD_LEN + SESSION_LEN + OPTION_LEN
	BUFFER_SIZE      = 4096
)

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

func (b *Buffer) Size() int {
	return b.index
}

// 获取缓存- 结束时后进行发送
func (b *Buffer) Buffer() []byte {
	defer func() {
		b.Clear()
	}()
	return b.buff[0:b.index]
}

func (b *Buffer) Cap() int {
	return b.size
}

func (b *Buffer) PushData(d []byte) {
	b.mu.Lock()
	for b.index+len(d) > b.size {
		b.buff = append(b.buff, make([]byte, BUFFER_SIZE)...)
		b.size += BUFFER_SIZE
	}
	copy(b.buff[b.index:], d)
	b.index += len(d)
	b.mu.Unlock()
}

// 客户端数据包解包 2bytes-2bytes-buff(2+len)
//
//	size -server-(msgid+data)
func (b *Buffer) PopClientData() (server uint16, msgid uint16, buff []byte, err error) {
	// b.mu.Lock()
	// defer b.mu.Unlock()

	if b.index < MSG_CLIENT_LEN {
		err = errors.New(" client data too short")
		return
	}

	reader := bytes.NewReader(b.buff[0:b.index])
	size := int16(0)
	binary.Read(reader, binary.BigEndian, &size)
	binary.Read(reader, binary.BigEndian, &server)
	if reader.Len() < int(size) {
		err = fmt.Errorf("package less len:%d, size:%d", reader.Len(), size)
		return
	}
	// 返回了msgid，并且将msgid buff 返回到gate
	binary.Read(reader, binary.BigEndian, &msgid)

	buff = make([]byte, size)
	copy(buff, b.buff[MSG_CLIENT_LEN:b.index])
	b.index -= int(size + MSG_CLIENT_LEN)
	if b.index != 0 {
		copy(b.buff[0:b.index], b.buff[int(size+MSG_CLIENT_LEN):])
	}
	return
}

type ServerPacket struct {
	Session uint32
	Option  uint16
	MsgId   uint16
	Buff    []byte
}

type ClientPacket struct {
	Session uint32
	MsgId   uint16
	Buff    []byte
}

func (b *Buffer) PopServerGroup() (pkg []*ServerPacket, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.index < PACKAGE_HEAD_LEN {
		err = errors.New("data too short")
		return
	}
	reader := bytes.NewReader(b.buff[0:b.index])
	pos := int(0)
	size := int16(0)
	for pos < b.index {
		pk := &ServerPacket{}
		binary.Read(reader, binary.BigEndian, &size)
		binary.Read(reader, binary.BigEndian, &pk.Session)
		binary.Read(reader, binary.BigEndian, &pk.Option)
		if reader.Len() < int(size) {
			copy(b.buff[0:b.index-int(pos)], b.buff[pos:])
			b.index -= pos
			return
		}

		binary.Read(reader, binary.BigEndian, &pk.MsgId)
		pos += PACKAGE_HEAD_LEN

		if size < 2 {
			err = errors.New(fmt.Sprintf("package size error:%d", size))
			return
		}

		pk.Buff = make([]byte, size-2)
		binary.Read(reader, binary.BigEndian, pk.Buff)
		pos += int(size)
		pkg = append(pkg, pk)
	}

	b.index = 0
	return
}

func (b *Buffer) PopServerData() (session uint32, option uint16, msgid uint16, buff []byte, err error) {
	// b.mu.Lock()
	// defer b.mu.Unlock()

	if b.index < PACKAGE_HEAD_LEN {
		err = errors.New("data too short")
		return
	}

	reader := bytes.NewReader(b.buff[0:b.index])
	size := int16(0)

	binary.Read(reader, binary.BigEndian, &size)
	binary.Read(reader, binary.BigEndian, &session)
	binary.Read(reader, binary.BigEndian, &option)
	if reader.Len() < int(size) {
		err = fmt.Errorf("package less len:%d, size:%d", reader.Len(), size)
		return
	}

	binary.Read(reader, binary.BigEndian, &msgid)
	buff = make([]byte, size-2)
	binary.Read(reader, binary.BigEndian, buff)
	b.index -= int(size + PACKAGE_HEAD_LEN)
	if b.index != 0 {
		copy(b.buff[0:b.index], b.buff[int(size+PACKAGE_HEAD_LEN):])
	}
	return
}

func (b *Buffer) PopClientGroup() (pkg []*ClientPacket, err error) {
	reader := bytes.NewReader(b.buff[0:b.index])
	size := int16(0)
	pos := int(0)

	for pos < b.index {
		pk := &ClientPacket{}
		binary.Read(reader, binary.BigEndian, &size)
		binary.Read(reader, binary.BigEndian, &pk.Session)
		if reader.Len() < int(size) {
			copy(b.buff[:b.index-int(pos)], b.buff[pos:])
			b.index -= pos
			return
		}
		pk.Buff = make([]byte, size)
		binary.Read(reader, binary.BigEndian, pk.Buff)
		pk.MsgId = uint16(pk.Buff[0]*16 + pk.Buff[1])

		pos += MSG_CLIENT_LEN
		pos += int(size) + 2
		pkg = append(pkg, pk)
	}
	b.index = 0
	return
}

func (b *Buffer) PopTranGroup() (pkg []*ServerPacket, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.index < PACKAGE_HEAD_LEN {
		err = errors.New("data too short")
		return
	}

	reader := bytes.NewReader(b.buff[0:b.index])
	size := int16(0)
	pos := int(0)

	for pos < b.index {
		pk := &ServerPacket{}

		binary.Read(reader, binary.BigEndian, &size)
		binary.Read(reader, binary.BigEndian, &pk.Session)
		binary.Read(reader, binary.BigEndian, &pk.Option)
		if reader.Len() < int(size) {
			copy(b.buff[:b.index-int(pos)], b.buff[pos:])
			b.index -= pos
			return
		}
		//中转时Buff包含了msgid和data数据。直接发送给gate
		pk.Buff = make([]byte, size)
		binary.Read(reader, binary.BigEndian, pk.Buff)
		pk.MsgId = uint16(pk.Buff[0]*16 + pk.Buff[1])

		pos += PACKAGE_HEAD_LEN
		pos += int(size)
		pkg = append(pkg, pk)
	}

	b.index = 0
	return
}

func (b *Buffer) PopTranData() (session uint32, option uint16, msgid uint16, buff []byte, err error) {
	// b.mu.Lock()
	// defer b.mu.Unlock()

	if b.index < PACKAGE_HEAD_LEN {
		err = errors.New("data too short")
		return
	}

	reader := bytes.NewReader(b.buff[0:b.index])
	size := int16(0)

	binary.Read(reader, binary.BigEndian, &size)
	binary.Read(reader, binary.BigEndian, &session)
	binary.Read(reader, binary.BigEndian, &option)
	if reader.Len() < int(size) {
		err = fmt.Errorf("trans package less len:%d, size:%d", reader.Len(), size)
		return
	}

	binary.Read(reader, binary.BigEndian, &msgid)
	buff = make([]byte, size)
	copy(buff, b.buff[PACKAGE_HEAD_LEN:b.index])
	b.index -= int(size + PACKAGE_HEAD_LEN)
	if b.index != 0 {
		copy(b.buff[0:b.index], b.buff[int(size+PACKAGE_HEAD_LEN):])
	}
	return
}

func (b *Buffer) PopGateData() (session uint32, msgid uint16, buff []byte, err error) {
	// b.mu.Lock()
	// defer b.mu.Unlock()

	if b.index < PACKAGE_HEAD_LEN {
		err = errors.New("data too short")
		return
	}

	reader := bytes.NewReader(b.buff[0:b.index])
	size := int16(0)

	binary.Read(reader, binary.BigEndian, &size)
	binary.Read(reader, binary.BigEndian, &session)
	binary.Read(reader, binary.BigEndian, &msgid)

	if reader.Len() < int(size) {
		err = fmt.Errorf("package less len:%d, size:%d", reader.Len(), size)
		return
	}

	buff = make([]byte, size)
	copy(buff, b.buff[PACKAGE_HEAD_LEN:b.index])
	b.index -= int(size + PACKAGE_HEAD_LEN)
	if b.index != 0 {
		copy(b.buff[0:b.index], b.buff[int(size+PACKAGE_HEAD_LEN):])
	}
	return
}

func CreateBuffer() *Buffer {
	return &Buffer{buff: make([]byte, BUFFER_SIZE), index: 0, size: BUFFER_SIZE}
}
