package handlers

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/go-chi/chi/v5"
)

type TransactionRequest struct {
	Sender    string  `json:"sender" validate:"account"`
	Receiver  string  `json:"receiver" validate:"account"`
	Amount    float64 `json:"amount" validate:"amount"`
	Timestamp string  `json:"timestamp"` // Time when transaction was initiated by the client

	// Base64 encoded signature
	// Request signed by sender
	Signature string `json:"signature" validate:"signature"`
}

func (req TransactionRequest) Hash() []byte {
	data, _ := json.Marshal(struct {
		Sender    string  `json:"sender"`
		Receiver  string  `json:"receiver"`
		Amount    float64 `json:"amount"`
		Timestamp string  `json:"timestamp"`
		// We purposefully omit the signature
	}{
		Sender:    req.Sender,
		Receiver:  req.Receiver,
		Amount:    req.Amount,
		Timestamp: req.Timestamp,
	})

	h := sha256.Sum256(data)
	return h[:]
}

func getCreditCardOwnedBy(phone string) (*database.CreditCard, error) {
	creditCards, err := database.GetAllCreditCardsOwnedBy(
		phone,
		func(cc *database.CreditCard) bool {
			return cc.IsActive
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching credit cards owned by phone number; %v", err)
	}

	if len(creditCards) == 0 {
		return nil, fmt.Errorf("no credit cards found owned by phone number")
	}

	return creditCards[0], nil
}

func verifySignature(pubKeyBytes []byte, data []byte, b64EncodedSignature string) error {
	pubKey, err := encrypt.LoadPublicKeyFromBytes(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("error loading public key; %v", err)
	}

	signature, err := base64.StdEncoding.DecodeString(b64EncodedSignature)
	if err != nil {
		return fmt.Errorf("error base64 decoding signature; %v", err)
	}

	ok := ecdsa.VerifyASN1(pubKey, data, signature)
	if !ok {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func TransferFunds(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	var req TransactionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "sender, receiver, amount, timestamp and signature fields required")
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	data := req.Hash()
	err = verifySignature(user.PublicKey, data, req.Signature)
	if err != nil {
		api.Errorf(w, "Error sending funds. Signature verification failed", err)
		return
	}

	if isValidPhoneNumber(req.Sender) {
		creditCard, err := getCreditCardOwnedBy(req.Sender)
		if err != nil {
			api.Errorf(w, "Sender has no active credit card accounts", err)
			return
		}
		req.Sender = creditCard.CardNo
	}

	if isValidPhoneNumber(req.Receiver) {
		creditCard, err := getCreditCardOwnedBy(req.Receiver)
		if err != nil {
			api.Errorf(w, "Receiver has no active credit card accounts", err)
			return
		}
		req.Receiver = creditCard.CardNo
	}

	if req.Sender == req.Receiver {
		api.BadRequest(w, "Sender and receiver share the same account")
		return
	}

	// Check amount is within spending limits
	ok = database.IsWithinSpendingLimits(req.Sender, req.Amount)
	if !ok {
		api.Conflict(w, "Credit card exceeded spending limits")
		return
	}

	pubKeyId := sha256.Sum256(user.PublicKey)

	transaction, err := database.CreateTransaction(
		req.Sender, req.Receiver, req.Amount,
		req.Timestamp, req.Signature,
		hex.EncodeToString(pubKeyId[:]),
	)
	if err != nil {
		api.Errorf(w, "Error transferring funds", err)
		return
	}

	api.OK2(w, transaction)
}

// A user can request funds from another user
func RequestFunds(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	var req TransactionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "sender, receiver, amount and signature fields required")
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	data := req.Hash()
	err = verifySignature(user.PublicKey, data, req.Signature)
	if err != nil {
		api.Errorf(w, "Error requesting funds. Signature verification failed", nil)
		return
	}

	// Add record to database
	pubKeyId := sha256.Sum256(user.PublicKey)

	transaction, err := database.CreateRequestFunds(
		req.Sender, req.Receiver, req.Amount,
		req.Timestamp, req.Signature,
		hex.EncodeToString(pubKeyId[:]),
	)
	if err != nil {
		api.Errorf(w, "Error requesting funds", err)
		return
	}

	api.OK2(w, transaction)
}

func GetTransaction(w http.ResponseWriter, r *http.Request) {
	transactionId := chi.URLParam(r, "transaction_id")
	transactionId = strings.TrimSpace(transactionId)

	if len(transactionId) < database.TRANSACTION_ID_LEN {
		api.BadRequest(w, "Invalid transaction id")
		return
	}

	t, err := database.GetTransaction(transactionId)
	if err != nil {
		api.Errorf(w, "Error fetching transaction", err)
		return
	}

	api.OK2(w, t)
}

func GetRecentTransactions(w http.ResponseWriter, r *http.Request) {
	cardNo := chi.URLParam(r, "card_no")
	if err := validateCardNumber(cardNo); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	transactions, err := database.GetRecentTransactions(cardNo)
	if err != nil {
		api.Errorf(w, "Error fetching credit card transactions", err)
		return
	}

	api.OK2(w, transactions)
}
