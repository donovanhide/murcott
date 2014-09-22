package murcott

import "testing"

func TestRouterMessageExchange(t *testing.T) {
	logger := newLogger()
	msg := "The quick brown fox jumps over the lazy dog"

	key1 := GeneratePrivateKey()
	key2 := GeneratePrivateKey()

	router1 := newRouter(key1, logger)
	router1.sendMessage(key2.PublicKeyHash(), []byte(msg))

	router2 := newRouter(key2, logger)
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

func TestRouterCancelMessage(t *testing.T) {
	logger := newLogger()
	msg1 := "The quick brown fox jumps over the lazy dog"
	msg2 := "Grumpy Wizards make toxic brew for the Evil"

	key1 := GeneratePrivateKey()
	key2 := GeneratePrivateKey()

	router1 := newRouter(key1, logger)
	id := router1.sendMessage(key2.PublicKeyHash(), []byte(msg1))
	router1.sendMessage(key2.PublicKeyHash(), []byte(msg2))
	router1.cancelMessage(id)

	router2 := newRouter(key2, logger)

	m, err := router2.recvMessage()
	if err != nil {
		t.Errorf("router2: recvMessage() returns error")
	}
	if m.id.cmp(router1.info.Id) != 0 {
		t.Errorf("router2: wrong source id")
	}
	if string(m.payload) != msg2 {
		t.Errorf("router2: wrong message body")
	}

	router1.close()
	router2.close()
}
