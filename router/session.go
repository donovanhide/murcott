package router

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"time"

	"github.com/h2so5/murcott/internal"
	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

type session struct {
	conn net.Conn
	r    io.Reader
	w    io.Writer
	rkey *utils.PublicKey
	lkey *utils.PrivateKey
}

func newSesion(conn net.Conn, lkey *utils.PrivateKey) (*session, error) {
	s := session{
		conn: conn,
		r:    conn,
		w:    conn,
		lkey: lkey,
	}

	err := s.sendPubkey()
	if err != nil {
		return nil, err
	}

	err = s.verifyPubkey()
	if err != nil {
		return nil, err
	}

	outkey, err := s.sendCommonKey()
	if err != nil {
		return nil, err
	}

	inkey, err := s.verifyCommonKey()
	if err != nil {
		return nil, err
	}
	s.setKey(inkey, outkey)

	return &s, nil
}

func (s *session) ID() utils.NodeID {
	return utils.NewNodeID(s.rkey.Digest())
}

func (s *session) Read() (internal.Packet, error) {
	var packet internal.Packet
	d := msgpack.NewDecoder(s.r)
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
	d := msgpack.NewEncoder(s.w)
	return d.Encode(p)
}

func (s *session) verifyPubkey() error {
	s.conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	defer s.conn.SetReadDeadline(time.Time{})
	r := msgpack.NewDecoder(s.r)
	var packet internal.Packet
	err := r.Decode(&packet)
	if err != nil {
		return err
	}
	if packet.Type == "pubkey" {
		var key utils.PublicKey
		err := msgpack.Unmarshal(packet.Payload, &key)
		if err == nil {
			id := utils.NewNodeID(key.Digest())
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

func (s *session) verifyCommonKey() ([]byte, error) {
	s.conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	defer s.conn.SetReadDeadline(time.Time{})
	r := msgpack.NewDecoder(s.r)
	var packet internal.Packet
	err := r.Decode(&packet)
	if err != nil {
		return nil, err
	}
	if packet.Type == "key" {
		return packet.Payload, nil
	} else {
		return nil, errors.New("receive wrong packet")
	}
}

func (s *session) sendPubkey() error {
	data, err := msgpack.Marshal(s.lkey.PublicKey)
	if err != nil {
		return err
	}

	pkt := internal.Packet{
		Src:     utils.NewNodeID(s.lkey.Digest()),
		Type:    "pubkey",
		Payload: data,
	}

	err = s.Write(pkt)
	if err != nil {
		return err
	}
	return nil
}

func (s *session) sendCommonKey() ([]byte, error) {
	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return nil, err
	}

	pkt := internal.Packet{
		Src:     utils.NewNodeID(s.lkey.Digest()),
		Type:    "key",
		Payload: key[:],
	}

	err = s.Write(pkt)
	if err != nil {
		return nil, err
	}
	return key[:], nil
}

func (s *session) setKey(inkey, outkey []byte) error {
	block, err := aes.NewCipher(inkey)
	if err != nil {
		return err
	}
	var iniv [aes.BlockSize]byte
	s.r = cipher.StreamReader{S: cipher.NewOFB(block, iniv[:]), R: s.r}

	block, err = aes.NewCipher(outkey)
	if err != nil {
		return err
	}
	var outiv [aes.BlockSize]byte
	s.w = cipher.StreamWriter{S: cipher.NewOFB(block, outiv[:]), W: s.w}
	return nil
}
