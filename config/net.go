package config

import (
	"flag"
	"net"

	tcpip "github.com/brewlin/net-protocol/protocol"
)

// mac地址
var Mac = flag.String("mac", "aa:00:01:01:01:01", "mac address to use in tap device")

// 网卡名
var NicName = "tap"

// 路由网段
var Cidrname = "192.168.1.0/24"

// localip
var Address = flag.String("adr", "192.168.1.1", "ip address")
var LocalAddres = tcpip.Address(net.ParseIP("192.168.1.1").To4())

// 物理网卡名,ip 作为连通外网的网关使用,不填默认自动获取
var HardwardIp = ""
var HardwardName = ""

var Port = flag.Int("port", 9000, "port")
var LocalPort uint16 = 9000

var Path = flag.String("path", "binary_file", "binary file path")

func SetConfig() {
	LocalAddres = tcpip.Address(net.ParseIP(*Address).To4())
	LocalPort = (uint16)(*Port)
}
