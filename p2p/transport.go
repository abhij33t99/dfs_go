package p2p

// Peer represents a node in the distributed system
type Peer interface {
	Close() error
}

// Transport takes care of communication bw nodes/peers. This can be
// TCP/P2P
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan Message
}
