package murcott

import (
	"net"
	"testing"
)

func TestDhtPing(t *testing.T) {
	node1 := nodeInfo{Id: NewRandomNodeId(), Addr: nil}
	node2 := nodeInfo{Id: NewRandomNodeId(), Addr: nil}

	dht1 := newDht(10, node1, newLogger())
	dht2 := newDht(10, node2, newLogger())
	go dht1.run()
	go dht2.run()

	dht1.addNode(node2)

	dst, payload, err := dht1.nextPacket()
	if err != nil {
		return
	}
	if dst.Cmp(node2.Id) != 0 {
		t.Errorf("wrong packet destination: %s", dst.String())
	} else {
		dht2.processPacket(node1, payload)
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
	logger := newLogger()

	n := 20
	dhtmap := make(map[string]*dht)
	idary := make([]nodeInfo, n)

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
		id, _ := NewNodeIdFromString(ids[i])
		addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:4000")
		node := nodeInfo{Id: id, Addr: addr}
		d := newDht(20, node, logger)
		idary[i] = node
		go d.run()
		dhtmap[id.String()] = d
		go func(d *dht) {
			for {
				dst, payload, err := d.nextPacket()
				if err != nil {
					return
				}
				id := dst.String()
				dht := dhtmap[id]
				dht.processPacket(d.info, payload)
			}
		}(d)
	}

	rootNode := idary[0]
	rootDht := dhtmap[rootNode.Id.String()]

	for _, d := range dhtmap {
		d.addNode(rootNode)
		d.findNearestNode(d.info.Id)
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

	// TODO: close dhts correctly
}
