package ec

import "testing"

func TestBasicSyncEncryption(t *testing.T) {
	k := genSyncKey(32)
	m := []byte("Wasp Hitler")
	e, _ := encryptSync(m, k)
	dm, _ := decryptSync(e, k)
	if string(m) != string(dm) {
		t.Fail()
	}
}

func TestBasicAsyncEncryption(t *testing.T) {
	pk, _ := genRSAKey(4098)
	m := []byte("I'm Mr.me6. Look at me.")
	em, _ := encrypt(m, &pk.PublicKey)
	dm, _ := decrypt(em, pk)
	if string(m) != string(dm) {
		t.Fail()
	}
}
