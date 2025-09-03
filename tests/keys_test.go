package tests

import (
	"path/filepath"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

// This is NOT an actual test.
// It is more of a helper function for the tests.
func TestGenerateKeys(t *testing.T) {
	privKey, pubKey, err := encrypt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Unexpected error generating key pair; %v", err)
	}

	privKeyPath := filepath.Join("keys", "test.key")
	pubKeyPath := filepath.Join("keys", "test.pub")

	err = encrypt.SavePrivateKeyToFile(privKey, privKeyPath, false)
	if err != nil {
		t.Errorf("Unexpected error saving EC private key to file; %v\n", err)
	}

	err = encrypt.SavePublicKeyToFile(pubKey, pubKeyPath, false)
	if err != nil {
		t.Errorf("Unexpected error saving EC public key to file; %v\n", err)
	}
}
