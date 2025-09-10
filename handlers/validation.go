package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"reflect"
	"strconv"
	"strings"

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
			if rule == "account" {
				str, _ := fieldValue.(string)

				// Check if value is either a credit card number or phone number
				err := validateCardNumber(str)
				isCreditCardNumber := err == nil

				err2 := validatePhoneNumber(str)
				isPhoneNumber := err2 == nil

				if !isCreditCardNumber && !isPhoneNumber {
					return fmt.Errorf("%v must either be a credit card number or phone number", field.Name)
				}
			}
			if rule == "amount" {
				value, _ := fieldValue.(float64)
				if err := validateAmount(value); err != nil {
					return err
				}
			}
			if rule == "signature" {
				str, _ := fieldValue.(string)
				if !isBase64Encoded(str) {
					return errors.New("signature must be base64-encoded")
				}
			}
			if rule == "public_key" {
				data, _ := fieldValue.([]byte)
				if err := validatePublicKey(data); err != nil {
					return err
				}
			}
		}
	}
	return nil
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

	_, err := phonenumbers.Parse(phone, "KE")
	return err == nil
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

func isBase64Encoded(value string) bool {
	_, err := base64.StdEncoding.DecodeString(value)
	return err == nil
}

// Credit card number will be in the format
//
//	1234 5678 8765 5432
func validateCardNumber(cardNo string) error {
	// TODO: Implement credit card_no validation using the Luhn algorithm

	cardNo = strings.TrimSpace(cardNo)
	fields := strings.Split(cardNo, " ")
	if len(fields) != 4 {
		return fmt.Errorf("invalid card number format")
	}

	for _, field := range fields {
		if _, err := strconv.Atoi(field); err != nil {
			return fmt.Errorf("card number must contain numbers only")
		}
		if len(field) != 4 {
			return fmt.Errorf("invalid card number format")
		}
	}
	return nil
}
