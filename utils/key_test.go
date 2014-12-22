package utils

import (
	"testing"

	"github.com/vmihailenco/msgpack"
)

func TestKeySignature(t *testing.T) {
	key := GeneratePrivateKey()
	data := "The quick brown fox jumps over the lazy dog"

	sign := key.Sign([]byte(data))
	if !key.Verify([]byte(data), sign) {
		t.Errorf("varification failed")
	}
}

func TestKeyString(t *testing.T) {
	key := GeneratePrivateKey()
	data := "The quick brown fox jumps over the lazy dog"

	str := key.String()
	key2 := PrivateKeyFromString(str)

	sign := key.Sign([]byte(data))
	if !key2.Verify([]byte(data), sign) {
		t.Errorf("varification failed")
	}
}

func TestKeyMsgpack(t *testing.T) {
	prikey := GeneratePrivateKey()
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

	sign := uprikey.Sign([]byte(data))
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

	if !upubkey.Verify([]byte(data), &usign) {
		t.Errorf("varification failed")
	}
}

func TestKeyPEM(t *testing.T) {
	prikey := GeneratePrivateKey()
	pubkey := prikey.PublicKey

	mprikey, err := prikey.MarshalText()
	if err != nil {
		t.Errorf("cannot marshal PrivateKey")
	}

	mpubkey, err := pubkey.MarshalText()
	if err != nil {
		t.Errorf("cannot marshal PublicKey")
	}

	var uprikey PrivateKey
	err = uprikey.UnmarshalText(mprikey)
	if err != nil {
		t.Errorf("cannot unmarshal PrivateKey")
	}

	var upubkey PublicKey
	err = upubkey.UnmarshalText(mpubkey)
	if err != nil {
		t.Errorf("cannot unmarshal PublicKey")
	}
}
