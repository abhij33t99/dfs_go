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
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
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
	//broadcast to all known peers in the network

	buff := new(bytes.Buffer)
	tee := io.TeeReader(r, buff)

	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := SMessage{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	msgBuff := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuff).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		if err := peer.Send(msgBuff.Bytes()); err != nil {
			return err
		}
	}

	time.Sleep(time.Second * 3)

	for _, peer := range s.peers {
		n, err := io.Copy(peer, buff)
		if err != nil {
			return err
		}

		fmt.Println("Recieved and written bytes to disk", n)
	}
	return nil
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
			var sm SMessage
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&sm); err != nil {
				log.Fatal(err)
			}

			if err := s.handleMessage(msg.From, &sm); err != nil {
				log.Println(err)
			}
		case <-s.quitch:
			return
		}
	}
}

func (s *Server) handleMessage(from string, msg *SMessage) error {
	switch t := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, t)
	}

	return nil
}

func (s *Server) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	if _, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size)); err != nil {
		return err
	}

	peer.(*p2p.TCPPeer).Wg.Done()
	return nil
}

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

func init() {
	gob.Register(MessageStoreFile{})
}
