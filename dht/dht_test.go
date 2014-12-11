package dht

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/h2so5/murcott/log"
	"github.com/h2so5/murcott/utils"
	"github.com/h2so5/utp"
)

var namespace = [4]byte{1, 1, 1, 1}

func getLoopbackAddr(addr net.Addr) (*net.UDPAddr, error) {
	_, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, err
	}
	uaddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("::1", port))
	if err != nil {
		return nil, err
	}
	return uaddr, nil
}

func TestDhtPing(t *testing.T) {

	addr, err := utp.ResolveAddr("utp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	utp1, err := utp.Listen("utp", addr)
	if err != nil {
		t.Fatal(err)
	}
	utp2, err := utp.Listen("utp", addr)
	if err != nil {
		t.Fatal(err)
	}

	addr1, err := getLoopbackAddr(utp1.Addr())
	if err != nil {
		t.Fatal(err)
	}
	addr2, err := getLoopbackAddr(utp2.Addr())
	if err != nil {
		t.Fatal(err)
	}

	node1 := utils.NodeInfo{ID: utils.NewRandomNodeID(namespace), Addr: addr1}
	node2 := utils.NodeInfo{ID: utils.NewRandomNodeID(namespace), Addr: addr2}

	dht1 := NewDHT(10, node1.ID, utp1.RawConn, log.NewLogger())
	dht2 := NewDHT(10, node2.ID, utp2.RawConn, log.NewLogger())
	defer dht1.Close()
	defer dht2.Close()

	dht1.AddNode(node2)

	time.Sleep(time.Millisecond * 100)

	if dht1.GetNodeInfo(node2.ID) == nil {
		t.Errorf("dht1 should know node2")
	}

	if dht2.GetNodeInfo(node1.ID) == nil {
		t.Errorf("dht2 should know node1")
	}
}

func TestDhtGroup(t *testing.T) {
	logger := log.NewLogger()

	n := 20
	dhtmap := make(map[string]*DHT)
	idary := make([]utils.NodeInfo, n)

	for i := 0; i < n; i++ {
		id := utils.NewRandomNodeID(namespace)
		addr, err := utp.ResolveAddr("utp", ":0")
		if err != nil {
			t.Fatal(err)
		}
		utp, err := utp.Listen("utp", addr)
		uaddr, err := getLoopbackAddr(utp.Addr())
		if err != nil {
			t.Fatal(err)
		}
		node := utils.NodeInfo{ID: id, Addr: uaddr}
		d := NewDHT(10, node.ID, utp.RawConn, logger)
		idary[i] = node
		dhtmap[id.String()] = d
		defer d.Close()
	}

	rootNode := idary[0]
	rootDht := dhtmap[rootNode.ID.String()]

	for _, d := range dhtmap {
		d.AddNode(rootNode)
		d.FindNearestNode(d.id)
	}

	kvs := map[string]string{}
	for i := 0; i < 20; i++ {
		kvs[fmt.Sprintf("<%d>", i)] = utils.NewRandomNodeID(namespace).String()
	}

	for k, v := range kvs {
		rootDht.StoreValue(k, v)
	}

	for _, d := range dhtmap {
		for k := range kvs {
			val := d.LoadValue(k)
			if val == nil {
				t.Errorf("key not found: %s", k)
			} else if *val != kvs[k] {
				t.Errorf("wrong value for the key: %s : %s; %s expected", k, val, kvs[k])
			}
		}
	}
}
