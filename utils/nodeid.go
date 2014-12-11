package utils

import (
	"errors"
	"math/big"
	"reflect"

	"github.com/tv42/base58"
	"github.com/vmihailenco/msgpack"
)

const NodeIDPrefix = 144

func init() {
	msgpack.Register(reflect.TypeOf(NodeID{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			id := v.Interface().(NodeID)
			return e.EncodeBytes(id.Bytes())
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			b, err := d.DecodeBytes()
			if err != nil {
				return err
			}
			id, err := NewNodeIDFromBytes(b)
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(id))
			return nil
		})
}

type PublicKeyDigest [20]byte
type Namespace [4]byte

// NodeID represents a 160-bit node identifier.
type NodeID struct {
	Digest PublicKeyDigest
	NS     Namespace
}

// NewNodeID generates NodeID from the given namespace and publickey digest.
func NewNodeID(ns Namespace, data PublicKeyDigest) NodeID {
	return NodeID{NS: ns, Digest: data}
}

// NewNodeIDFromBytes generates NodeID from the given big-endian byte array.
func NewNodeIDFromBytes(b []byte) (NodeID, error) {
	var ns Namespace
	if len(b)-1 < len(ns) {
		return NodeID{}, errors.New("too short bytes")
	}
	b = b[1:]
	var digest PublicKeyDigest
	if len(b) > len(ns)+len(digest) {
		return NodeID{}, errors.New("too long bytes")
	}
	copy(ns[:], b[:])
	l := len(digest) - (len(b) - len(ns))
	copy(digest[l:], b[len(ns):])
	return NodeID{NS: ns, Digest: digest}, nil
}

// NewNodeIDFromString generates NodeID from the given base58-encoded string.
func NewNodeIDFromString(str string) (NodeID, error) {
	i, err := base58.DecodeToBig([]byte(str))
	if err != nil {
		return NodeID{}, err
	}
	return NewNodeIDFromBytes(i.Bytes())
}

func NewRandomNodeID(ns Namespace) NodeID {
	return NewNodeID(ns, GeneratePrivateKey().Digest())
}

// Bytes returns identifier as a big-endian byte array.
func (id NodeID) Bytes() []byte {
	return append([]byte{NodeIDPrefix}, append(id.NS[:], id.Digest[:]...)...)
}

// String returns identifier as a base58-encoded byte array.
func (id NodeID) String() string {
	var i big.Int
	i.SetBytes(id.Bytes())
	return string(base58.EncodeBig(nil, &i))
}

func (d PublicKeyDigest) Xor(n PublicKeyDigest) PublicKeyDigest {
	var b, c big.Int
	b.SetBytes(d[:])
	c.SetBytes(n[:])
	b.Xor(&b, &c)
	var e PublicKeyDigest
	copy(e[:], b.Bytes())
	return e
}

func (d PublicKeyDigest) BitLen() int {
	return len(d) * 8
}

func (d PublicKeyDigest) Bit(i int) uint {
	var b big.Int
	b.SetBytes(d[:])
	return b.Bit(len(d)*8 - 1 - i)
}

func (d PublicKeyDigest) Cmp(n PublicKeyDigest) int {
	var b, c big.Int
	b.SetBytes(d[:])
	c.SetBytes(n[:])
	return b.Cmp(&c)
}

func (d PublicKeyDigest) Log2int() int {
	var b big.Int
	b.SetBytes(d[:])
	b.Add(&b, big.NewInt(1))
	l := len(d)*8 - 1
	for i := len(d) * 8; i >= 0 && b.Bit(i) == 0; i-- {
		l--
	}
	if l < 0 {
		return 0
	}
	return l
}
