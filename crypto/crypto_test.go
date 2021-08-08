package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCrypto(t *testing.T) {
	crypto, err := NewCrypto(ChaCha20Poly1305, "key")

	require.NotNil(t, crypto)
	require.Nil(t, err)
}

func TestEncryptAndDecrypt(t *testing.T) {
	crypto, err := NewCrypto(ChaCha20Poly1305, "key")

	require.NotNil(t, crypto)
	require.Nil(t, err)

	data := []byte{8, 0, 0, 0, 5, 6, 7, 8}

	cipherdata, err := crypto.Encrypt(data)
	require.Nil(t, err)

	plaindata, err := crypto.Decrypt(cipherdata)
	require.Nil(t, err)

	require.Equal(t, data, plaindata)
}

// func TestEncryptAndOpen(t *testing.T) {
// 	crypto, err := NewCrypto(ChaCha20Poly1305, "key")

// 	require.NotNil(t, crypto)
// 	require.Nil(t, err)

// 	data := []byte{8, 0, 0, 0, 5, 6, 7, 8}

// 	cipherdata, err := crypto.Encrypt(data)
// 	require.Nil(t, err)

// 	plaindata, err := crypto.Open([]byte{36, 0, 0, 0}, cipherdata[4:])
// 	require.Nil(t, err)

// 	require.Equal(t, data, plaindata)
// }
