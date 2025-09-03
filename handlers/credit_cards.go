package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

const (
	CREDIT_CARD_MIN_LEN         int     = 12
	CREDIT_CARD_INITIAL_DEPOSIT float64 = 100

	// Field name given to the public key a client uploads
	// when creating a new credit card
	PUB_KEY_FIELDNAME string = "public_key"
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

func NewCreditCardHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "You are not authorized to view this resource",
		})
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		log.Printf("Error parsing request; %v\n", err)
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "Error parsing multipart/form-data",
		})
		return
	}

	file, _, err := r.FormFile(PUB_KEY_FIELDNAME)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Missing public_key file upload",
		})
		return
	}
	defer file.Close()

	pubKeyBytes, err := io.ReadAll(file)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error reading public_key file upload",
		})
		return
	}

	if err := validateECDSAPublicKey(pubKeyBytes); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	cardNo := generateCreditCardNo()
	creditCard, err := database.CreateCreditCard(
		user.Id, cardNo,
		CREDIT_CARD_INITIAL_DEPOSIT, pubKeyBytes,
	)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error creating credit card",
		})
		return
	}

	jsonResponse(w, http.StatusOK, creditCard)
}

type CreditCardRequest struct {
	CardNo string `json:"card_no"`
}

func (req CreditCardRequest) Validate() error {
	return validateCreditCardNo(req.CardNo)
}

// Fetch all credit cards associated with currently logged in user
func GetCreditCardsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "You are not authorized to view this resource",
		})
		return
	}

	creditCards, err := database.GetAllCreditCards(user.Id)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error fetching account credit cards",
		})
		return
	}

	jsonResponse(w, http.StatusOK, creditCards)
}

func FreezeCreditCardHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "You are not authorized to view this resource",
		})
		return
	}

	var req CreditCardRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "card_no field required",
		})
		return
	}

	if err := req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	err = database.FreezeCreditCard(user.Id, req.CardNo)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error freezing credit card account",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Credit card deactivated successfully",
	})
}

func ActivateCreditCardHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "You are not authorized to view this resource",
		})
		return
	}

	var req CreditCardRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "card_no field required",
		})
		return
	}

	if err := req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	err = database.ActivateCreditCard(user.Id, req.CardNo)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error activating credit card account",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Credit card activated successfully",
	})
}
