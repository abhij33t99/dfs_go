package main

import (
	"fmt"
	"log"

	"github.com/abhij33t99/dfs_go/p2p"
)

type ServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
}

type Server struct {
	ServerOpts
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
	}
}

func (s *Server) Stop() {
	close(s.quitch)
}

func (s *Server) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action")
		s.Transport.Close()
	}()
	
	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Println(msg)
		case <-s.quitch:
			return
		}
	}
}

func (s *Server) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	s.loop()

	return nil
}
