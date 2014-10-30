package utils

import (
	"testing"

	"github.com/h2so5/murcott/utils"
	"github.com/vmihailenco/msgpack"
)

func TestKeySignature(t *testing.T) {
	key := utils.GeneratePrivateKey()
	data := "The quick brown fox jumps over the lazy dog"

	sign := key.sign([]byte(data))
	if !key.verify([]byte(data), sign) {
		t.Errorf("varification failed")
	}
}

func TestKeyString(t *testing.T) {
	key := utils.GeneratePrivateKey()
	data := "The quick brown fox jumps over the lazy dog"

	str := key.String()
	key2 := PrivateKeyFromString(str)

	sign := key.sign([]byte(data))
	if !key2.verify([]byte(data), sign) {
		t.Errorf("varification failed")
	}
}

func TestKeyMsgpack(t *testing.T) {
	prikey := utils.GeneratePrivateKey()
	pubkey := prikey.PublicKey
	data := "The quick brown fox jumps over the lazy dog"

	mprikey, err := msgpack.Marshal(prikey)
	if err != nil {
		t.Errorf("cannot marshal PrivateKey")
	}

	mpubkey, err := msgpack.Marshal(pubkey)
	if err != nil {
		t.Errorf("cannot marshal PublicKey")
	}

	var uprikey PrivateKey
	err = msgpack.Unmarshal(mprikey, &uprikey)
	if err != nil {
		t.Errorf("cannot unmarshal PrivateKey")
	}

	sign := uprikey.sign([]byte(data))
	msign, err := msgpack.Marshal(sign)
	if err != nil {
		t.Errorf("cannot marshal signatur")
	}

	var upubkey PublicKey
	err = msgpack.Unmarshal(mpubkey, &upubkey)
	if err != nil {
		t.Errorf("cannot unmarshal PublicKey")
	}

	var usign Signature
	err = msgpack.Unmarshal(msign, &usign)
	if err != nil {
		t.Errorf("cannot unmarshal signature")
	}

	if !upubkey.verify([]byte(data), &usign) {
		t.Errorf("varification failed")
	}
}
