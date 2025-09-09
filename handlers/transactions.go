package handlers

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
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
	CreatedAt string  `json:"created_at"` // RFC3339 formatted string

	// Base64 encoded signature
	// Request signed by sender
	Signature string `json:"signature" validate:"signature"`
}

func (req TransactionRequest) Hash() []byte {
	data, _ := json.Marshal(struct {
		Sender    string  `json:"sender"`
		Receiver  string  `json:"receiver"`
		Amount    float64 `json:"amount"`
		CreatedAt string  `json:"created_at"`
		// We purposefully omit the signature
	}{
		Sender:    req.Sender,
		Receiver:  req.Receiver,
		Amount:    req.Amount,
		CreatedAt: req.CreatedAt,
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

func TransferFunds(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	var req TransactionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "sender, receiver, amount, created_at and signature fields required")
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	// Load sender's public key to verify signature on send funds request
	sendersPubKey, err := encrypt.LoadPublicKeyFromBytes(user.PublicKey)
	if err != nil {
		api.Errorf(w, "Error decoding senders public key", err)
		return
	}

	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		api.BadRequest(w, "Error transferring funds. Invalid signature")
		return
	}

	digest := req.Hash()
	ok = ecdsa.VerifyASN1(sendersPubKey, digest, signature)
	if !ok {
		api.Errorf(w, "Error transferring funds. Signature verification failed", nil)
		return
	}

	var (
		sender   string = req.Sender
		receiver string = req.Receiver
	)

	if isValidPhoneNumber(sender) {
		creditCard, err := getCreditCardOwnedBy(sender)
		if err != nil {
			api.Errorf(w, "Sender has no active credit card accounts", err)
			return
		}
		sender = creditCard.CardNo
	}

	if isValidPhoneNumber(receiver) {
		creditCard, err := getCreditCardOwnedBy(receiver)
		if err != nil {
			api.Errorf(w, "Receiver has no active credit card accounts", err)
			return
		}
		receiver = creditCard.CardNo
	}

	// Check that sender != receiver
	if sender == receiver {
		api.BadRequest(w, "Sender and receiver share the same account")
		return
	}

	transaction, err := database.CreateTransaction(
		sender, receiver, req.Amount,
		req.CreatedAt, req.Signature,
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

	// Load receivers public key
	receiversPubKey, err := encrypt.LoadPublicKeyFromBytes(user.PublicKey)
	if err != nil {
		api.Errorf(w, "Error decoding receiver's public key", err)
		return
	}

	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		api.BadRequest(w, "Error transferring funds. Invalid signature")
		return
	}

	// Verify request signature belongs to the receiver
	digest := req.Hash()
	ok = ecdsa.VerifyASN1(receiversPubKey, digest, signature)
	if !ok {
		api.Errorf(w, "Error requesting funds. Signature verification failed", nil)
		return
	}

	// Add record to database
	transaction, err := database.CreateRequestFunds(
		req.Sender, req.Receiver, req.Amount,
		req.CreatedAt, req.Signature,
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
