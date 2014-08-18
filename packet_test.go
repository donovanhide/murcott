package murcott

import (
	"testing"
)

func TestPacketSignature(t *testing.T) {
	packet := packet{
		Dst:     NewRandomNodeId(),
		Src:     NewRandomNodeId(),
		Type:    "dht",
		Payload: []byte("payload"),
	}

	key := GeneratePrivateKey()
	packet.sign(key)

	if !packet.verify(&key.PublicKey) {
		t.Errorf("varification failed")
	}
}
