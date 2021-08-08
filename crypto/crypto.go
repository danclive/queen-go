package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"

	"github.com/danclive/nson-go"

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

type Crypto struct {
	aead   cipher.AEAD
	method Method
}

func NewCrypto(method Method, key string) (*Crypto, error) {
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

		return &Crypto{aesgcm, method}, nil
	case Aes256Gcm:
		block, err := aes.NewCipher(aead_key)
		if err != nil {
			return nil, err
		}

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		return &Crypto{aesgcm, method}, nil
	case ChaCha20Poly1305:
		aesgcm, err := chacha20poly1305.New(aead_key)
		if err != nil {
			return nil, err
		}

		return &Crypto{aesgcm, method}, nil
	}

	return nil, errors.New("unsupport crypto method")
}

func (c *Crypto) Method() Method {
	return c.method
}

func (c *Crypto) Encrypt(in []byte) ([]byte, error) {
	if len(in) <= 4 {
		return nil, errors.New("invalid in size")
	}

	nonce := nonce()

	cipherdata := c.aead.Seal(in[:4], nonce, in[4:], nil)

	cipherdata = append(cipherdata, nonce...)

	b := util.UInt32ToBytes(uint32(len(cipherdata)))
	for i := 0; i < len(b); i++ {
		cipherdata[i] = b[i]
	}

	return cipherdata, nil
}

func (c *Crypto) Decrypt(in []byte) ([]byte, error) {
	if len(in) <= 4+16+12 {
		return nil, errors.New("invalid in size")
	}

	nonce := in[len(in)-12:]

	plaindata, err := c.aead.Open(in[:4], nonce, in[4:len(in)-12], nil)
	if err != nil {
		return nil, err
	}

	b := util.UInt32ToBytes(uint32(len(plaindata)))
	for i := 0; i < len(b); i++ {
		plaindata[i] = b[i]
	}

	return plaindata, nil
}

// func (c *Crypto) Open(dst, in []byte) ([]byte, error) {
// 	if len(in) <= 16+12 {
// 		return nil, errors.New("invalid in size")
// 	}

// 	nonce := in[len(in)-12:]

// 	plaindata, err := c.aead.Open(dst, nonce, in[:len(in)-12], nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	b := util.UInt32ToBytes(uint32(len(plaindata)))
// 	for i := 0; i < len(b); i++ {
// 		plaindata[i] = b[i]
// 	}

// 	return plaindata, nil
// }

func nonce() []byte {
	return []byte(nson.NewMessageId())
}

// func randNonce() ([]byte, error) {
// 	nonce := make([]byte, 12)
// 	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
// 		return nil, err
// 	}

// 	return nonce, nil
// }

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
