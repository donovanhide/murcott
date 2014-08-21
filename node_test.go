package murcott

import (
	"testing"
)

func TestNodeMessageExchange(t *testing.T) {
	logger := newLogger()
	msg := "The quick brown fox jumps over the lazy dog"

	node1 := newNode(GeneratePrivateKey(), logger)
	node2 := newNode(GeneratePrivateKey(), logger)

	node1.sendMessage(node2.info.Id, []byte(msg))

	id, data, err := node2.recvMessage()
	if err != nil {
		t.Errorf("node2: recvMessage() returns error")
	}
	if id.Cmp(node1.info.Id) != 0 {
		t.Errorf("node2: wrong source id")
	}
	if string(data) != msg {
		t.Errorf("node2: wrong message body")
	}

	node2.sendMessage(node1.info.Id, []byte(msg))

	id, data, err = node1.recvMessage()
	if err != nil {
		t.Errorf("node1: recvMessage() returns error")
	}
	if id.Cmp(node2.info.Id) != 0 {
		t.Errorf("node1: wrong source id")
	}
	if string(data) != msg {
		t.Errorf("node1: wrong message body")
	}

	node1.close()
	node2.close()
}
