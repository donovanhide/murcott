package murcott

import (
	"errors"
	"net"

	"github.com/vmihailenco/msgpack"
)

type packet struct {
	Dst     NodeId    `msgpack:"dst"`
	Src     NodeId    `msgpack:"src"`
	Type    string    `msgpack:"type"`
	Payload []byte    `msgpack:"payload"`
	Sign    signature `msgpack:"sign"`
	addr    *net.UDPAddr
}

func (p *packet) serialize() []byte {
	ary := []interface{}{
		p.Dst.Bytes(),
		p.Src.Bytes(),
		p.Type,
		p.Payload,
	}

	data, _ := msgpack.Marshal(ary)
	return data
}

func (p *packet) sign(key *PrivateKey) error {
	sign := key.sign(p.serialize())
	if sign == nil {
		return errors.New("cannot sign packet")
	}
	p.Sign = *sign
	return nil
}

func (p *packet) verify(key *PublicKey) bool {
	return key.verify(p.serialize(), &p.Sign)
}
