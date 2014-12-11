package utils

import (
	"net"
	"sort"
	"testing"

	"github.com/vmihailenco/msgpack"
)

func TestNodeInfoMsgpack(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		t.Errorf("cannot marshal NodeInfo")
	}

	info := NodeInfo{ID: NewRandomNodeID([4]byte{1, 1, 1, 1}), Addr: addr}
	data, err := msgpack.Marshal(info)
	if err != nil {
		t.Errorf("cannot marshal NodeInfo")
	}
	var info2 NodeInfo
	err = msgpack.Unmarshal(data, &info2)
	if err != nil {
		t.Errorf("cannot unmarshal NodeInfo")
	}
	if info.ID.Digest.Cmp(info2.ID.Digest) != 0 {
		t.Errorf("node2.ID must be equal to node.ID")
	}
	if info.Addr.String() != info2.Addr.String() {
		t.Errorf("node2.Addr must be equal to node.Addr")
	}
}

func TestNodeInfoSort(t *testing.T) {
	ids := []string{
		"ZXF3qv5dsuaXy2AAoj6nTdYVZQ4TdUtcUp",
		"ZXF3qv3siKMSbZck8Ek6cMyMvrUmveitZE",
		"ZXF3qv3iH4GEeFuCgQVQns5Lrnr1ks7bCT",
		"ZXF3qv2DUw58cym8cpzn6Bg7re7P3xSWXp",
		"ZXF3qv5kFtmuXarw5t7DJkdTqQn9TkfSAJ",
		"ZXF3qv3i8q4y3UGboNW89n2bRy95ngwuDT",
		"ZXF3qv53NguoqoAhhnVkJRdCk6BNcJm9yE",
		"ZXF3qv5M1U8oaByBX3YoGHQxfWctgnZmzR",
		"ZXF3qv5oMTEha58tXa5ky7QovHS6fc9Atx",
		"ZXF3qv4Bywkot1zbRLWmGDKkivUfYyp6H5",
		"ZXF3qv4gsmv8f5twDGYZyHiB1HcT1NFypb",
		"ZXF3qv5y6o4VeFaoTjvLLwQ5bkWLahukr8",
		"ZXF3qv2rc6UEg2sh5Y3mzNY6oUbWukiHCp",
		"ZXF3qv3YwL8GcWMix817XPvP82dH2WgUiq",
		"ZXF3qv3AhuHsm4fBFSJege3Fk8XHnpJnd8",
		"ZXF3qv3n1vcYmHBYUUJ1mtGkt9PfcCuZCa",
		"ZXF3qv2RLQa9M5Q74nJDgLYo5UuXR17iv1",
		"ZXF3qv5yUgLsHMnB9qBQJ3pJoiWf3WsR5L",
		"ZXF3qv5FgfyAVf5zU1HiuWB2f9J2BpCBmw",
		"ZXF3qv5mkNeMvaPHNmga7mwSBXAmX6khV6",
	}

	sorted := []string{
		"ZXF3qv5dsuaXy2AAoj6nTdYVZQ4TdUtcUp",
		"ZXF3qv5kFtmuXarw5t7DJkdTqQn9TkfSAJ",
		"ZXF3qv5oMTEha58tXa5ky7QovHS6fc9Atx",
		"ZXF3qv5mkNeMvaPHNmga7mwSBXAmX6khV6",
		"ZXF3qv5FgfyAVf5zU1HiuWB2f9J2BpCBmw",
		"ZXF3qv5y6o4VeFaoTjvLLwQ5bkWLahukr8",
		"ZXF3qv5yUgLsHMnB9qBQJ3pJoiWf3WsR5L",
		"ZXF3qv5M1U8oaByBX3YoGHQxfWctgnZmzR",
		"ZXF3qv4gsmv8f5twDGYZyHiB1HcT1NFypb",
		"ZXF3qv4Bywkot1zbRLWmGDKkivUfYyp6H5",
		"ZXF3qv53NguoqoAhhnVkJRdCk6BNcJm9yE",
		"ZXF3qv3siKMSbZck8Ek6cMyMvrUmveitZE",
		"ZXF3qv3i8q4y3UGboNW89n2bRy95ngwuDT",
		"ZXF3qv3iH4GEeFuCgQVQns5Lrnr1ks7bCT",
		"ZXF3qv3n1vcYmHBYUUJ1mtGkt9PfcCuZCa",
		"ZXF3qv3AhuHsm4fBFSJege3Fk8XHnpJnd8",
		"ZXF3qv3YwL8GcWMix817XPvP82dH2WgUiq",
		"ZXF3qv2rc6UEg2sh5Y3mzNY6oUbWukiHCp",
		"ZXF3qv2DUw58cym8cpzn6Bg7re7P3xSWXp",
		"ZXF3qv2RLQa9M5Q74nJDgLYo5UuXR17iv1",
	}

	ary := make([]NodeInfo, len(ids))
	for i := range ary {
		id, _ := NewNodeIDFromString(ids[i])
		ary[i] = NodeInfo{ID: id}
	}

	id, _ := NewNodeIDFromString(ids[0])
	sorter := NodeInfoSorter{Nodes: ary, ID: id}
	sort.Sort(sorter)

	for i, n := range sorter.Nodes {
		id, err := NewNodeIDFromString(sorted[i])
		if err != nil {
			t.Fatal(err)
		}
		if id.Digest.Cmp(n.ID.Digest) != 0 {
			t.Errorf("sorter.nodes[%d] expects %s", i, sorted[i])
		}
	}
}
