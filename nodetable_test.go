package murcott

import (
	"math/big"
	"testing"
)

func TestNodeTableInsertRemove(t *testing.T) {
	b := big.NewInt(int64(0))
	selfid := NewNodeId(b.Bytes())
	n := newNodeTable(50, selfid)

	ary := make([]NodeId, 100)

	for i := 0; i < len(ary); i++ {
		id := NewNodeId(b.Bytes())
		ary[i] = id
		b.Add(b, big.NewInt(int64(1)))
	}

	for _, id := range ary {
		n.insert(nodeInfo{Id: id, Addr: nil})
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
