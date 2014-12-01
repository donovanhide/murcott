package murcott

import (
	"net"
	"testing"
	"time"

	"github.com/h2so5/murcott/utils"

	"github.com/h2so5/murcott/log"
)

func TestRouterMessageExchange(t *testing.T) {
	logger := log.NewLogger()
	msg := "The quick brown fox jumps over the lazy dog"

	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()

	router1, err := newRouter(key1, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router1.discover(DefaultConfig.getBootstrap())
	router1.sendMessage(key2.PublicKeyHash(), []byte(msg))

	router2, err := newRouter(key2, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router2.discover(DefaultConfig.getBootstrap())
	m, err := router2.recvMessage()
	if err != nil {
		t.Errorf("router2: recvMessage() returns error")
	}
	if m.id.Cmp(router1.info.ID) != 0 {
		t.Errorf("router2: wrong source id")
	}
	if string(m.payload) != msg {
		t.Errorf("router2: wrong message body")
	}

	router2.sendMessage(router1.info.ID, []byte(msg))

	m, err = router1.recvMessage()
	if err != nil {
		t.Errorf("router1: recvMessage() returns error")
	}
	if m.id.Cmp(router2.info.ID) != 0 {
		t.Errorf("router1: wrong source id")
	}
	if string(m.payload) != msg {
		t.Errorf("router1: wrong message body")
	}

	router1.close()
	router2.close()
}

func TestRouterCancelMessage(t *testing.T) {
	logger := log.NewLogger()
	msg1 := "The quick brown fox jumps over the lazy dog"
	msg2 := "Grumpy Wizards make toxic brew for the Evil"

	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()

	router1, err := newRouter(key1, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router1.discover(DefaultConfig.getBootstrap())
	id := router1.sendMessage(key2.PublicKeyHash(), []byte(msg1))
	router1.sendMessage(key2.PublicKeyHash(), []byte(msg2))
	router1.cancelMessage(id)

	router2, err := newRouter(key2, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router2.discover(DefaultConfig.getBootstrap())

	m, err := router2.recvMessage()
	if err != nil {
		t.Errorf("router2: recvMessage() returns error")
	}
	if m.id.Cmp(router1.info.ID) != 0 {
		t.Errorf("router2: wrong source id")
	}
	if string(m.payload) != msg2 {
		t.Errorf("router2: wrong message body")
	}

	router1.close()
	router2.close()
}

func TestRouterRouteExchange(t *testing.T) {
	logger := log.NewLogger()
	msg := "The quick brown fox jumps over the lazy dog"

	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()
	key3 := utils.GeneratePrivateKey()

	router1, err := newRouter(key1, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router1.discover(DefaultConfig.getBootstrap())

	router2, err := newRouter(key2, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router2.discover(DefaultConfig.getBootstrap())

	time.Sleep(100 * time.Millisecond)
	router3, err := newRouter(key3, logger, DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	addr, _ := net.ResolveUDPAddr("udp", router1.conn.LocalAddr().String())
	router3.discover([]net.UDPAddr{net.UDPAddr{Port: addr.Port, IP: net.ParseIP("127.0.0.1")}})

	time.Sleep(100 * time.Millisecond)
	router3.sendMessage(key2.PublicKeyHash(), []byte(msg))

	m, err := router2.recvMessage()
	if err != nil {
		t.Errorf("router2: recvMessage() returns error")
	}
	if m.id.Cmp(router3.info.ID) != 0 {
		t.Errorf("router2: wrong source id")
	}
	if string(m.payload) != msg {
		t.Errorf("router2: wrong message body")
	}

	router1.close()
	router2.close()
	router3.close()
}
