package murcott

import (
	"testing"
)

func TestRouterMessageExchange(t *testing.T) {
	logger := newLogger()
	msg := "The quick brown fox jumps over the lazy dog"

	router1 := newRouter(GeneratePrivateKey(), logger)
	router2 := newRouter(GeneratePrivateKey(), logger)

	router1.sendMessage(router2.info.Id, []byte(msg))

	m, err := router2.recvMessage()
	if err != nil {
		t.Errorf("router2: recvMessage() returns error")
	}
	if m.id.cmp(router1.info.Id) != 0 {
		t.Errorf("router2: wrong source id")
	}
	if string(m.payload) != msg {
		t.Errorf("router2: wrong message body")
	}

	router2.sendMessage(router1.info.Id, []byte(msg))

	m, err = router1.recvMessage()
	if err != nil {
		t.Errorf("router1: recvMessage() returns error")
	}
	if m.id.cmp(router2.info.Id) != 0 {
		t.Errorf("router1: wrong source id")
	}
	if string(m.payload) != msg {
		t.Errorf("router1: wrong message body")
	}

	router1.close()
	router2.close()
}