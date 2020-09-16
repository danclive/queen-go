package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/danclive/queen-go/util"
	"golang.org/x/crypto/chacha20poly1305"
)

type Method uint8

const (
	None Method = iota
	Aes128Gcm
	Aes256Gcm
	ChaCha20Poly1305
)

func (m Method) ToString() string {
	switch m {
	case Aes128Gcm:
		return "A1G"
	case Aes256Gcm:
		return "A2G"
	case ChaCha20Poly1305:
		return "CP1"
	}

	return ""
}

type Aead struct {
	inner cipher.AEAD
}

func NewAead(method Method, key string) (*Aead, error) {
	h := sha256.New()
	h.Write([]byte(key))
	aead_key := h.Sum(nil)

	switch method {
	case Aes128Gcm:
		block, err := aes.NewCipher(aead_key[:16])
		if err != nil {
			return nil, err
		}

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		return &Aead{inner: aesgcm}, nil
	case Aes256Gcm:
		block, err := aes.NewCipher(aead_key)
		if err != nil {
			return nil, err
		}

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		return &Aead{inner: aesgcm}, nil
	case ChaCha20Poly1305:
		aesgcm, err := chacha20poly1305.New(aead_key)
		if err != nil {
			return nil, err
		}

		return &Aead{inner: aesgcm}, nil
	}

	return nil, errors.New("unsupport")
}

func (aead *Aead) Encrypt(in []byte) ([]byte, error) {
	if len(in) <= 4 {
		return nil, errors.New("invalid in size")
	}

	nonce, err := randNonce()
	if err != nil {
		return nil, err
	}

	cipherdata := aead.inner.Seal(nil, nonce, in[4:], nil)

	bytes := util.UInt32ToBytes(uint32(4 + len(cipherdata) + 12))
	bytes = append(bytes, cipherdata...)
	bytes = append(bytes, nonce...)

	fmt.Println(bytes)

	return bytes, nil
}

func (aead *Aead) Decrypt(in []byte) ([]byte, error) {
	if len(in) <= 4+16+12 {
		return nil, errors.New("invalid in size")
	}

	nonce := in[len(in)-12:]

	plaindata, err := aead.inner.Open(nil, nonce, in[4:len(in)-12], nil)
	if err != nil {
		return nil, err
	}

	bytes := util.UInt32ToBytes(uint32(len(plaindata) + 4))
	bytes = append(bytes, plaindata...)

	return bytes, nil
}

func randNonce() ([]byte, error) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return nonce, nil
}

// func increaseNonce(nonce []byte) {
// 	for i, v := range nonce {
// 		if v == 255 {
// 			nonce[i] = 0
// 		} else {
// 			nonce[i] += 1
// 			return
// 		}
// 	}
// }
