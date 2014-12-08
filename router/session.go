package router

import (
	"errors"
	"net"
	"time"

	"github.com/h2so5/murcott/internal"
	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

type session struct {
	conn net.Conn
	rkey *utils.PublicKey
	lkey *utils.PrivateKey
}

func newSesion(conn net.Conn, lkey *utils.PrivateKey) (*session, error) {
	s := session{
		conn: conn,
		lkey: lkey,
	}

	data, err := msgpack.Marshal(lkey.PublicKey)
	if err != nil {
		return nil, err
	}

	pkt := internal.Packet{
		Src:     lkey.PublicKeyHash(),
		Type:    "key",
		Payload: data,
	}

	err = s.Write(pkt)
	if err != nil {
		return nil, err
	}

	err = s.verify()
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *session) ID() utils.NodeID {
	return s.rkey.PublicKeyHash()
}

func (s *session) Read() (internal.Packet, error) {
	var packet internal.Packet
	d := msgpack.NewDecoder(s.conn)
	err := d.Decode(&packet)
	if err != nil {
		return internal.Packet{}, err
	}
	if !packet.Verify(s.rkey) {
		return internal.Packet{}, errors.New("receive wrong packet")
	}
	return packet, nil
}

func (s *session) Write(p internal.Packet) error {
	err := p.Sign(s.lkey)
	if err != nil {
		return err
	}
	d := msgpack.NewEncoder(s.conn)
	return d.Encode(p)
}

func (s *session) verify() error {
	s.conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	defer s.conn.SetReadDeadline(time.Time{})
	r := msgpack.NewDecoder(s.conn)
	var packet internal.Packet
	err := r.Decode(&packet)
	if err != nil {
		return err
	}
	if packet.Type == "key" {
		var key utils.PublicKey
		err := msgpack.Unmarshal(packet.Payload, &key)
		if err == nil {
			id := key.PublicKeyHash()
			if id.Cmp(packet.Src) != 0 {
				return errors.New("receive wrong public key")
			}
			s.rkey = &key
		}
	} else {
		return errors.New("receive wrong packet")
	}
	return nil
}
