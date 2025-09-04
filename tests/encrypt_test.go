package tests

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

func readOrCreateSeedPhrase(path string) ([]byte, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if len(data) > 0 {
		return data, nil
	}

	// There was no seed phrase stored in file
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	seed := encrypt.GenerateSeedPhrase()
	_, err = file.WriteString(seed)
	if err != nil {
		return nil, err
	}
	return []byte(seed), nil
}

// Test that generating keys with same seed phrase creates the
// same keys
func TestGenerateKeys(t *testing.T) {
	seedPhraseFile := filepath.Join("keys", "seed_phrase")
	seedPhrase, err := readOrCreateSeedPhrase(seedPhraseFile)
	if err != nil {
		t.Fatalf("Error reading or creating seed phrase; %v\n", err)
	}

	// Generate 1st key pair and hash private key
	seedPhraseReader := encrypt.NewSeedPhraseReader(seedPhrase)
	privKey, _, err := encrypt.GenerateKeyPair(seedPhraseReader)
	if err != nil {
		t.Fatalf("Unexpected error generating key pair; %v", err)
	}

	privKeyBytes, err := encrypt.PemEncodePrivateKey(privKey)
	if err != nil {
		t.Fatalf("Error PEM encoding private key; %v\n", err)
	}
	hash1 := sha256.Sum256(privKeyBytes)

	// Generate 2nd key pair and hash private key
	seedPhraseReader = encrypt.NewSeedPhraseReader(seedPhrase)
	privKey, _, err = encrypt.GenerateKeyPair(seedPhraseReader)
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
