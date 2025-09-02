package handlers

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

const (
	MIN_NAME_LEN     = 4
	MIN_PASSWORD_LEN = 8
)

func isEmpty(value string) bool {
	return strings.TrimSpace(value) == ""
}

func validateName(field, value string) error {
	if isEmpty(value) {
		return fmt.Errorf("%v value cannot be empty", field)
	}

	value = strings.TrimSpace(value)
	if len(value) < MIN_NAME_LEN {
		return fmt.Errorf("%v too short", field)
	}
	return nil
}

func validateEmail(email string) error {
	if isEmpty(email) {
		return fmt.Errorf("email value cannot be empty")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

func validatePassword(password string) error {
	if isEmpty(password) {
		return fmt.Errorf("password value cannot be empty")
	}

	password = strings.TrimSpace(password)
	if len(password) < MIN_PASSWORD_LEN {
		return fmt.Errorf("password too short")
	}

	// TODO: check password strength
	return nil
}

func validatePhone(phone string) error {
	if isEmpty(phone) {
		return fmt.Errorf("phone number value cannot be empty")
	}

	// E.164 format: + followed by 8 to 15 digits
	re := regexp.MustCompile(`^\+?[1-9]\d{7,14}$`)
	if !re.MatchString(phone) {
		return fmt.Errorf("invalid phone number")
	}
	return nil
}

func validateECDSAPublicKey(pubKeyBytes []byte) error {
	block, _ := pem.Decode(pubKeyBytes)
	if block == nil || block.Type != "PUBLIC KEY" {
		return fmt.Errorf("Invalid PEM block")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}

	// Cast public key into ecdsa.PublicKey type
	_, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("Unsupported public key. Platform only supports ECDSA public keys")
	}
	return nil
}
