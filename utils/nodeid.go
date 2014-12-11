package utils

import (
	"crypto/rand"
	"errors"
	"math/big"
	"reflect"

	"github.com/tv42/base58"
	"github.com/vmihailenco/msgpack"
)

func init() {
	msgpack.Register(reflect.TypeOf(NodeID{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			id := v.Interface().(NodeID)
			return e.EncodeBytes(id.Digest[:])
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			b, err := d.DecodeBytes()
			if err != nil {
				return err
			}
			var digest PublicKeyDigest
			if len(b) > len(digest) {
				return errors.New("too long id")
			}
			copy(digest[:], b)
			v.Set(reflect.ValueOf(NodeID{Digest: digest}))
			return nil
		})
}

type PublicKeyDigest [20]byte

// NodeID represents a 160-bit node identifier.
type NodeID struct {
	Digest PublicKeyDigest
}

// NewNodeID generates NodeID from the given big-endian byte array.
func NewNodeID(data PublicKeyDigest) NodeID {
	return NodeID{Digest: data}
}

// NewNodeIDFromString generates NodeID from the given base58-encoded string.
func NewNodeIDFromString(str string) (NodeID, error) {
	i, err := base58.DecodeToBig([]byte(str))
	if err != nil {
		return NodeID{}, err
	}
	var digest PublicKeyDigest
	copy(digest[:], i.Bytes())
	return NodeID{Digest: digest}, nil
}

func NewRandomNodeID() NodeID {
	var data PublicKeyDigest
	_, err := rand.Read(data[:])
	if err != nil {
		panic(err)
	} else {
		return NewNodeID(data)
	}
}

// Bytes returns identifier as a big-endian byte array.
func (id NodeID) Bytes() []byte {
	return id.Digest[:]
}

// String returns identifier as a base58-encoded byte array.
func (id NodeID) String() string {
	var i big.Int
	i.SetBytes(id.Digest[:])
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
