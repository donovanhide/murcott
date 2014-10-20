package murcott

import (
	"math/big"
	"testing"
)

func TestNodeTableInsertRemove(t *testing.T) {
	b := big.NewInt(int64(0))
	var id [20]byte
	copy(id[:], b.Bytes()[:])
	selfid := NewNodeID(id)
	n := newNodeTable(50, selfid)

	ary := make([]NodeID, 100)

	for i := 0; i < len(ary); i++ {
		b.Add(b, big.NewInt(int64(1)))
		var id [20]byte
		copy(id[:], b.Bytes()[:])
		node := NewNodeID(id)
		ary[i] = node
	}

	for _, id := range ary {
		n.insert(nodeInfo{ID: id, Addr: nil})
	}

	for _, id := range ary {
		if n.find(id) == nil {
			t.Errorf("%s not found", id.String())
		}
	}

	for _, id := range ary {
		n.remove(id)
		if n.find(id) != nil {
			t.Errorf("%s should be removed", id.String())
		}
	}
}
