package utils

import (
	"testing"

	"github.com/vmihailenco/msgpack"
)

func TestNodeIDMsgpack(t *testing.T) {
	id := NewRandomNodeID([4]byte{1, 1, 1, 1})
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
	id := NewRandomNodeID([4]byte{1, 1, 1, 1})
	str := id.String()
	id2, err := NewNodeIDFromString(str)
	if err != nil || id.Digest.Cmp(id2.Digest) != 0 {
		t.Errorf("failed to generate NodeID from string")
	}
}
