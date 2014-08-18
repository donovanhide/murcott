package murcott

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"github.com/tv42/base58"
	"github.com/vmihailenco/msgpack"
	"math/big"
	"reflect"
)

type PublicKey struct {
	x, y *big.Int
}

type PrivateKey struct {
	PublicKey
	d *big.Int
}

type signature struct {
	r, s *big.Int
}

func (p *PublicKey) PublicKeyHash() NodeId {
	return NewNodeId(sha1.Sum(append(p.x.Bytes(), p.y.Bytes()...)))
}

func GeneratePrivateKey() *PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err == nil {
		return &PrivateKey{
			PublicKey: PublicKey{x: key.X, y: key.Y},
			d:         key.D,
		}
	} else {
		return nil
	}
}

func PrivateKeyFromString(str string) *PrivateKey {
	b, err := base58.DecodeToBig([]byte(str))
	if err != nil {
		return nil
	}
	var out PrivateKey
	err = msgpack.Unmarshal(b.Bytes(), &out)
	if err != nil {
		return nil
	}
	return &out
}

func (p *PrivateKey) String() string {
	data, _ := msgpack.Marshal(p)
	return string(base58.EncodeBig(nil, big.NewInt(0).SetBytes(data)))
}

func (p *PrivateKey) sign(data []byte) *signature {
	key := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: elliptic.P256(), X: p.x, Y: p.y},
		D:         p.d,
	}
	hash := sha1.Sum(data)
	r, s, err := ecdsa.Sign(rand.Reader, &key, hash[:])
	if err == nil {
		return &signature{r: r, s: s}
	} else {
		return nil
	}
}

func (p *PublicKey) verify(data []byte, sign *signature) bool {
	key := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     p.x,
		Y:     p.y,
	}
	hash := sha1.Sum(data)
	return ecdsa.Verify(&key, hash[:], sign.r, sign.s)
}

func init() {
	msgpack.Register(reflect.TypeOf(signature{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			sign := v.Interface().(signature)
			return e.Encode(map[string][]byte{
				"r": sign.r.Bytes(),
				"s": sign.s.Bytes(),
			})
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			i, err := d.DecodeMap()
			if err != nil {
				return nil
			}
			m := i.(map[interface{}]interface{})
			if r, ok := m["r"].(string); ok {
				if s, ok := m["s"].(string); ok {
					v.Set(reflect.ValueOf(signature{
						r: big.NewInt(0).SetBytes([]byte(r)),
						s: big.NewInt(0).SetBytes([]byte(s)),
					}))
				}
			}
			return nil
		})

	msgpack.Register(reflect.TypeOf(PrivateKey{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			sign := v.Interface().(PrivateKey)
			return e.Encode(map[string][]byte{
				"x": sign.x.Bytes(),
				"y": sign.y.Bytes(),
				"d": sign.d.Bytes(),
			})
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			i, err := d.DecodeMap()
			if err != nil {
				return nil
			}
			m := i.(map[interface{}]interface{})
			if x, ok := m["x"].(string); ok {
				if y, ok := m["y"].(string); ok {
					if d, ok := m["d"].(string); ok {
						v.Set(reflect.ValueOf(PrivateKey{
							PublicKey: PublicKey{
								x: big.NewInt(0).SetBytes([]byte(x)),
								y: big.NewInt(0).SetBytes([]byte(y)),
							},
							d: big.NewInt(0).SetBytes([]byte(d)),
						}))
					}
				}
			}
			return nil
		})

	msgpack.Register(reflect.TypeOf(PublicKey{}),
		func(e *msgpack.Encoder, v reflect.Value) error {
			sign := v.Interface().(PublicKey)
			return e.Encode(map[string][]byte{
				"x": sign.x.Bytes(),
				"y": sign.y.Bytes(),
			})
		},
		func(d *msgpack.Decoder, v reflect.Value) error {
			i, err := d.DecodeMap()
			if err != nil {
				return nil
			}
			m := i.(map[interface{}]interface{})
			if x, ok := m["x"].(string); ok {
				if y, ok := m["y"].(string); ok {
					v.Set(reflect.ValueOf(PublicKey{
						x: big.NewInt(0).SetBytes([]byte(x)),
						y: big.NewInt(0).SetBytes([]byte(y)),
					}))
				}
			}
			return nil
		})
}
