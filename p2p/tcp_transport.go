package p2p

import (
	"fmt"
	"net"
)

// TCPPeer is the representation of the remote node over a tcp conn.
type TCPPeer struct {
	conn net.Conn
	// if we dial and retrieve a conn, outbound is true
	// if we accept and retrieve a conn, outbound is false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

type TCPTransportOpts struct {
	ListenAddr string
	HandshakeFunc
	Decoder Decoder
	OnPeer  func(Peer) error
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

func (t *TCPTransport) Consume() <-chan Message {
	return t.msgChan
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)

	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
		}
		go t.handleConn(conn)
	}
}

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	peer := NewTCPPeer(conn, true)

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
		t.msgChan <- msg
	}

}
