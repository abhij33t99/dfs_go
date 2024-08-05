package main

import (
	"log"
	"time"

	"github.com/abhij33t99/dfs_go/p2p"
)

func main() {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		//TODO: onPeer func
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	serverOpts := ServerOpts{
		ListenAddr:        ":3000",
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPathTransportFunc,
		Transport:         tcpTransport,
	}
	server := NewServer(serverOpts)
	
	go func() {
		time.Sleep(time.Second * 3)
		server.Stop()
	}()

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
