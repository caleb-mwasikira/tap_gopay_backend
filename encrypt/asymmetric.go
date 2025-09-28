package encrypt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
)

var (
	ErrKeyNotFound           error = fmt.Errorf("no key found")
	ErrUnsupportedPublicKey  error = fmt.Errorf("invalid PEM block or block type. Server expects 'EC PUBLIC KEY' PEM block type")
	ErrUnsupportedPrivateKey error = fmt.Errorf("invalid PEM block or block type. Server expects 'EC PRIVATE KEY' PEM block type")
)

func GenerateKeyPair(seedPhrase string) (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	// Generate stronger key from seed phrase using KDF
	key, err := generateArgon2Key(seedPhrase, nil)
	if err != nil {
		return nil, nil, err
	}

	// Use P256 curve (secure and widely used)
	reader, err := NewKeyReader(key.Key)
	if err != nil {
		return nil, nil, err
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), reader)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, &privateKey.PublicKey, nil
}

func PemEncodePrivateKey(privKey *ecdsa.PrivateKey) ([]byte, error) {
	derBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	block := pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derBytes,
	}
	return pem.EncodeToMemory(&block), nil
}

func SavePrivateKeyToFile(privKey *ecdsa.PrivateKey, path string, overwrite bool) error {
	log.Printf("Saving EC private key to file; '%v'\n", path)

	privKeyBytes, err := PemEncodePrivateKey(privKey)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	fileEmpty := stat.Size() == 0
	if fileEmpty || overwrite {
		_, err = file.Write(privKeyBytes)
	}

	return err
}

func PemEncodePublicKey(pubKey *ecdsa.PublicKey) ([]byte, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	block := pem.Block{
		Type:  "EC PUBLIC KEY",
		Bytes: derBytes,
	}
	return pem.EncodeToMemory(&block), nil
}

func LoadPrivateKeyFromFile(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, ErrKeyNotFound
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, ErrUnsupportedPrivateKey
	}

	return x509.ParseECPrivateKey(block.Bytes)
}

func PemDecodePublicKey(pubKeyBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pubKeyBytes)
	if block == nil || block.Type != "EC PUBLIC KEY" {
		return nil, ErrUnsupportedPublicKey
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing public key")
	}

	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unsupported public key. Server expects ECDSA public keys")
	}
	return ecdsaPubKey, nil
}
