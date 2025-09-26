package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/nyaruka/phonenumbers"
)

const (
	MIN_NAME_LEN             = 4
	MIN_PASSWORD_LEN         = 8
	MIN_AMOUNT       float64 = 1.0
	CURRENCY_CODE    string  = "KES"
)

func validateStruct(obj any) error {
	objValue := reflect.ValueOf(obj)
	objTyp := reflect.TypeOf(obj)
	objKind := reflect.TypeOf(obj).Kind()

	if objKind != reflect.Struct {
		return errors.New("obj is not a Struct")
	}

	for i := 0; i < objTyp.NumField(); i++ {
		field := objTyp.Field(i)
		fieldValue := objValue.Field(i).Interface()

		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			if rule == "required" {
				isZero := reflect.ValueOf(field).IsZero()
				if isZero {
					return fmt.Errorf("%s is required", field.Name)
				}
			}
			if strings.HasPrefix(rule, "min=") {
				min, _ := strconv.Atoi(strings.TrimPrefix(rule, "min="))
				if lessThan(fieldValue, min) {
					return fmt.Errorf("%s must be greater than %d", field.Name, min)
				}
			}
			if strings.HasPrefix(rule, "max=") {
				max, _ := strconv.Atoi(strings.TrimPrefix(rule, "max="))
				if greaterThan(fieldValue, max) {
					return fmt.Errorf("%s must be less than %d", field.Name, max)
				}
			}
			if rule == "email" {
				str, _ := fieldValue.(string)
				if err := validateEmail(str); err != nil {
					return err
				}
			}
			if rule == "phone_no" {
				str, _ := fieldValue.(string)
				if err := validatePhoneNumber(str); err != nil {
					return err
				}
			}
			if rule == "wallet_address" {
				str, _ := fieldValue.(string)
				if err := validateWalletAddress(str); err != nil {
					return err
				}
			}
			if rule == "account" {
				str, _ := fieldValue.(string)

				// Check if value is either a wallet address or phone number
				err := validateWalletAddress(str)
				isWalletNumber := err == nil

				err2 := validatePhoneNumber(str)
				isPhoneNumber := err2 == nil

				if !isWalletNumber && !isPhoneNumber {
					return fmt.Errorf("%v must either be a wallet address or phone number", field.Name)
				}
			}
			if rule == "amount" {
				value, _ := fieldValue.(float64)
				if err := validateAmount(value); err != nil {
					return err
				}
			}
			if rule == "public_key_hash" {
				str, _ := fieldValue.(string)
				if !isBase64Encoded(str) {
					return errors.New("public_key_hash must be base64-encoded")
				}
			}
			if rule == "signature" {
				str, _ := fieldValue.(string)
				if !isBase64Encoded(str) {
					return errors.New("signature must be base64-encoded")
				}
			}
			if rule == "public_key" {
				data, _ := fieldValue.(string)
				if err := validatePublicKey(data); err != nil {
					return err
				}
			}
			if rule == "period" {
				str, _ := fieldValue.(string)
				if err := validatePeriod(str); err != nil {
					return err
				}
			}
			if rule == "expiry" {
				str, ok := fieldValue.(string)
				if !ok {
					return fmt.Errorf("invalid expiry format; expected RFC3339 string")
				}

				expiresAt, err := time.Parse(time.RFC3339, str)
				if err != nil {
					return fmt.Errorf("invalid expiry format; expected RFC3339 string")
				}

				if expiresAt.Before(time.Now()) {
					return fmt.Errorf("invalid expiry time; expiry time already passed")
				}
			}
		}
	}
	return nil
}

func greaterThan(v interface{}, limit int) bool {
	switch val := v.(type) {
	case int:
		return val > limit
	case string:
		return len(val) > limit
	default:
		return false
	}
}

func lessThan(v interface{}, limit int) bool {
	switch val := v.(type) {
	case int:
		return val < limit
	case string:
		return len(val) < limit
	default:
		return false
	}
}

func isEmpty(value string) bool {
	return strings.TrimSpace(value) == ""
}

func validateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

func isValidPhoneNumber(phone string) bool {
	if isEmpty(phone) {
		return false
	}

	phoneNo, err := phonenumbers.Parse(phone, "KE")
	if err != nil {
		return false
	}
	return phonenumbers.IsValidNumber(phoneNo)
}

func validatePhoneNumber(phone string) error {
	if isEmpty(phone) {
		return fmt.Errorf("phone number value cannot be empty")
	}

	if !isValidPhoneNumber(phone) {
		return fmt.Errorf("invalid phone number")
	}
	return nil
}

// Right now this only supports ecdsa public keys
func validatePublicKey(pubKey string) error {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		return err
	}

	_, err = encrypt.PemDecodePublicKey(pubKeyBytes)
	return err
}

func validateAmount(amount float64) error {
	if amount < MIN_AMOUNT {
		return fmt.Errorf("minimum transferrable amount is %v %v", MIN_AMOUNT, CURRENCY_CODE)
	}
	return nil
}

func isBase64Encoded(value string) bool {
	if isEmpty(value) {
		return false
	}

	_, err := base64.StdEncoding.DecodeString(value)
	return err == nil
}

func isDigitsOnly(s string) bool {
	for _, char := range s {
		if !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}

// Wallet address will be in the format
//
//	"0xwall" for inidividual wallets
//	"0xmult" for multi signature wallets
//	"0xp00l" for cash pools
func validateWalletAddress(walletAddress string) error {
	walletAddress = strings.TrimSpace(walletAddress)
	if walletAddress == "" {
		return fmt.Errorf("wallet address cannot be empty")
	}

	fields := strings.Split(walletAddress, " ")
	if len(fields) < 3 {
		return fmt.Errorf("invalid wallet address")
	}

	// Ensure wallet is made of digits only
	walletAddress = strings.ReplaceAll(walletAddress, " ", "")
	if !isDigitsOnly(walletAddress) {
		return fmt.Errorf("expected numeric wallet address")
	}

	return nil
}

// Used for setting spending limits on wallets.
// Expects period value to be one of ['week', 'month', 'year']
func validatePeriod(period string) error {
	allowedPeriods := []string{"week", "month", "year"}
	if !slices.Contains(allowedPeriods, period) {
		return fmt.Errorf("invalid period. Expects period value to be one of ['week', 'month', 'year']")
	}
	return nil
}
