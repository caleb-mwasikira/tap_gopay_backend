package handlers

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/go-chi/chi/v5"
)

const (
	CREDIT_CARD_MIN_LEN         int     = 16
	CREDIT_CARD_INITIAL_DEPOSIT float64 = 100
)

// Credit card number will be in the format
//
//	1234 5678 8765 5432
func generateCreditCardNo() string {
	// TODO: Implement credit card_no generation using the Luhn algorithm

	str := strings.Builder{}
	index := 0

	for range CREDIT_CARD_MIN_LEN {
		if index != 0 && (index%4) == 0 {
			str.WriteString(" ")
		}

		num := rand.IntN(10)
		str.WriteString(fmt.Sprintf("%d", num))
		index++
	}
	cardNo := str.String()
	return strings.TrimSpace(cardNo)
}

func NewCreditCard(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	cardNo := generateCreditCardNo()
	creditCard, err := database.CreateCreditCard(
		user.Id, cardNo, CREDIT_CARD_INITIAL_DEPOSIT,
	)
	if err != nil {
		api.Errorf(w, "Error creating credit card", err)
		return
	}

	api.OK2(w, creditCard)
}

// Fetch all credit cards associated with currently logged in user
func GetAllCreditCards(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	creditCards, err := database.GetAllCreditCards(user.Id)
	if err != nil {
		api.Errorf(w, "Error fetching account credit cards", err)
		return
	}

	api.OK2(w, creditCards)
}

func GetCreditCard(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	cardNo := chi.URLParam(r, "card_no")
	if err := validateCardNumber(cardNo); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	creditCard, err := database.GetCreditCard(user.Id, cardNo)
	if err != nil {
		api.Errorf(w, "Error fetching credit card details", err)
		return
	}

	api.OK2(w, creditCard)
}

func FreezeCreditCard(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	cardNo := chi.URLParam(r, "card_no")
	if err := validateCardNumber(cardNo); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	err := database.FreezeCreditCard(user.Id, cardNo)
	if err != nil {
		api.Errorf(w, "Error freezing credit card account", err)
		return
	}

	api.OK(w, fmt.Sprintf("Credit card %v deactivated successfully", cardNo))
}

func ActivateCreditCard(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	cardNo := chi.URLParam(r, "card_no")
	if err := validateCardNumber(cardNo); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	err := database.ActivateCreditCard(user.Id, cardNo)
	if err != nil {
		api.Errorf(w, "Error activating credit card account", err)
		return
	}

	api.OK(w, fmt.Sprintf("Credit card %v activated successfully", cardNo))
}
