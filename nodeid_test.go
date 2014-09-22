package murcott

import (
	"testing"

	"github.com/vmihailenco/msgpack"
)

func TestNodeIdMsgpack(t *testing.T) {
	id := newRandomNodeId()
	data, err := msgpack.Marshal(id)
	if err != nil {
		t.Errorf("cannot marshal NodeId")
	}
	var id2 NodeId
	err = msgpack.Unmarshal(data, &id2)
	if err != nil {
		t.Errorf("cannot unmarshal NodeId")
	}
}

func TestNodeIdString(t *testing.T) {
	id := newRandomNodeId()
	str := id.String()
	id2, err := NewNodeIdFromString(str)
	if err != nil || id.cmp(id2) != 0 {
		t.Errorf("failed to generate NodeId from string")
	}
}
