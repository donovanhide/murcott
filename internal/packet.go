package internal

import (
	"errors"
	"net"

	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

type Packet struct {
	Dst     utils.NodeID    `msgpack:"dst"`
	Src     utils.NodeID    `msgpack:"src"`
	Type    string          `msgpack:"type"`
	Payload []byte          `msgpack:"payload"`
	S       utils.Signature `msgpack:"sign"`
	Addr    *net.UDPAddr
}

func (p *Packet) Serialize() []byte {
	ary := []interface{}{
		p.Dst.Bytes(),
		p.Src.Bytes(),
		p.Type,
		p.Payload,
	}

	data, _ := msgpack.Marshal(ary)
	return data
}

func (p *Packet) Sign(key *utils.PrivateKey) error {
	sign := key.Sign(p.Serialize())
	if sign == nil {
		return errors.New("cannot sign packet")
	}
	p.S = *sign
	return nil
}

func (p *Packet) Verify(key *utils.PublicKey) bool {
	return key.Verify(p.Serialize(), &p.S)
}
