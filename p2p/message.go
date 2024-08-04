package p2p

// Message represents any arbitraty data being sent over each transport bw two nodes
type Message struct {
	From    string
	Payload []byte
}
