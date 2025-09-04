package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

const (
	CREDIT_CARD_MIN_LEN         int     = 12
	CREDIT_CARD_INITIAL_DEPOSIT float64 = 100

	// Field name given to the public key a client uploads
	// when creating a new credit card
	PUB_KEY_FIELD string = "public_key"
)

func generateCreditCardNo() string {
	// TODO: Implement credit card_no generation using the Luhn algorithm

	str := strings.Builder{}

	for range CREDIT_CARD_MIN_LEN {
		num := rand.IntN(10)
		str.WriteString(fmt.Sprintf("%d", num))
	}
	return str.String()
}

func NewCreditCard(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		api.BadRequest(w, "Error parsing multipart/form-data")
		return
	}

	file, _, err := r.FormFile(PUB_KEY_FIELD)
	if err != nil {
		api.BadRequest(w, "Missing %v file upload", PUB_KEY_FIELD)
		return
	}
	defer file.Close()

	pubKeyBytes, err := io.ReadAll(file)
	if err != nil {
		message := fmt.Sprintf("Error reading %v file upload", PUB_KEY_FIELD)
		api.Errorf(w, message, err)
		return
	}

	if err := validatePublicKey(pubKeyBytes); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	cardNo := generateCreditCardNo()
	creditCard, err := database.CreateCreditCard(
		user.Id, cardNo,
		CREDIT_CARD_INITIAL_DEPOSIT, pubKeyBytes,
	)
	if err != nil {
		api.Errorf(w, "Error creating credit card", err)
		return
	}

	api.OK2(w, creditCard)
}

// Fetch all credit cards associated with currently logged in user
func GetCreditCards(w http.ResponseWriter, r *http.Request) {
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

type CreditCardRequest struct {
	CardNo string `json:"card_no" validate:"card_no"`
}

func FreezeCreditCard(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	var req CreditCardRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "card_no field required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
		return
	}

	err = database.FreezeCreditCard(user.Id, req.CardNo)
	if err != nil {
		api.Errorf(w, "Error freezing credit card account", err)
		return
	}

	api.OK(w, "Credit card deactivated successfully")
}

func ActivateCreditCard(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	var req CreditCardRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "card_no field required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
		return
	}

	err = database.ActivateCreditCard(user.Id, req.CardNo)
	if err != nil {
		api.Errorf(w, "Error activating credit card account", err)
		return
	}

	api.OK(w, "Credit card activated successfully")
}
