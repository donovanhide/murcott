package murcott

import (
	"errors"
	"net"

	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

type packet struct {
	Dst     utils.NodeID    `msgpack:"dst"`
	Src     utils.NodeID    `msgpack:"src"`
	Type    string          `msgpack:"type"`
	Payload []byte          `msgpack:"payload"`
	Sign    utils.Signature `msgpack:"sign"`
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

func (p *packet) sign(key *utils.PrivateKey) error {
	sign := key.Sign(p.serialize())
	if sign == nil {
		return errors.New("cannot sign packet")
	}
	p.Sign = *sign
	return nil
}

func (p *packet) verify(key *utils.PublicKey) bool {
	return key.Verify(p.serialize(), &p.Sign)
}
