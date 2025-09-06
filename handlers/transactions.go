package handlers

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

type SendFundsRequest struct {
	Sender    string  `json:"sender" validate:"card_no"`
	Receiver  string  `json:"receiver" validate:"card_no"`
	Amount    float64 `json:"amount" validate:"amount"`
	CreatedAt string  `json:"created_at"` // ISO 8601 string

	// Base64 encoded signature
	// Request signed by sender
	Signature string `json:"signature" validate:"signature"`
}

func (req SendFundsRequest) Hash() []byte {
	data, _ := json.Marshal(struct {
		Sender    string  `json:"sender"`
		Receiver  string  `json:"receiver"`
		Amount    float64 `json:"amount"`
		CreatedAt string  `json:"created_at"`
	}{
		Sender:    req.Sender,
		Receiver:  req.Receiver,
		Amount:    req.Amount,
		CreatedAt: req.CreatedAt,
	})

	h := sha256.Sum256(data)
	return h[:]
}

func SendFunds(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	var req SendFundsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "sender, receiver, amount, created_at and signature fields required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
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

	err = database.CreateTransaction(
		req.Sender, req.Receiver, req.Amount,
		req.CreatedAt, req.Signature,
	)
	if err != nil {
		api.Errorf(w, "Error transferring funds", err)
		return
	}

	message := fmt.Sprintf("Confirmed. Account %v sent %v %v to %v",
		req.Sender, CURRENCY_CODE, req.Amount, req.Receiver,
	)
	api.OK(w, message)
}

// A user can request funds from another user
func RequestFunds(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	var req SendFundsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "sender, receiver, amount and signature fields required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
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
	err = database.CreateRequestFunds(
		req.Sender, req.Receiver, req.Amount,
		req.CreatedAt, req.Signature,
	)
	if err != nil {
		api.Errorf(w, "Error requesting funds", err)
		return
	}

	message := fmt.Sprintf("Confirmed. Account %v requested %v %v from %v",
		req.Receiver, CURRENCY_CODE, req.Amount, req.Sender,
	)
	api.OK(w, message)
}
