package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPPeer is the representation of the remote node over a tcp conn.
type TCPPeer struct {
	// underlying conn of the peer which in this case is tcp conn
	net.Conn
	// if we dial and retrieve a conn, outbound is true
	// if we accept and retrieve a conn, outbound is false
	outbound bool

	Wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		Wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	msgChan  chan Message
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		msgChan:          make(chan Message),
	}
}

// func (t *TCPTransport) ListenAddress() string {
// 	return t.ListenAddr
// }

func (t *TCPTransport) Consume() <-chan Message {
	return t.msgChan
}

// Dial implements the transport interface
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go t.handleConn(conn, true)
	return nil
}

// Close the listener of the transport interface
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)

	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	log.Printf("TCP transport listening on port %s\n", t.ListenAddr)

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
		}
		// fmt.Printf("New incoming connection %+v\n", conn)
		go t.handleConn(conn, false)
	}
}

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error
	peer := NewTCPPeer(conn, outbound)

	defer func() {
		fmt.Printf("Dropping peer connection: %s", err)
		conn.Close()
	}()

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	// Read loop
	msg := Message{}
	for {
		err = t.Decoder.Decode(conn, &msg)
		if err != nil {
			fmt.Printf("TCP error: %s\n", err)
			return
		}
		msg.From = conn.RemoteAddr().String()
		peer.Wg.Add(1)
		fmt.Println("Waiting till stream is done")
		t.msgChan <- msg
		peer.Wg.Wait()
		fmt.Println("stream done, continuing read loop")
	}

}
