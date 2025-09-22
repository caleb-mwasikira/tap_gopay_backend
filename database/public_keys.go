package database

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

func getPubKeyHash(b64EncodedPubKey string) (string, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(b64EncodedPubKey)
	if err != nil {
		return "", err
	}

	hashed := sha256.Sum256(pubKeyBytes)
	pubKeyHash := base64.StdEncoding.EncodeToString(hashed[:])
	return pubKeyHash, nil
}

func CreatePublicKey(email string, b64EncodedPubKey string) error {
	pubKeyHash, err := getPubKeyHash(b64EncodedPubKey)
	if err != nil {
		return err
	}

	query := `
		INSERT IGNORE INTO public_keys(email, public_key_hash, public_key)
		VALUES(?, ?, ?)
	`
	_, err = db.Exec(
		query,
		email,
		pubKeyHash,
		b64EncodedPubKey,
	)
	return err
}

func GetPublicKey(email, b64EncodedPubKeyHash string) (*ecdsa.PublicKey, error) {
	query := "SELECT public_key FROM public_keys WHERE email= ? AND public_key_hash= ?"
	row := db.QueryRow(query, email, b64EncodedPubKeyHash)

	var b64EncodedPubKey string

	err := row.Scan(&b64EncodedPubKey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(b64EncodedPubKey)
	if err != nil {
		return nil, err
	}

	return encrypt.PemDecodePublicKey(pubKeyBytes)
}

func ChangePublicKey(email string, b64EncodedPubKey string) error {
	pubKeyHash, err := getPubKeyHash(b64EncodedPubKey)
	if err != nil {
		return err
	}

	query := "UPDATE public_keys SET public_key_hash= ?, public_key= ? WHERE email= ?"
	_, err = db.Exec(query, pubKeyHash, b64EncodedPubKey, email)
	return err
}
