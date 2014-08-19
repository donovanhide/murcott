package murcott

import (
	//	"math/big"
	"github.com/vmihailenco/msgpack"
	"testing"
)

func TestNodeIdMsgpack(t *testing.T) {
	id := NewRandomNodeId()
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
	id := NewRandomNodeId()
	str := id.String()
	id2 := NewNodeIdFromString(str)
	if id.Cmp(id2) != 0 {
		t.Errorf("cannot parse NodeId string")
	}
}
