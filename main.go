package main

import (
	"bytes"
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
	// server2 := makeServer(":4000", "")

	go func() {
		log.Fatal(server1.Start())
	}()
	time.Sleep(time.Second * 2)

	go server2.Start()
	time.Sleep(time.Second * 2)

	// for i := 0; i < 10; i++ {
	data := bytes.NewReader([]byte("my big data file here!"))
	server2.Store("coolPicture.jpg", data)
	time.Sleep(time.Millisecond * 5)
	// }

	// r, err := server2.Get("coolPicture.jpg")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// b, err := io.ReadAll(r)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(string(b))
}
