package tests

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
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

func transferFunds(
	testServerUrl string,
	sender, receiver, privKeyFilename string,
	amount float64,
) (*http.Response, error) {
	req := handlers.TransactionRequest{
		Sender:    sender,
		Receiver:  receiver,
		Amount:    amount,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	log.Printf("Sending funds from '%v' to '%v'\n", sender, receiver)

	// Load user's private key from file
	privKeyPath := filepath.Join("keys", privKeyFilename)
	privKey, err := encrypt.LoadPrivateKeyFromFile(privKeyPath)
	if err != nil {
		return nil, err
	}

	// Sign send funds request
	digest := req.Hash()
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, digest)
	if err != nil {
		return nil, err
	}
	req.Signature = base64.StdEncoding.EncodeToString(signature)

	// Send sends fund request to server
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	return http.Post(testServerUrl+"/transfer-funds", jsonContentType, bytes.NewBuffer(body))
}

func TestTransferFunds(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	tommysWallet, err := getUsersWallet(
		testServer.URL,
		tommy,
		func(wallet database.Wallet) bool {
			return wallet.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching users wallet; %v\n", err)
	}

	leesWallet, err := getUsersWallet(
		testServer.URL,
		lee,
		func(wallet database.Wallet) bool {
			return wallet.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching users wallet; %v\n", err)
	}

	// Test: Transfer funds from one wallet to another
	requireLogin(tommy, testServer.URL)

	resp, err := transferFunds(
		testServer.URL,
		tommysWallet.Address,
		leesWallet.Address,
		fmt.Sprintf("%v.key", tommy.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Transfer funds from one phone number to another
	resp, err = transferFunds(
		testServer.URL,
		tommysWallet.PhoneNo,
		leesWallet.PhoneNo,
		fmt.Sprintf("%v.key", tommy.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
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
	var results []database.Transaction
	err = json.NewDecoder(resp.Body).Decode(&results)
	return results, err
}

func TestGetRecentTransactions(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Fetch one of tommy's wallets
	tommysWallet, err := getUsersWallet(testServer.URL, tommy, nil)
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v\n", err)
	}

	// Get all transactions made by that wallet
	requireLogin(tommy, testServer.URL)

	_, err = getTransactions(testServer.URL, tommysWallet.Address)
	if err != nil {
		t.Errorf("Error fetching wallet transactions; %v\n", err)
	}
}

func TestGetTransaction(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(tommy, testServer.URL)

	// Fetch one of tommy's wallet
	tommysWallet, err := getUsersWallet(testServer.URL, tommy, nil)
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v\n", err)
	}

	// Get all transactions made by tommy's wallet
	transactions, err := getTransactions(testServer.URL, tommysWallet.Address)
	if err != nil {
		t.Fatalf("Error fetching wallet transactions; %v\n", err)
	}

	// Fetch one transaction
	transaction := randomChoice(transactions)
	if transaction == nil {
		t.Fatalf("At least one transaction required in database for test to complete")
	}

	resp, err := http.Get(testServer.URL + fmt.Sprintf("/transactions/%v", transaction.TransactionId))
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
