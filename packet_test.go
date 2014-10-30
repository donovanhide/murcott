package murcott

import (
	"testing"

	"github.com/h2so5/murcott/utils"
)

func TestPacketSignature(t *testing.T) {
	packet := packet{
		Dst:     murcott.NewRandomNodeID(),
		Src:     murcott.NewRandomNodeID(),
		Type:    "dht",
		Payload: []byte("payload"),
	}

	key := murcott.GeneratePrivateKey()
	packet.sign(key)

	if !packet.verify(&key.PublicKey) {
		t.Errorf("varification failed")
	}
}
