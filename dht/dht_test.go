package dht

import (
	"net"
	"testing"
	"time"

	"github.com/h2so5/murcott/log"
	"github.com/h2so5/murcott/utils"
	"github.com/h2so5/utp"
)

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

	node1 := utils.NodeInfo{ID: utils.NewRandomNodeID(), Addr: addr1}
	node2 := utils.NodeInfo{ID: utils.NewRandomNodeID(), Addr: addr2}

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
		id, _ := utils.NewNodeIDFromString(ids[i])
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
		d := NewDHT(20, node.ID, utp.RawConn, logger)
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

	kvs := map[string]string{
		"a": "b",
		"c": "d",
		"e": "f",
		"g": "h",
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
