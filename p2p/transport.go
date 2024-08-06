package p2p

import "net"

// Peer represents a node in the distributed system
type Peer interface {
	net.Conn
	Send([]byte) error
	CloseStream()
}

// Transport takes care of communication bw nodes/peers. This can be
// TCP/P2P
type Transport interface {
	Addr() string
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
