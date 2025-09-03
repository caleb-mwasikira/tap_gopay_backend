package handlers

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

type SendFundsRequest struct {
	Sender    string  `json:"sender"`
	Receiver  string  `json:"receiver"`
	Amount    float64 `json:"amount"`
	CreatedAt string  `json:"created_at"` // ISO 8601 string

	// Hex encoded signature
	// Request signed by sender
	Signature string `json:"signature"`

	signature []byte
}

func (req *SendFundsRequest) Validate() error {
	if err := validateCreditCardNo(req.Sender); err != nil {
		return err
	}
	if err := validateCreditCardNo(req.Receiver); err != nil {
		return err
	}
	if req.Sender == req.Receiver {
		return fmt.Errorf("sender and receiver cannot be the same account")
	}
	if err := validateAmount(req.Amount); err != nil {
		return err
	}

	signature, err := validateSignature(req.Signature)
	if err != nil {
		return err
	}
	req.signature = signature
	return nil
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

func SendFundsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "You are not authorized to view this resource",
		})
		return
	}

	var req SendFundsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "sender, receiver, amount, created_at and signature fields required",
		})
		return
	}

	if err := req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Check if sender's credit card_no belongs to logged in user and if the
	// credit card is still active
	sendersCard, err := database.GetCreditCard(user.Id, req.Sender, true)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error transferring funds",
		})
		return
	}

	// Load sender's public key to verify signature on send funds request
	sendersPubKey, err := encrypt.LoadPublicKeyFromBytes(sendersCard.PublicKey)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error decoding senders public key",
		})
		return
	}

	digest := req.Hash()
	hexDigest := hex.EncodeToString(digest)
	log.Println("Hash server side: ", hexDigest)

	ok = ecdsa.VerifyASN1(sendersPubKey, digest, req.signature)
	if !ok {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "Error transferring funds. Signature verification failed",
		})
		return
	}

	err = database.CreateTransaction(
		req.Sender, req.Receiver, req.Amount,
		req.CreatedAt, req.Signature,
	)
	if err != nil {
		log.Printf("Error transferring funds; %v\n", err)

		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error transferring funds",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Transaction completed successfully",
	})
}

type requestFundsRequest struct {
	Sender    string  `json:"sender"`   // One being asked to pay
	Receiver  string  `json:"receiver"` // One asking for funds
	Amount    float64 `json:"amount"`
	CreatedAt string  `json:"created_at"` // ISO 8601 string

	// Hex-encoded signature
	// Receiver is the one who signs the request this time
	Signature string `json:"signature"`

	signature []byte
}

func (req *requestFundsRequest) Validate() error {
	if err := validateCreditCardNo(req.Sender); err != nil {
		return err
	}
	if err := validateCreditCardNo(req.Receiver); err != nil {
		return err
	}
	if req.Sender == req.Receiver {
		return fmt.Errorf("sender and receiver cannot be the same account")
	}
	if err := validateAmount(req.Amount); err != nil {
		return err
	}

	signature, err := validateSignature(req.Signature)
	if err != nil {
		return err
	}
	req.signature = signature
	return nil
}

// A user can request funds from another user
func RequestFundsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "You are not authorized to view this resource",
		})
		return
	}

	var req requestFundsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "sender, receiver, amount and signature fields required",
		})
		return
	}

	if err := req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// Check if receiver's credit card belongs to currently logged in user
	receiversCard, err := database.GetCreditCard(user.Id, req.Receiver, true)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error requesting funds",
		})
		return
	}

	receiversPubKey, err := encrypt.LoadPublicKeyFromBytes(receiversCard.PublicKey)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error decoding receiver's public key",
		})
		return
	}

	// Hash request data and verify receiver's signature
	hash := sha256.New()
	hash.Write([]byte(req.Sender))
	hash.Write([]byte(req.Receiver))
	hash.Write([]byte(fmt.Sprintf("%f", req.Amount)))
	digest := hash.Sum(nil)

	ok = ecdsa.VerifyASN1(receiversPubKey, digest, req.signature)
	if !ok {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "Error requesting funds. Signature verification failed",
		})
		return
	}

	// Add record to database
	err = database.CreateRequestFunds(
		req.Sender, req.Receiver, req.Amount,
		req.CreatedAt, req.Signature,
	)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error requesting funds",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Funds requested successfully",
	})
}
