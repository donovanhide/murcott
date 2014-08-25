package murcott

import (
	"testing"
)

func TestClientMessage(t *testing.T) {
	key1 := GeneratePrivateKey()
	key2 := GeneratePrivateKey()
	client1 := NewClient(key1, NewStorage(":memory:"))
	client2 := NewClient(key2, NewStorage(":memory:"))

	success := make(chan bool)
	plainmsg := NewPlainChatMessage("Hello")

	client2.HandleMessages(func(src NodeId, msg ChatMessage) {
		if src.cmp(key1.PublicKeyHash()) == 0 {
			if msg.Text() == plainmsg.Text() {
				success <- true
			} else {
				t.Errorf("wrong message body")
				success <- false
			}
		} else {
			t.Errorf("wrong source id")
			success <- false
		}
	})

	client1.SendMessage(key2.PublicKeyHash(), plainmsg, func(ok bool) {
		if ok {
			success <- true
		} else {
			t.Errorf("SendMessage() timed out")
			success <- false
		}
	})

	go client1.Run()
	go client2.Run()

	for i := 0; i < 2; i++ {
		if !<-success {
			return
		}
	}

	client1.Close()
	client2.Close()
}
