package router

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

	router1, err := NewRouter(key1, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router1.Discover(utils.DefaultConfig.Bootstrap())
	router1.SendMessage(key2.PublicKeyHash(), []byte(msg))

	router2, err := NewRouter(key2, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router2.Discover(utils.DefaultConfig.Bootstrap())
	m, err := router2.RecvMessage()
	if err != nil {
		t.Errorf("router2: recvMessage() returns error")
	}
	if m.ID.Cmp(router1.info.ID) != 0 {
		t.Errorf("router2: wrong source id")
	}
	if string(m.Payload) != msg {
		t.Errorf("router2: wrong message body")
	}

	router2.SendMessage(router1.info.ID, []byte(msg))

	m, err = router1.RecvMessage()
	if err != nil {
		t.Errorf("router1: recvMessage() returns error")
	}
	if m.ID.Cmp(router2.info.ID) != 0 {
		t.Errorf("router1: wrong source id")
	}
	if string(m.Payload) != msg {
		t.Errorf("router1: wrong message body")
	}

	router1.Close()
	router2.Close()
}

func TestRouterCancelMessage(t *testing.T) {
	logger := log.NewLogger()
	msg1 := "The quick brown fox jumps over the lazy dog"
	msg2 := "Grumpy Wizards make toxic brew for the Evil"

	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()

	router1, err := NewRouter(key1, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router1.Discover(utils.DefaultConfig.Bootstrap())
	id := router1.SendMessage(key2.PublicKeyHash(), []byte(msg1))
	router1.SendMessage(key2.PublicKeyHash(), []byte(msg2))
	router1.CancelMessage(id)

	router2, err := NewRouter(key2, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router2.Discover(utils.DefaultConfig.Bootstrap())

	m, err := router2.RecvMessage()
	if err != nil {
		t.Errorf("router2: recvMessage() returns error")
	}
	if m.ID.Cmp(router1.info.ID) != 0 {
		t.Errorf("router2: wrong source id")
	}
	if string(m.Payload) != msg2 {
		t.Errorf("router2: wrong message body")
	}

	router1.Close()
	router2.Close()
}

func TestRouterRouteExchange(t *testing.T) {
	logger := log.NewLogger()
	msg := "The quick brown fox jumps over the lazy dog"

	key1 := utils.GeneratePrivateKey()
	key2 := utils.GeneratePrivateKey()
	key3 := utils.GeneratePrivateKey()

	router1, err := NewRouter(key1, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router1.Discover(utils.DefaultConfig.Bootstrap())

	router2, err := NewRouter(key2, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	router2.Discover(utils.DefaultConfig.Bootstrap())

	time.Sleep(100 * time.Millisecond)
	router3, err := NewRouter(key3, logger, utils.DefaultConfig)
	if err != nil {
		t.Fatal(err)
	}
	addr, _ := net.ResolveUDPAddr("udp", router1.conn.LocalAddr().String())
	router3.Discover([]net.UDPAddr{net.UDPAddr{Port: addr.Port, IP: net.ParseIP("127.0.0.1")}})

	time.Sleep(100 * time.Millisecond)
	router3.SendMessage(key2.PublicKeyHash(), []byte(msg))

	m, err := router2.RecvMessage()
	if err != nil {
		t.Errorf("router2: recvMessage() returns error")
	}
	if m.ID.Cmp(router3.info.ID) != 0 {
		t.Errorf("router2: wrong source id")
	}
	if string(m.Payload) != msg {
		t.Errorf("router2: wrong message body")
	}

	router1.Close()
	router2.Close()
	router3.Close()
}
