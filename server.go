package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/abhij33t99/dfs_go/p2p"
)

type ServerOpts struct {
	ID                string
	EncKey            []byte
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

	if len(opts.ID) == 0 {
		opts.ID = generateId()
	}

	return &Server{
		ServerOpts: opts,
		store:      NewStore(storeOpts),
		quitch:     make(chan struct{}),
		peers:      make(map[string]p2p.Peer),
	}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
	ID   string
}

type MessageGetFile struct {
	Key string
	ID  string
}

func (s *Server) broadcast(msg *Message) error {
	msgBuff := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuff).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(msgBuff.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Get(key string) (io.Reader, error) {
	if s.store.Has(s.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", s.Transport.Addr(), key)
		_, r, err := s.store.Read(s.ID, key)
		return r, err
	}

	fmt.Printf("[%s] don't have the file (%s) locally, fetching from network...\n", s.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			Key: hashKey(key),
			ID:  s.ID,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range s.peers {
		// read the file size to limit the amount of bytes we read from the conn
		// to prevent hanging
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)
		n, err := s.store.WriteDecrypt(s.EncKey, s.ID, key, io.LimitReader(peer, fileSize))

		if err != nil {
			return nil, err
		}
		fmt.Printf("[%s] received {%d} bytes over the network from [%s]\n", s.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}

	_, r, err := s.store.Read(s.ID, key)
	return r, err
}

func (s *Server) Store(key string, r io.Reader) error {
	//store the data to disk
	//broadcast to all known peers in the network
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := s.store.Write(s.ID, key, tee)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] received and written [%d] bytes to disk\n", s.Transport.Addr(), size)

	msg := Message{
		Payload: MessageStoreFile{
			Key:  hashKey(key),
			Size: size + 16,
			ID:   s.ID,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})

	_, err = copyEncrypt(s.EncKey, fileBuffer, mw)
	if err != nil {
		return err
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
		log.Println("file server stopped due to error or user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error:", err)
			}

			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Println("handle message error:", err)
			}
		case <-s.quitch:
			return
		}
	}
}

func (s *Server) handleMessage(from string, msg *Message) error {
	switch t := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, t)
	case MessageGetFile:
		return s.handleMessageGetFile(from, t)
	}

	return nil
}

func (s *Server) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] needed file {%s} but it doesn't exist on disk", s.Transport.Addr(), msg.Key)
	}
	fmt.Printf("[%s] serving file (%s) over the network\n", s.Transport.Addr(), msg.Key)
	fileSize, r, err := s.store.Read(msg.ID, msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		defer rc.Close()
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in server", from)
	}
	// first send the incomingStream bytes to the peer and then the file size
	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes over the network to %s\n", s.Transport.Addr(), n, from)
	return nil
}

func (s *Server) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := s.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written %d bytes to disk\n", s.Transport.Addr(), n)

	// peer.(*p2p.TCPPeer).Wg.Done()
	peer.CloseStream()
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
	gob.Register(MessageGetFile{})
}
