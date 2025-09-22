package tests

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

// Signs data using user's private key loaded from file.
// Returns signature, a hash of the public key to verify signature and
// an error is any exists
func signPayload(email string, data []byte) ([]byte, []byte, error) {
	// Load user's private key from file
	privKeyPath := filepath.Join("keys", fmt.Sprintf("%v.key", email))
	privKey, err := encrypt.LoadPrivateKeyFromFile(privKeyPath)
	if err != nil {
		return nil, nil, err
	}

	// Sign send funds request
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, data)
	if err != nil {
		return nil, nil, err
	}

	// Tell server which public key to use to verify signature
	pubKeyBytes, err := encrypt.PemEncodePublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	pubKeyHash := sha256.Sum256(pubKeyBytes)

	return signature, pubKeyHash[:], nil
}

func sendMoney(
	serverUrl string,
	sender string,
	receiver string,
	loginUser User,
	amount float64,
) (*http.Response, error) {
	requireLogin(loginUser, serverUrl)

	fee, err := getTransactionFee(serverUrl, amount)
	if err != nil {
		return nil, fmt.Errorf("error fetching transaction fees; %v", err)
	}

	req := handlers.TransactionRequest{
		Sender:    sender,
		Receiver:  receiver,
		Amount:    amount,
		Fee:       fee,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	log.Printf("Sending funds from '%v' to '%v'\n", sender, receiver)

	// Sign transaction details
	signature, pubKeyHash, err := signPayload(loginUser.Email, req.Hash())
	if err != nil {
		return nil, fmt.Errorf("Error signing data; %v", err)
	}
	req.Signature = base64.StdEncoding.EncodeToString(signature)
	req.PublicKeyHash = base64.StdEncoding.EncodeToString(pubKeyHash)

	// Send sends fund request to server
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	return http.Post(serverUrl+"/send-money", jsonContentType, bytes.NewBuffer(body))
}

func TestSendMoney(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	tommysWallet, err := createWallet(testServer.URL, tommy)
	if err != nil {
		t.Fatalf("Error fetching users wallet; %v\n", err)
	}

	leesWallet, err := createWallet(testServer.URL, lee)
	if err != nil {
		t.Fatalf("Error fetching users wallet; %v\n", err)
	}

	// Test: Transfer funds from one wallet to another
	resp, err := sendMoney(
		testServer.URL,
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestSendMoneyViaPhoneNo(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	tommysWallet, err := createWallet(testServer.URL, tommy)
	if err != nil {
		t.Fatalf("Error fetching users wallet; %v\n", err)
	}

	leesWallet, err := createWallet(testServer.URL, lee)
	if err != nil {
		t.Fatalf("Error fetching users wallet; %v\n", err)
	}

	resp, err := sendMoney(
		testServer.URL,
		tommysWallet.PhoneNo,
		leesWallet.PhoneNo,
		tommy,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)
}

func getTransactions(serverUrl, walletAddress string) ([]database.Transaction, error) {
	resp, err := http.Get(serverUrl + fmt.Sprintf("/recent-transactions/%v", walletAddress))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if server returns a list of transactions
	var notifications []database.Transaction
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	return notifications, err
}

func TestGetRecentTransactions(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Fetch one of tommy's wallets
	tommysWallet, err := createWallet(testServer.URL, tommy)
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v\n", err)
	}

	// Get all transactions made by that wallet
	requireLogin(tommy, testServer.URL)

	_, err = getTransactions(testServer.URL, tommysWallet.WalletAddress)
	if err != nil {
		t.Errorf("Error fetching wallet transactions; %v\n", err)
	}
}

func TestGetTransaction(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(tommy, testServer.URL)

	// Fetch one of tommy's wallet
	tommysWallet, err := createWallet(testServer.URL, tommy)
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v\n", err)
	}

	// Get all transactions made by tommy's wallet
	transactions, err := getTransactions(testServer.URL, tommysWallet.WalletAddress)
	if err != nil {
		t.Fatalf("Error fetching wallet transactions; %v\n", err)
	}

	// Fetch one transaction
	transaction := randomChoice(transactions)
	if transaction == nil {
		t.Fatalf("At least one transaction required in database for test to complete")
	}

	resp, err := http.Get(testServer.URL + fmt.Sprintf("/transactions/%v", transaction.TransactionCode))
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	body := expectStatus(t, resp, http.StatusOK)

	var fetchedTransaction database.Transaction
	err = json.Unmarshal(body, &fetchedTransaction)
	if err != nil {
		t.Errorf("Expected transaction but got garbage data")
	}
}

func TestSendingInvalidAmount(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Create wallet for lee
	leesWallet, err := createWallet(testServer.URL, lee)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	// Create wallet for tommy
	tommysWallet, err := createWallet(testServer.URL, tommy)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	// Test sending amount > INITIAL_DEPOSIT
	resp, err := sendMoney(
		testServer.URL,
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		handlers.INITIAL_DEPOSIT+1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusInternalServerError)

	// Test sending negative amount
	resp, err = sendMoney(
		testServer.URL,
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		-10.0,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusBadRequest)
}
