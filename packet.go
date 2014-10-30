package murcott

import (
	"errors"
	"net"

	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

type packet struct {
	Dst     murcott.NodeID    `msgpack:"dst"`
	Src     murcott.NodeID    `msgpack:"src"`
	Type    string            `msgpack:"type"`
	Payload []byte            `msgpack:"payload"`
	Sign    murcott.Signature `msgpack:"sign"`
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

func (p *packet) sign(key *murcott.PrivateKey) error {
	sign := key.Sign(p.serialize())
	if sign == nil {
		return errors.New("cannot sign packet")
	}
	p.Sign = *sign
	return nil
}

func (p *packet) verify(key *murcott.PublicKey) bool {
	return key.Verify(p.serialize(), &p.Sign)
}
