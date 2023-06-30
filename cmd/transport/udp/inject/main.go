package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
	"flag"

	"io"
	"os"

	"github.com/brewlin/net-protocol/config"
	"github.com/brewlin/net-protocol/pkg/logging"
	"github.com/brewlin/net-protocol/pkg/waiter"
	tcpip "github.com/brewlin/net-protocol/protocol"
	"github.com/brewlin/net-protocol/protocol/link/fdbased"
	"github.com/brewlin/net-protocol/protocol/link/rawfile"
	"github.com/brewlin/net-protocol/protocol/link/tuntap"
	"github.com/brewlin/net-protocol/protocol/network/arp"
	"github.com/brewlin/net-protocol/protocol/network/ipv4"
	"github.com/brewlin/net-protocol/protocol/network/ipv6"
	"github.com/brewlin/net-protocol/protocol/transport/tcp"
	"github.com/brewlin/net-protocol/protocol/transport/udp"
	"github.com/brewlin/net-protocol/stack"
)

func init() {
	flag.Parse()
	config.SetConfig()
	logging.Setup()
}

func max(a ,b int64) int64{
	var res int64
	if(a > b) {
		res = a
	}else {
		res = b
	}

	return res;
}

func min(a ,b int64) int64{
	var res int64
	if(a > b) {
		res = b
	}else {
		res = a
	}

	return res;
}

func main() {
	// if len(flag.Args()) < 4 {
	// 	log.Fatal("Usage: ", os.Args[0], " -mac <mac> -ip <ip> -port <port> <data_path>")
	// }
	
	//fmt.Println(*config.Path)

	file, err := os.Open(*config.Path)
	if err != nil {
		fmt.Println("打开文件失败:", err)
		return
	}
	defer file.Close()

	// 获取文件的大小
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("获取文件信息失败:", err)
		return
	}
	fileSize := fileInfo.Size()

	// 创建一个和文件大小相同的byte数组
	byteArray := make([]byte, fileSize)

	// 读取文件内容到byte数组
	n, err := file.Read(byteArray)
	if err != nil && err != io.EOF {
		fmt.Println("读取文件失败:", err)
		return
	}
	fmt.Printf("读取了%d个字节\n", n)

	// 打印byte数组的内容
	fmt.Println(byteArray)

	parseAddr := net.ParseIP(*config.Address)

	//解析地址ip地址，ipv4 或者ipv6 地址都支持
	var addr tcpip.Address
	var proto tcpip.NetworkProtocolNumber
	if parseAddr.To4() != nil {
		addr = tcpip.Address(parseAddr.To4())
		proto = ipv4.ProtocolNumber
	} else if parseAddr.To16() != nil {
		addr = tcpip.Address(parseAddr.To16())
		proto = ipv6.ProtocolNumber
	} else {
		log.Fatalf("Unknown IP type:%v", parseAddr)
	}


	//虚拟网卡配置
	conf := &tuntap.Config{
		Name: config.NicName,
		Mode: tuntap.TAP,
	}

	var fd int
	//新建虚拟网卡
	fd, err = tuntap.NewNetDev(conf)
	if err != nil {
		log.Fatal(err)
	}

	//启动tap网卡
	tuntap.SetLinkUp(config.NicName)
	//设置tap网卡ip地址
	tuntap.AddIP(config.NicName, *config.Address)
	tuntap.SetRoute(config.NicName, config.Cidrname)

	//解析mac地址
	maddr, err := tuntap.GetHardwareAddr(config.NicName)
	if err != nil {
		log.Fatal(*config.Mac)
	}

	//抽象网卡层接口
	linkID := fdbased.New(&fdbased.Options{
		FD:                 fd,
		MTU:                1500,
		Address:            tcpip.LinkAddress(maddr),
	})
	//新建相关协议的协议栈
	s := stack.New([]string{ipv4.ProtocolName, ipv6.ProtocolName, arp.ProtocolName}, []string{tcp.ProtocolName, udp.ProtocolName}, stack.Options{})
	
	//新建抽象网卡
	if err := s.CreateNamedNIC(1, "vnic1", linkID); err != nil {
		log.Fatal(err)
	}

	//在该协议栈上添加和注册相关的网络层协议 也就是注册本地地址
	if err := s.AddAddress(1, proto, addr); err != nil {
		log.Fatal(err)
	}
	//在该协议栈上添加和注册arp协议
	if err := s.AddAddress(1, arp.ProtocolNumber, arp.ProtocolAddress); err != nil {
		log.Fatal(err)
	}
	//添加默认路由
	s.SetRouteTable([]tcpip.Route{
		{
			Destination: tcpip.Address(strings.Repeat("\x00", len(addr))),
			Mask:        tcpip.AddressMask(strings.Repeat("\x00", len(addr))),
			Gateway:     "",
			NIC:         1,
		},
	})
	
	//新建一个UDP端
	ep, wq:= udpListen(s, ipv4.ProtocolNumber)
	defer ep.Close()
	// 打印byte数组的内容
	//fmt.Println(byteArray)
			
		
	//创建队列 通知 channel
	waitEntry, notifych := waiter.NewChannelEntry(nil)
	wq.EventRegister(&waitEntry, waiter.EventIn)
	defer wq.EventUnregister(&waitEntry)

	var saddr tcpip.FullAddress

	go injectFrame(fd, byteArray)
	for {
		buf := make([]byte, 98)
		_, err := rawfile.BlockingRead(fd, buf)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("@网卡 :recv ", buf)

		v, _, err := ep.Read(&saddr)
		if err != nil {
			if err == tcpip.ErrWouldBlock {
				<-notifych
				continue
			}
			return
		}
		fmt.Printf("@main :read and write data:%s %v", string(v), saddr)
		_, _, err = ep.Write(tcpip.SlicePayload(v), tcpip.WriteOptions{To: &saddr})
		if err == tcpip.ErrWouldBlock {
			<-notifych
		}
		if err != nil && err != tcpip.ErrWouldBlock {
			log.Fatal(err)
		}
	}
}

func injectFrame(fd int, frame []byte){
	for{
		if err := rawfile.NonBlockingWrite(fd, frame); err != nil{
			log.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func udpListen(s *stack.Stack, proto tcpip.NetworkProtocolNumber) (tcpip.Endpoint, waiter.Queue){
	var wq waiter.Queue
	//新建一个udp端
	ep, err := s.NewEndpoint(udp.ProtocolNumber, proto, &wq)
	if err != nil {
		log.Fatal(err)
	}

	if err := ep.Bind(tcpip.FullAddress{1, "", config.LocalPort}, nil); err != nil {
		log.Fatal("Bind failed: ", err)
	}
	//注意udp是无连接的，它不需要listen
	return ep, wq
}

