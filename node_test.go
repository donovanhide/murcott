package murcott

import (
	"reflect"
	"testing"
)

func TestNodeChatMessage(t *testing.T) {
	key1 := GeneratePrivateKey()
	key2 := GeneratePrivateKey()
	node1 := NewNode(key1)
	node2 := NewNode(key2)
	body := "Hello"

	err := node1.Send(key2.PublicKeyHash(), ChatMessage{Body: body})
	if err != nil {
		t.Errorf("Send() error: %v", err)
	}

	id, msg, err := node2.Recv()
	if err != nil {
		t.Errorf("Recv() error: %v", err)
	}

	if id.cmp(key1.PublicKeyHash()) != 0 {
		t.Errorf("wrong source id")
	}

	if m, ok := msg.(ChatMessage); ok {
		if m.Body != body {
			t.Errorf("wrong message body: %s; expects %s", m.Body, body)
		}
	} else {
		t.Errorf("wrong message type: %v; expects ChatMessage", reflect.ValueOf(msg).Type())
	}

	node1.Close()
	node2.Close()
}
