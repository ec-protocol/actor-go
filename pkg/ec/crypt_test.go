package ec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicSyncEncryption(t *testing.T) {
	key := genSyncKey(32)
	expect := []byte("Wasp Hitler!!!!! Wasp Hitler!!!!! Wasp Hitler!!!!! Wasp Hitler!!!!! Wasp Hitler!!!!!")
	e, _ := encryptSync(expect, key)
	actual, _ := decryptSync(e, key)

	assert.Equal(t, expect, actual)
}

func TestBasicAsyncEncryption(t *testing.T) {
	privateKey, _ := genRSAKey(4098)
	expect := []byte("I'm Mr.me6. Look at me.")
	e, _ := encrypt(expect, &privateKey.PublicKey)
	actual, _ := decrypt(e, privateKey)

	assert.Equal(t, expect, actual)
}
