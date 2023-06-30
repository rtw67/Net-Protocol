package main

import (
	"flag"
	"log"
	"net"
	"os"
)

func main() {
	flag.Parse()
	if len(flag.Args()) < 2 {
		log.Fatal("Usage: ", os.Args[0], " <local-address:port> <data>")
	}
	addressPort := flag.Arg(0)
	data := flag.Arg(1)
	var (
		addr = flag.String("a", addressPort, "udp dst address")
	)
	log.SetFlags(log.Lshortfile)

	udpAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		panic(err)
	}
	//建立udp连接
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		panic(err)
	}

	send := []byte(data)
	recv := make([]byte, 10)
	if _, err := conn.Write(send); err != nil {
		log.Fatal(err)
	}
	log.Printf("send :%s", string(send))

	rn, _, err := conn.ReadFrom(recv)
	log.Println(rn)
	if err != nil {
		panic(err)
	}
	log.Printf("recv :%s", string(recv[:rn]))
}
