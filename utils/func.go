package utils

import (
	"fmt"
	"math/big"
	"net"
	"strings"
)

const (
	CenterServer uint32 = 0
	GateServer          = 1
)

func InetItoa(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

func IpToInt(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(strings.Split(ip, ":")[0]).To4())
	return ret.Int64()
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

func init() {

}
