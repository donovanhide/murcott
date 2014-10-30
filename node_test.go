// +build !race

package murcott

import (
	"testing"

	"github.com/h2so5/murcott/utils"
)

func TestNodeChatMessage(t *testing.T) {
	logger := newLogger()
	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()
	node1, err := newNode(key1, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	node2, err := newNode(key2, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	plainmsg := NewPlainChatMessage("Hello")

	success := make(chan bool)

	node2.handle(func(src utils.NodeID, msg interface{}) interface{} {
		if m, ok := msg.(ChatMessage); ok {
			if m.Text() == plainmsg.Text() {
				if src.Cmp(key1.PublicKeyHash()) == 0 {
					success <- true
				} else {
					t.Errorf("wrong source id")
					success <- false
				}
			} else {
				t.Errorf("wrong message body")
				success <- false
			}
		} else {
			t.Errorf("wrong message type")
			success <- false
		}
		return messageAck{}
	})

	node1.send(key2.PublicKeyHash(), plainmsg, func(msg interface{}) {
		if _, ok := msg.(messageAck); ok {
			success <- true
		} else {
			t.Errorf("wrong ack type")
			success <- false
		}
	})

	go node1.run()
	go node2.run()

	for i := 0; i < 2; i++ {
		if !<-success {
			return
		}
	}

	node1.close()
	node2.close()
}
