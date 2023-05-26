package main

import (
	"fmt"
	"testing"

	"github.com/lixxix/lingo/message"
	"github.com/lixxix/lingo/packer"
	"github.com/lixxix/lingo/utils"
)

type TestData struct {
	Name string
	Len  int32
}

func TestPackData(t *testing.T) {
	Data := &TestData{
		Name: "Hello world",
		Len:  10,
	}
	buf, size := message.PackClient(99, 10, packer.PackMessage(Data)) //结构的数据序列化
	fmt.Println("size:", size)
	lbuffer := utils.CreateBuffer()
	lbuffer.PushData(buf)
	fmt.Println(lbuffer.Size())

	server, _, data, err := lbuffer.PopClientData()
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(server, data)
	if lbuffer.Empty() {
		fmt.Println("Pass PopClientData")
	}

	buf = utils.PackData(99999, 1, message.PackPackage(1000, packer.PackMessage(Data)))
	lbuffer.PushData(buf)
	lbuffer.PushData(buf)
	lbuffer.PushData(buf)
	lbuffer.PushData(buf)

	fmt.Println(lbuffer.PopServerGroup())

	if lbuffer.Empty() {
		fmt.Println("pass PopServerData`")
	}
}

func BenchmarkUnpackSingle(b *testing.B) {
	Data := &TestData{
		Name: "Hello world",
		Len:  10,
	}
	lbuffer := utils.CreateBuffer()
	buf := utils.PackData(99999, 1, message.PackPackage(1000, packer.PackMessage(Data)))
	for i := 0; i < 100000; i++ {
		lbuffer.PushData(buf)
	}

	for !lbuffer.Empty() {
		lbuffer.PopServerData()
	}

	for i := 0; i < 100000; i++ {
		lbuffer.PushData(buf)
	}

	lbuffer.PopServerGroup()
}

// func BenchmarkGroup(b *testing.B) {
// 	Data := &TestData{
// 		Name: "Hello world",
// 		Len:  10,
// 	}
// 	lbuffer := utils.CreateBuffer()
// 	buf := utils.PackData(99999, 1, message.PackPackage(1000, packer.PackMessage(Data)))
// 	for i := 0; i < 10000; i++ {
// 		lbuffer.PushData(buf)
// 	}

// 	lbuffer.PopServerGroup()
// }

func TestMap(t *testing.T) {
	xxx := make(map[int]int)
	cc := 1
	xxx[cc]++
	// fmt.Println(xx, ret, xxx)
	xx, ret := xxx[0]
	fmt.Println(xx, ret, xxx)
}
