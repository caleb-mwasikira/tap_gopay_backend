package handlers

import (
	"encoding/hex"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

const (
	MIN_NAME_LEN            = 4
	MIN_PASSWORD_LEN        = 8
	MIN_AMOUNT              = 1
	CURRENCY_CODE    string = "KES"
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
	_, err := encrypt.LoadPublicKeyFromBytes(pubKeyBytes)
	return err
}

func validateAmount(amount float64) error {
	if amount < MIN_AMOUNT {
		return fmt.Errorf("minimum transferrable amount is %v %v", MIN_AMOUNT, CURRENCY_CODE)
	}
	return nil
}

// Hex-decodes a signature value
func validateSignature(signature string) ([]byte, error) {
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("server expects hex-encoded signatures")
	}
	return sig, nil
}

func validateCreditCardNo(cardNo string) error {
	// TODO: Implement credit card_no validation using the Luhn algorithm

	cardNo = strings.TrimSpace(cardNo)
	if len(cardNo) < CREDIT_CARD_MIN_LEN {
		return fmt.Errorf("credit card number too short")
	}
	return nil
}
