package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathTransPortFunc(t *testing.T) {
	
}

func TestTCPTransport(t *testing.T) {
	opts := TCPTransportOpts{
		ListenAddr: ":4000",
		HandshakeFunc: NOPHandshakeFunc,
		Decoder: DefaultDecoder{},
	}
	tr := NewTCPTransport(opts)

	assert.Equal(t, opts.ListenAddr, tr.ListenAddr)

	assert.Nil(t, tr.ListenAndAccept())
}
