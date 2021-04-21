package sign

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerify(t *testing.T) {
	publicKey := []byte{13, 117, 98, 254, 249, 102, 178, 155, 99, 220, 60, 180, 148, 53, 58, 48, 96, 91, 213, 128, 79, 229, 3, 102, 122, 73, 187, 161, 120, 42, 224, 7}

	sig := []byte{234, 206, 72, 37, 189, 155, 140, 64, 110, 84, 22, 93, 197, 169, 166, 188, 33, 42, 117, 127, 133, 102, 249, 26, 37, 108, 123, 36, 134, 47, 109, 200, 97, 111, 7, 217, 49, 72, 213, 43, 22, 98, 6, 206, 51, 72, 83, 68, 51, 129, 232, 182, 107, 81, 115, 91, 193, 92, 9, 74, 164, 150, 247, 11}

	ret := ed25519.Verify(ed25519.PublicKey(publicKey), []byte("hello world"), sig)
	assert.True(t, ret)
}

func TestSign(t *testing.T) {
	// rand.Seed(time.Now().Unix())

	pubkey, prikey, err := ed25519.GenerateKey(nil)
	assert.Nil(t, err)

	log.Println(pubkey)
	log.Println(prikey)
	log.Println(base64.StdEncoding.EncodeToString(prikey))
	log.Println(base64.StdEncoding.EncodeToString(prikey.Seed()))

	key, err := x509.MarshalPKCS8PrivateKey(prikey)
	assert.Nil(t, err)
	log.Println(base64.StdEncoding.EncodeToString(key))

	prikey2, err := x509.ParsePKCS8PrivateKey(key)
	assert.Nil(t, err)
	log.Println(prikey2.(ed25519.PrivateKey))

	// sig := ed25519.Sign(prikey, []byte("hello world"))
	// log.Println(sig)

	// ret := ed25519.Verify(ed25519.PublicKey(pubkey), []byte("hello world"), sig)
	// assert.True(t, ret)
}

func TestSign2(t *testing.T) {
	//prikey := []byte
}
