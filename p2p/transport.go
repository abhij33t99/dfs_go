package p2p

import "net"

// Peer represents a node in the distributed system
type Peer interface {
	net.Conn
	Send([]byte) error
}

// Transport takes care of communication bw nodes/peers. This can be
// TCP/P2P
type Transport interface {
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan Message
	Close() error
}
