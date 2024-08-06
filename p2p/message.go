package p2p

const (
	IncomingStream  = 0x2
	IncomingMessage = 0x1
)

// Message represents any arbitraty data being sent over each transport bw two nodes
type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}
