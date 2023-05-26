package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	CenterServer uint32 = 0 //中心协调
	GateServer   uint32 = 1 //网关
	DBServer     uint32 = 2 //db服务器
)

// OPTION
const (
	CLIENT int16 = 0
	SERVER int16 = 1
)

func CopyBytes(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func InetItoa[T int32 | int64 | uint32 | uint64](ip T) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

func IpToInt(ip string) uint32 {
	reader := bytes.NewReader(net.ParseIP(strings.Split(ip, ":")[0]).To4())
	ip_int := uint32(0)
	binary.Read(reader, binary.BigEndian, &ip_int)

	return ip_int
}

func LocalIP() uint32 {
	return IpToInt(GetLocalIP())
}

func GetLocalIP() string {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("net.Interfaces failed, err:", err.Error())
	}
	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()
			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						fmt.Println(ipnet.IP.String())
						return ipnet.IP.String()
					}
				}
			}
		}
	}
	return ""
}

func GetFreePort() (port int, err error) {
	// 解析地址
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, nil
	}
	// 利用 ListenTCP 方法的如下特性
	// 如果 addr 的端口字段为0，函数将选择一个当前可用的端口
	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	// 关闭资源
	defer listen.Close()
	// 为了拿到具体的端口值，我们转换成 *net.TCPAddr类型获取其Port
	return listen.Addr().(*net.TCPAddr).Port, nil
}

func WaitSignal() os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return <-c
}
