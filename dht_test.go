package murcott

import (
	"net"
	"reflect"
	"testing"
)

func TestDhtPing(t *testing.T) {
	node1 := nodeInfo{Id: NewRandomNodeId(), Addr: nil}
	node2 := nodeInfo{Id: NewRandomNodeId(), Addr: nil}

	dht1 := newDht(10, node1, NewLogger())
	dht2 := newDht(10, node2, NewLogger())
	go dht1.run()
	go dht2.run()

	dht1.addNode(node2)

	dht1ch := dht1.rpcChannel()
	exit := make(chan bool)

	go func() {
		select {
		case p := <-dht1ch:
			if p.Dst.Cmp(node2.Id) != 0 {
				t.Errorf("wrong packet destination: %s", p.Dst)
				exit <- true
			} else {
				dht2.processPacket(node1, p.Payload)
				exit <- true
			}
		default:
			exit <- false
		}
	}()

	if <-exit == false {
		t.Errorf("dht1 never makes PING packet")
	}

	if dht1.getNodeInfo(node2.Id) == nil {
		t.Errorf("dht1 doesn't know node2")
	}

	if dht2.getNodeInfo(node1.Id) == nil {
		t.Errorf("dht2 doesn't know node1")
	}

	dht1.close()
	dht2.close()
}

func TestDhtGroup(t *testing.T) {
	logger := NewLogger()

	n := 20
	dhtmap := make(map[string]*dht)
	idary := make([]nodeInfo, n)
	chans := make([]<-chan dhtPacket, n)

	ids := []string{
		"2R2eoXNPEhbmhx7aNqgY1e2SdKrJ",
		"4cLuxzdqZgKCatw2HJqEoZEAhkdD",
		"4fmJMvhoXrmBrHdeZnQ5iX5ropm3",
		"4fqqyXWVWmBRnLUVHfZgzjKdtFcd",
		"218GStqPqa7iLzLsAQBS9eZRrUik",
		"2vm8ByjrLATzFR6qqEHCdwua6eCf",
		"3nvgbcBzvt9y9Uvf1AbwVfnqV2RG",
		"33m8NJkskAUdCGw3uYAxeBD5jjBY",
		"3ru66Gjzx2cDuddRTzA47yMqEoLE",
		"2S68uiyhVt5c59zgXh1mj3v8vThp",
		"43eYKjPkMX3gqqWuzzBYvejLSQgJ",
		"2EhubMbxHHTHSdsLUuJmNpvRakt6",
		"hVrmqGmWDeRWcWVwTxMBEr1pszM",
		"3Fx6deQbP8arwtVAxbcts5d9KaTw",
		"36gDQzwABf2bscJwTjw9y2UU8Adg",
		"dT378JwadJ4h7HTgeh8UkMgAuVm",
		"37ZrjbsRymbaCD14mUu6FX3nHnPF",
		"3qtQPy3WCq3sx4vhGW1vR46aRRSo",
		"2Af1fPjeQ8jdtsHrrRxZCfJNBWGr",
		"3HJJyARx667UUrwoEDzCJAMx6tMg",
	}

	for i := 0; i < n; i++ {
		id := NewNodeIdFromString(ids[i])
		addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:4000")
		node := nodeInfo{Id: id, Addr: addr}
		dht := newDht(20, node, logger)
		idary[i] = node
		chans[i] = dht.rpcChannel()
		go dht.run()
		dhtmap[id.String()] = dht
	}

	rootNode := idary[0]
	rootDht := dhtmap[rootNode.Id.String()]

	trans := func() {
		cases := make([]reflect.SelectCase, len(chans)+1)
		for i, ch := range chans {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
		}
		cases[len(cases)-1] = reflect.SelectCase{Dir: reflect.SelectDefault}

		for {
			chosen, value, ok := reflect.Select(cases)
			if !ok || chosen == len(cases)-1 {
				break
			}
			p := value.Interface().(dhtPacket)
			id := p.Dst.String()
			dht := dhtmap[id]
			dht.processPacket(idary[chosen], p.Payload)
		}
	}

	exit := make(chan struct{})
	go func() {
		for {
			select {
			case <-exit:
				return
			default:
				trans()
			}
		}
	}()

	for _, d := range dhtmap {
		d.addNode(rootNode)
		d.findNearestNode(d.selfnode.Id)
	}

	kvs := map[string]string{
		"a": "b",
		"c": "d",
		"e": "f",
		"g": "h",
	}

	for k, v := range kvs {
		rootDht.storeValue(k, v)
	}

	for _, d := range dhtmap {
		for k, _ := range kvs {
			val := d.loadValue(k)
			if val == nil {
				t.Errorf("key not found: %s", k)
			} else if *val != kvs[k] {
				t.Errorf("wrong value for the key: %s : %s; %s expected", k, val, kvs[k])
			}
		}
	}

	exit <- struct{}{}

	for _, d := range dhtmap {
		d.close()
	}
}
