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

func NewNodeId(data []byte) NodeId {
	i := big.NewInt(0)
	i.SetBytes(data)
	return NodeId{*i}
}

func NewNodeIdFromString(str string) NodeId {
	i, err := base58.DecodeToBig([]byte(str))
	if err != nil {
		panic(err)
	}
	return NodeId{*i}
}

func NewRandomNodeId() NodeId {
	data := make([]byte, 20)
	_, err := rand.Read(data)
	if err != nil {
		panic(err)
	} else {
		return NewNodeId(data)
	}
}

func (p *NodeId) Xor(n NodeId) NodeId {
	d := big.NewInt(0)
	return NodeId{i: *d.Xor(&p.i, &n.i)}
}

func (p *NodeId) BitLen() int {
	return p.i.BitLen()
}

func (p *NodeId) Bit(i int) uint {
	return p.i.Bit(i)
}

func (p *NodeId) Cmp(n NodeId) int {
	return p.i.Cmp(&n.i)
}

func (p *NodeId) String() string {
	return string(base58.EncodeBig(nil, &p.i))
}
