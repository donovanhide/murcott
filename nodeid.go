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

// NodeId represents a 160-bit node identifier.
type NodeId struct {
	i big.Int
}

// NewNodeId generates NodeId from the given big-endian byte array.
func NewNodeId(data [20]byte) NodeId {
	i := big.NewInt(0)
	i.SetBytes(data[:])
	return NodeId{*i}
}

// NewNodeIdFromString generates NodeId from the given base58-encoded string.
func NewNodeIdFromString(str string) (NodeId, error) {
	i, err := base58.DecodeToBig([]byte(str))
	if err != nil {
		return NodeId{}, err
	}
	return NodeId{*i}, nil
}

func newRandomNodeId() NodeId {
	var data [20]byte
	_, err := rand.Read(data[:])
	if err != nil {
		panic(err)
	} else {
		return NewNodeId(data)
	}
}

func (id NodeId) xor(n NodeId) NodeId {
	d := big.NewInt(0)
	return NodeId{i: *d.Xor(&id.i, &n.i)}
}

func (id NodeId) bitLen() int {
	return 160
}

func (id NodeId) bit(i int) uint {
	return id.i.Bit(159 - i)
}

func (id NodeId) cmp(n NodeId) int {
	return id.i.Cmp(&n.i)
}

// Bytes returns identifier as a big-endian byte array.
func (id NodeId) Bytes() []byte {
	return id.i.Bytes()
}

// String returns identifier as a base58-encoded byte array.
func (id NodeId) String() string {
	return string(base58.EncodeBig(nil, &id.i))
}
