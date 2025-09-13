package tests

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"os"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

// Reads private key from file.
// If reading private key fails the func generates a new private key
// using provided seed phrase.
// The private key is saved to the same file that was to be read.
func getOrGeneratePrivateKey(path string, seedPhrase []byte) (*ecdsa.PrivateKey, error) {
	privateKey, err := encrypt.LoadPrivateKeyFromFile(path)
	if err == nil {
		return privateKey, nil
	}

	reader := bytes.NewBuffer(seedPhrase)
	privKey, _, err := encrypt.GenerateKeyPair(reader)
	if err != nil {
		return nil, err
	}

	data, err := encrypt.PemEncodePrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(path, data, 0700)
	return privKey, err
}

// Test deterministic key generation using password + KDF
func TestGenerateKeys(t *testing.T) {
	argon2Key, err := encrypt.DeriveKey(tommy.Password, nil)
	if err != nil {
		t.Fatalf("Error generating KDF based password; %v\n", err)
	}

	// Generate 1st key pair and hash private key
	reader := bytes.NewBuffer(argon2Key.Key)
	privKey, _, err := encrypt.GenerateKeyPair(reader)
	if err != nil {
		t.Fatalf("Unexpected error generating key pair; %v", err)
	}

	privKeyBytes, err := encrypt.PemEncodePrivateKey(privKey)
	if err != nil {
		t.Fatalf("Error PEM encoding private key; %v\n", err)
	}
	hash1 := sha256.Sum256(privKeyBytes)

	// Generate 2nd key pair and hash private key
	reader = bytes.NewBuffer(argon2Key.Key)
	privKey, _, err = encrypt.GenerateKeyPair(reader)
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
