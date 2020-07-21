package ec

import "testing"

func TestBasicEncryption(t *testing.T) {
	pk, _ := genRSAKey(4098)
	m := []byte("I'm Mr.me6. Look at me.")
	em, _ := encrypt(m, &pk.PublicKey)
	dm, _ := decrypt(em, pk)
	if string(m) != string(dm) {
		t.Fail()
	}
}
