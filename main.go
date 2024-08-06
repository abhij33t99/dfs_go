package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/abhij33t99/dfs_go/p2p"
)

func makeServer(listenAddr string, nodes ...string) *Server {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	serverOpts := ServerOpts{
		// ListenAddr:        ":3000",
		EncKey:            newEncryptionKey(),
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransportFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	server := NewServer(serverOpts)
	tcpTransport.OnPeer = server.OnPeer
	return server
}

func main() {
	server1 := makeServer(":3000", "")
	server2 := makeServer(":4000", ":3000")
	server3 := makeServer(":5000", ":3000", ":4000")

	go func() {
		log.Fatal(server1.Start())

	}()
	time.Sleep(time.Second * 2)

	go func() {
		log.Fatal(server2.Start())

	}()
	time.Sleep(time.Second * 2)

	go server3.Start()
	time.Sleep(time.Second * 2)

	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("coolpic_%d.gif", i)
		data := bytes.NewReader([]byte("my big data file here!"))
		server3.Store(key, data)
		// time.Sleep(time.Millisecond * 5)

		if err := server3.store.Delete(key); err != nil {
			log.Fatal(err)
		}

		r, err := server3.Get(key)
		if err != nil {
			log.Fatal(err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(b))
	}
}
