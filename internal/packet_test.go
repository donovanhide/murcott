package internal

import (
	"testing"

	"github.com/h2so5/murcott/utils"
)

func TestPacketSignature(t *testing.T) {
	packet := Packet{
		Dst:     utils.NewRandomNodeID(),
		Src:     utils.NewRandomNodeID(),
		Type:    "dht",
		Payload: []byte("payload"),
	}

	key := utils.GeneratePrivateKey()
	packet.Sign(key)

	if !packet.Verify(&key.PublicKey) {
		t.Errorf("varification failed")
	}
}
