package dht

import (
	"math/big"
	"testing"

	"github.com/h2so5/murcott/utils"
)

func TestNodeTableInsertRemove(t *testing.T) {
	b := big.NewInt(int64(0))
	var id [20]byte
	copy(id[:], b.Bytes()[:])
	selfid := utils.NewNodeID(id)
	n := newNodeTable(50, selfid)

	ary := make([]utils.NodeID, 100)

	for i := 0; i < len(ary); i++ {
		b.Add(b, big.NewInt(int64(1)))
		var id [20]byte
		copy(id[:], b.Bytes()[:])
		node := utils.NewNodeID(id)
		ary[i] = node
	}

	for _, id := range ary {
		n.insert(utils.NodeInfo{ID: id, Addr: nil})
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
