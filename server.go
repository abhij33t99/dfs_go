package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/abhij33t99/dfs_go/p2p"
)

type ServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type Server struct {
	ServerOpts

	peerMu sync.Mutex
	peers  map[string]p2p.Peer
	store  *Store
	quitch chan struct{}
}

func NewServer(opts ServerOpts) *Server {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &Server{
		ServerOpts: opts,
		store:      NewStore(storeOpts),
		quitch:     make(chan struct{}),
		peers:      make(map[string]p2p.Peer),
	}
}

type SMessage struct {
	From    string
	Payload any
}

func (s *Server) broadcast(msg *SMessage) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (s *Server) StoreData(key string, r io.Reader) error {
	//store the data to disk
	//broadcast to al known peers in the network

	msg := SMessage{
		Payload: []byte("storageKey"),
	}
	buff := new(bytes.Buffer)
	if err := gob.NewEncoder(buff).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		if err := peer.Send(buff.Bytes()); err != nil {
			return err
		}
	}

	time.Sleep(time.Second * 3)

	payload := []byte("THIS IS A LARGE FILE")

	for _, peer := range s.peers {
		if err := peer.Send(payload); err != nil {
			return err
		}
	}
	return nil

	// buff := new(bytes.Buffer)
	// tee := io.TeeReader(r, buff)

	// if err := s.store.Write(key, tee); err != nil {
	// 	return err
	// }

	// p := &DataMessage{
	// 	Key:  key,
	// 	Data: buff.Bytes(),
	// }

	// return s.broadcast(&SMessage{
	// 	From: "TODO",
	// 	Payload: p,
	// })
}

func (s *Server) Stop() {
	close(s.quitch)
}

func (s *Server) OnPeer(peer p2p.Peer) error {
	s.peerMu.Lock()
	defer s.peerMu.Unlock()

	s.peers[peer.RemoteAddr().String()] = peer
	log.Printf("connected with remote %s\n", peer.RemoteAddr())
	return nil
}

func (s *Server) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			var m SMessage
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&m); err != nil {
				log.Fatal(err)
			}

			fmt.Printf("recieved: %s\n", string(m.Payload.([]byte)))

			// open up the file to stream
			peer, ok := s.peers[msg.From]
			if !ok {
				panic("peer not found in the peer map")
			}

			buff := make([]byte, 1000)
			if _, err := peer.Read(buff); err != nil {
				panic(err)
			}

			fmt.Printf("bigdata: %s\n", string(buff))

			// if err := s.handleMessage(&m); err != nil {
			// 	log.Println(err)
			// }
		case <-s.quitch:
			return
		}
	}
}

// func (s *Server) handleMessage(msg *SMessage) error {
// 	switch t := msg.Payload.(type) {
// 	case *DataMessage:
// 		fmt.Printf("Recieved data %+v\n", t)
// 	}

// 	return nil
// }

func (s *Server) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error :", err)
			}
		}(addr)
	}
	return nil
}

func (s *Server) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	s.bootstrapNetwork()
	s.loop()

	return nil
}
