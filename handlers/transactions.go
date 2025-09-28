package handlers

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/go-chi/chi/v5"
)

type TransactionRequest struct {
	Sender    string  `json:"sender" validate:"account"`
	Receiver  string  `json:"receiver" validate:"account"`
	Amount    float64 `json:"amount" validate:"amount"`
	Fee       float64 `json:"fee" validate:"min=0"`
	Timestamp string  `json:"timestamp"` // Time when transaction was initiated by the client

	Signature string `json:"signature" validate:"signature"` // Base64 encoded signature

	// Base64 encoded hash of public key
	// that should be used to verify signature
	PublicKeyHash string `json:"public_key_hash" validate:"public_key_hash"`
}

func (req TransactionRequest) Hash() []byte {
	data := fmt.Sprintf("%s|%s|%.2f|%.2f|%s", req.Sender, req.Receiver, req.Amount, req.Fee, req.Timestamp)
	h := sha256.Sum256([]byte(data))
	return h[:]
}

func verifySignature(
	b64EncodedSignature string,
	data []byte,
	email string,
	b64EncodedPubKeyHash string,
) error {
	// Fetch user's public key that was used to sign data
	pubKey, err := database.GetPublicKey(email, b64EncodedPubKeyHash)
	if err != nil {
		return err
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

func SendMoney(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	var req TransactionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	if isValidPhoneNumber(req.Sender) {
		wallets, err := database.GetWalletsOwnedByPhoneNo(
			req.Sender,
			func(w *database.Wallet) bool {
				return w.IsActive
			},
		)
		if err != nil || len(wallets) == 0 {
			err = fmt.Errorf("error fetching wallets owned by '%v'; %v", req.Sender, err)
			api.Errorf(w, "Sender has no active wallet accounts", err)
			return
		}
		req.Sender = wallets[0].WalletAddress
	}

	if isValidPhoneNumber(req.Receiver) {
		wallets, err := database.GetWalletsOwnedByPhoneNo(
			req.Receiver,
			func(w *database.Wallet) bool {
				return w.IsActive
			},
		)
		if err != nil || len(wallets) == 0 {
			err = fmt.Errorf("error fetching wallets owned by '%v'; %v", req.Receiver, err)
			api.Errorf(w, "Receiver has no active wallet accounts", err)
			return
		}
		req.Receiver = wallets[0].WalletAddress
	}

	if req.Sender == req.Receiver {
		api.BadRequest(w, "Sender and receiver share the same account", nil)
		return
	}

	// Check amount is within spending limits
	ok = database.IsWithinSpendingLimits(req.Sender, req.Amount)
	if !ok {
		api.Conflict(w, "Wallet exceeded spending limits")
		return
	}

	// Verify fee amount
	var expectedFee float64

	transactionFee, err := getTransactionFees(req.Amount)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		api.Errorf(w, "Error fetching transaction fees", err)
		return
	}

	if transactionFee != nil {
		expectedFee = transactionFee.Fee
	}

	if req.Fee != expectedFee {
		api.BadRequest(w, "Invalid transaction fees", nil)
		return
	}

	transaction, err := database.CreateTransaction(
		user.Id,
		req.Sender, req.Receiver,
		req.Amount, req.Fee,
		req.Timestamp, req.Signature,
		req.PublicKeyHash,
	)
	if err != nil {
		api.Errorf(w, "Error transferring funds", err)
		return
	}

	go sendNotification(*transaction)

	switch transaction.Status {
	case "confirmed":
		api.OK2(w, transaction)
	case "pending":
		api.Accepted(w, transaction)
	default:
		// rejected
		api.Errorf(w, "Transaction rejected", nil)
	}
}

// A user can request funds from another user
func RequestFunds(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	var req TransactionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	data := req.Hash()
	err = verifySignature(req.Signature, data, user.Email, req.PublicKeyHash)
	if err != nil {
		api.Errorf(w, "Error requesting funds. Signature verification failed", nil)
		return
	}

	transaction, err := database.CreateRequestFunds(
		req.Sender, req.Receiver, req.Amount,
		req.Timestamp, req.Signature,
		req.PublicKeyHash,
	)
	if err != nil {
		api.Errorf(w, "Error requesting funds", err)
		return
	}

	api.OK2(w, transaction)
}

func GetTransaction(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	transactionCode := chi.URLParam(r, "transaction_code")

	if !database.IsSenderOrReceiver(user.Id, transactionCode) {
		api.NotFound(w, fmt.Sprintf("Transaction '%v' not found", transactionCode))
		return
	}

	t, err := database.GetTransaction(transactionCode)
	if err != nil {
		api.Errorf(w, "Error fetching transaction", err)
		return
	}

	api.OK2(w, t)
}

func GetRecentTransactions(w http.ResponseWriter, r *http.Request) {
	walletAddress := chi.URLParam(r, "wallet_address")
	if err := validateWalletAddress(walletAddress); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	transactions, err := database.GetRecentTransactions(walletAddress)
	if err != nil {
		api.Errorf(w, "Error fetching wallet transactions", err)
		return
	}

	api.OK2(w, transactions)
}

type SignTransactionRequest struct {
	Signature string `json:"signature" validate:"signature"` // Base64 encoded signature

	// Base64 encoded hash of public key
	// that should be used to verify signature
	PublicKeyHash string `json:"public_key_hash" validate:"public_key_hash"`
}

func SignTransaction(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	transactionCode := chi.URLParam(r, "transaction_code")

	var req SignTransactionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err = validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	t, err := database.GetTransaction(transactionCode)
	if err != nil {
		message := fmt.Sprintf("Error fetching transaction '%v'", transactionCode)
		api.Errorf(w, message, nil)
		return
	}

	err = verifySignature(
		req.Signature,
		t.Hash(),
		user.Email,
		req.PublicKeyHash,
	)
	if err != nil {
		api.Errorf(w, "Error verifying signature", err)
		return
	}

	err = database.AddSignature(
		user.Id,
		transactionCode,
		req.Signature,
		req.PublicKeyHash,
	)
	if err != nil {
		message := fmt.Sprintf("Error signing transaction '%v'", transactionCode)
		api.Errorf(w, message, err)
		return
	}

	transaction, err := database.GetTransaction(transactionCode)
	if err != nil {
		api.Errorf(w, "Error fetching signed transaction", err)
		return
	}

	api.OK2(w, transaction)
}
