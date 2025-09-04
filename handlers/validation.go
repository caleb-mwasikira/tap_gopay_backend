package handlers

import (
	"encoding/hex"
	"fmt"
	"net/mail"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

const (
	MIN_NAME_LEN            = 4
	MIN_PASSWORD_LEN        = 8
	MIN_AMOUNT              = 1
	CURRENCY_CODE    string = "KES"
)

func validateStruct(obj any) []string {
	var errs []string

	objValue := reflect.ValueOf(obj)
	objTyp := reflect.TypeOf(obj)
	objKind := reflect.TypeOf(obj).Kind()

	if objKind != reflect.Struct {
		errs = append(errs, "obj is not a Struct")
		return errs
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
					errs = append(errs, fmt.Sprintf("%s is required", field.Name))
				}
			}
			if strings.HasPrefix(rule, "min=") {
				min, _ := strconv.Atoi(strings.TrimPrefix(rule, "min="))
				if lessThan(fieldValue, min) {
					errs = append(errs, fmt.Sprintf("%s must be greater than %d", field.Name, min))
				}
			}
			if strings.HasPrefix(rule, "max=") {
				max, _ := strconv.Atoi(strings.TrimPrefix(rule, "max="))
				if greaterThan(fieldValue, max) {
					errs = append(errs, fmt.Sprintf("%s must be less than %d", field.Name, max))
				}
			}
			if rule == "email" {
				str, _ := fieldValue.(string)
				if err := validateEmail(str); err != nil {
					errs = append(errs, err.Error())
				}
			}
			if rule == "password" {
				str, _ := fieldValue.(string)
				if err := validatePassword(str); err != nil {
					errs = append(errs, err.Error())
				}
			}
			if rule == "phone_no" {
				str, _ := fieldValue.(string)
				if err := validatePhone(str); err != nil {
					errs = append(errs, err.Error())
				}
			}
			if rule == "card_no" {
				str, _ := fieldValue.(string)
				if err := validateCreditCardNo(str); err != nil {
					errs = append(errs, err.Error())
				}
			}
			if rule == "amount" {
				value, _ := fieldValue.(float64)
				if err := validateAmount(value); err != nil {
					errs = append(errs, err.Error())
				}
			}
			if rule == "signature" {
				str, _ := fieldValue.(string)
				if _, err := validateSignature(str); err != nil {
					errs = append(errs, err.Error())
				}
			}
		}
	}
	return errs
}

func greaterThan(v interface{}, limit int) bool {
	switch val := v.(type) {
	case int:
		return val >= limit
	case string:
		return len(val) >= limit
	default:
		return false
	}
}

func lessThan(v interface{}, limit int) bool {
	switch val := v.(type) {
	case int:
		return val <= limit
	case string:
		return len(val) <= limit
	default:
		return false
	}
}

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

// Right now this only supported ecdsa public keys
func validatePublicKey(pubKeyBytes []byte) error {
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
