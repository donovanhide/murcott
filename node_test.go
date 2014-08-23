package murcott

import (
	"testing"
)

func TestNodeChatMessage(t *testing.T) {
	logger := newLogger()
	key1 := GeneratePrivateKey()
	key2 := GeneratePrivateKey()
	node1 := newNode(key1, logger)
	node2 := newNode(key2, logger)
	plainmsg := NewPlainChatMessage("Hello")

	success := make(chan bool)

	node2.handle(func(src NodeId, msg interface{}) interface{} {
		if m, ok := msg.(ChatMessage); ok {
			if m.Text() == plainmsg.Text() {
				if src.cmp(key1.PublicKeyHash()) == 0 {
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
