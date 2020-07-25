package ec

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"io"
)

func genRSAKey(s int) (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, s)
}

func encrypt(m []byte, k *rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, k, m, []byte{})
}

func decrypt(c []byte, k *rsa.PrivateKey) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, k, c, []byte{})
}

func genSyncKey(l int) []byte {
	t := make([]byte, l)
	rand.Read(t)
	return t
}

func encryptSync(m []byte, k []byte) ([]byte, error) {
	c, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, m, nil), nil
}

func decryptSync(s []byte, k []byte) ([]byte, error) {
	c, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(s) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ct := s[:nonceSize], s[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}
