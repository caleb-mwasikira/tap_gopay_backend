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
	"slices"
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
	Fee       float64 `json:"fee" validate:"min=0"`
	Timestamp string  `json:"timestamp"` // Time when transaction was initiated by the client

	// Base64 encoded hash of public key
	// that should be used to verify signature
	PublicKeyHash string `json:"public_key_hash" validate:"public_key_hash"`
	Signature     string `json:"signature" validate:"signature"` // Base64 encoded signature
}

func (req TransactionRequest) Hash() []byte {
	data := fmt.Sprintf("%s|%s|%.2f|%.2f|%s", req.Sender, req.Receiver, req.Amount, req.Fee, req.Timestamp)
	h := sha256.Sum256([]byte(data))
	return h[:]
}

func getWalletOwnedBy(phone string) (*database.Wallet, error) {
	wallets, err := database.GetAllWalletsOwnedBy(
		phone,
		func(wallet *database.Wallet) bool {
			return wallet.IsActive
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching wallets owned by phone number; %v", err)
	}

	if len(wallets) == 0 {
		return nil, fmt.Errorf("no wallets found owned by phone number")
	}

	// TODO: Return wallet that hasn't exceeded its spending limits

	return wallets[0], nil
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

	// Get user's public key used to sign the transaction
	pubKeyBytes, err := database.GetPublicKey(user.Email, req.PublicKeyHash)
	if err != nil {
		api.Errorf(w, "Error sending funds. Public key not found", err)
		return
	}

	data := req.Hash()
	err = verifySignature(pubKeyBytes, data, req.Signature)
	if err != nil {
		api.Errorf(w, "Error sending funds. Signature verification failed", err)
		return
	}

	if isValidPhoneNumber(req.Sender) {
		wallet, err := getWalletOwnedBy(req.Sender)
		if err != nil {
			api.Errorf(w, "Sender has no active wallet accounts", err)
			return
		}
		req.Sender = wallet.Address
	}

	if isValidPhoneNumber(req.Receiver) {
		wallet, err := getWalletOwnedBy(req.Receiver)
		if err != nil {
			api.Errorf(w, "Receiver has no active wallet accounts", err)
			return
		}
		req.Receiver = wallet.Address
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

	go notifyInterestedParties(*transaction)

	api.OK2(w, transaction)
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

	// Get user's public key used to sign the transaction
	pubKeyBytes, err := database.GetPublicKey(user.Email, req.PublicKeyHash)
	if err != nil {
		api.Errorf(w, "Error sending funds. Public key not found", err)
		return
	}

	data := req.Hash()
	err = verifySignature(pubKeyBytes, data, req.Signature)
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
	transactionCode = strings.TrimSpace(transactionCode)

	if len(transactionCode) < database.TRANSACTION_ID_LEN {
		api.BadRequest(w, "Invalid transaction id", nil)
		return
	}

	t, err := database.GetTransaction(transactionCode)
	if err != nil {
		api.Errorf(w, "Error fetching transaction", err)
		return
	}

	// Get involved parties
	involvedParties, err := database.GetWalletOwners(t.Sender.Address, t.Receiver.Address)
	if err != nil {
		api.Errorf(w, "Error fetching involved parties in transaction", err)
		return
	}

	// A user is authorized to view transaction details
	// if they are among involved parties
	if !slices.Contains(involvedParties, user.Id) {
		api.Unauthorized(w, "You are not authorized to view transaction details")
		return
	}

	api.OK2(w, t)
}

func GetRecentTransactions(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	walletAddress := chi.URLParam(r, "wallet_address")
	if err := validateWalletAddress(walletAddress); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	// Check if wallet address belongs to logged in user
	ok = database.WalletExists(user.Id, walletAddress)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	transactions, err := database.GetRecentTransactions(walletAddress)
	if err != nil {
		api.Errorf(w, "Error fetching wallet transactions", err)
		return
	}

	api.OK2(w, transactions)
}
