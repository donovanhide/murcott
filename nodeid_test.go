package murcott

import (
	"testing"

	"github.com/vmihailenco/msgpack"
)

func TestNodeIDMsgpack(t *testing.T) {
	id := newRandomNodeID()
	data, err := msgpack.Marshal(id)
	if err != nil {
		t.Errorf("cannot marshal NodeID")
	}
	var id2 NodeID
	err = msgpack.Unmarshal(data, &id2)
	if err != nil {
		t.Errorf("cannot unmarshal NodeID")
	}
}

func TestNodeIDString(t *testing.T) {
	id := newRandomNodeID()
	str := id.String()
	id2, err := NewNodeIDFromString(str)
	if err != nil || id.cmp(id2) != 0 {
		t.Errorf("failed to generate NodeID from string")
	}
}
