package murcott

import (
	"crypto/rand"
	"github.com/tv42/base58"
	"github.com/vmihailenco/msgpack"
	"math/big"
	"reflect"
)

func init() {
	msgpack.Register(reflect.TypeOf(NodeId{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			id := v.Interface().(NodeId)
			return e.EncodeBytes(id.i.Bytes())
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			b, err := d.DecodeBytes()
			if err != nil {
				return nil
			}
			i := big.NewInt(0)
			i.SetBytes(b)
			if i.BitLen() > 160 {
				return nil
			}
			v.Set(reflect.ValueOf(NodeId{*i}))
			return nil
		})
}

type NodeId struct {
	i big.Int
}

func NewNodeId(data [20]byte) NodeId {
	i := big.NewInt(0)
	i.SetBytes(data[:])
	return NodeId{*i}
}

func NewNodeIdFromString(str string) (NodeId, error) {
	i, err := base58.DecodeToBig([]byte(str))
	if err != nil {
		return NodeId{}, err
	}
	return NodeId{*i}, nil
}

func NewRandomNodeId() NodeId {
	var data [20]byte
	_, err := rand.Read(data[:])
	if err != nil {
		panic(err)
	} else {
		return NewNodeId(data)
	}
}

func (id NodeId) Xor(n NodeId) NodeId {
	d := big.NewInt(0)
	return NodeId{i: *d.Xor(&id.i, &n.i)}
}

func (id NodeId) BitLen() int {
	return 160
}

func (id NodeId) Bit(i int) uint {
	return id.i.Bit(159 - i)
}

func (id NodeId) Cmp(n NodeId) int {
	return id.i.Cmp(&n.i)
}

func (id NodeId) Bytes() []byte {
	return id.i.Bytes()
}

func (id NodeId) String() string {
	return string(base58.EncodeBig(nil, &id.i))
}
