package tests

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

// Reads user's private key from file.
func getPrivateKey(email string) (*ecdsa.PrivateKey, error) {
	fpath := filepath.Join("keys", fmt.Sprintf("%v.key", email))
	return encrypt.LoadPrivateKeyFromFile(fpath)
}

func generatePrivateKey(path string, password string) (*ecdsa.PrivateKey, error) {
	privKey, _, err := encrypt.GenerateKeyPair(password)
	if err != nil {
		return nil, err
	}

	err = encrypt.SavePrivateKeyToFile(privKey, path, true)
	return privKey, err
}

// Test deterministic key generation using password + KDF
func TestGenerateKeys(t *testing.T) {
	// Generate 1st key pair and hash private key
	privKey, _, err := encrypt.GenerateKeyPair(tommy.Password)
	if err != nil {
		t.Fatalf("Unexpected error generating key pair; %v", err)
	}

	privKeyBytes, err := encrypt.PemEncodePrivateKey(privKey)
	if err != nil {
		t.Fatalf("Error PEM encoding private key; %v\n", err)
	}
	hash1 := sha256.Sum256(privKeyBytes)

	// Generate 2nd key pair and hash private key
	privKey, _, err = encrypt.GenerateKeyPair(tommy.Password)
	if err != nil {
		t.Fatalf("Unexpected error generating key pair; %v", err)
	}

	privKeyBytes, err = encrypt.PemEncodePrivateKey(privKey)
	if err != nil {
		t.Fatalf("Error PEM encoding private key; %v\n", err)
	}
	hash2 := sha256.Sum256(privKeyBytes)

	// Compare the 2 hashes
	if hash1 != hash2 {
		t.Errorf("Expected 2 private keys generated with the same seed phrase to be equal")
	}
}
