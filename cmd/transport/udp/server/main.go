package main

import (
	"log"
	"fmt"
	"os/exec"
	
	"github.com/brewlin/net-protocol/config"
	"github.com/brewlin/net-protocol/pkg/logging"
	"github.com/brewlin/net-protocol/internal/endpoint"
	"github.com/brewlin/net-protocol/pkg/waiter"
	"github.com/brewlin/net-protocol/protocol/network/ipv4"
	"github.com/brewlin/net-protocol/protocol/transport/udp"
	"github.com/brewlin/net-protocol/stack"
	"github.com/brewlin/net-protocol/protocol/link/tuntap"

	tcpip "github.com/brewlin/net-protocol/protocol"
)

func init() {
	logging.Setup()
}

func main() {
	up()
	defer tuntap.DelTap(config.NicName)
	s := endpoint.NewEndpoint()
	echo(s)
}

func echo(s *stack.Stack) {
	var wq waiter.Queue
	//新建一个UDP端
	ep, err := s.NewEndpoint(udp.ProtocolNumber, ipv4.ProtocolNumber, &wq)
	if err != nil {
		log.Fatal(err)
	}
	//绑定本地端口
	if err := ep.Bind(tcpip.FullAddress{1, config.LocalAddres, config.LocalPort}, nil); err != nil {
		log.Fatal("@main : bind failed :", err)
	}
	defer ep.Close()
	//创建队列 通知 channel
	waitEntry, notifych := waiter.NewChannelEntry(nil)
	wq.EventRegister(&waitEntry, waiter.EventIn)
	defer wq.EventUnregister(&waitEntry)

	var saddr tcpip.FullAddress

	for {
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

func up(){
	firstIp, firstNic := ipv4.InternalInterfaces()
	if config.HardwardIp == "" {
		config.HardwardIp = firstIp
	}
	if config.HardwardName == "" {
		config.HardwardName = firstNic
	}

	// //创建网卡
	// if err := tuntap.CreateTap(config.NicName); err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	//启动网卡
	if err := tuntap.SetLinkUp(config.NicName); err != nil {
		fmt.Println(err)
		return
	}
	//添加路由
	if err := tuntap.SetRoute(config.NicName, config.Cidrname); err != nil {
		fmt.Println(err)
		return
	}
	//开启防火墙规则 nat数据包转发
	if err := IpForwardAndNat(); err != nil {
		fmt.Println(err)
		tuntap.DelTap(config.NicName)
		return
	}
}

func IpForwardAndNat() (err error) {
	//清楚本地物联网看的数据包规则， 模拟防火墙
	//out, cmdErr := exec.Command("iptables", "-F").CombinedOutput()
	//if cmdErr != nil {
	//	err = fmt.Errorf("iptables -F %v:%v", cmdErr, string(out))
	//	return
	//}

	out, cmdErr := exec.Command("iptables", "-P", "INPUT", "ACCEPT").CombinedOutput()
	if cmdErr != nil {
		err = fmt.Errorf("iptables -P INPUT ACCEPT %v:%v", cmdErr, string(out))
		return
	}
	out, cmdErr = exec.Command("iptables", "-P", "FORWARD", "ACCEPT").CombinedOutput()
	if cmdErr != nil {
		err = fmt.Errorf("iptables -P FORWARD ACCEPT %v:%v", cmdErr, string(out))
		return
	}
	out, cmdErr = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", config.Cidrname, "-o", config.HardwardName, "-j", "MASQUERADE").CombinedOutput()
	if cmdErr != nil {
		err = fmt.Errorf("iptables nat %v:%v", cmdErr, string(out))
		return
	}
	return
}
