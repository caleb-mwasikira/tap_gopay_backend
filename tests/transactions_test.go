package tests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/gorilla/websocket"
)

func transferFunds(
	serverUrl string,
	sender, receiver string,
	sendersPrivKeyFilename string,
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
	privKeyPath := filepath.Join("keys", sendersPrivKeyFilename)
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

	// Tell server which public key to use to verify signature
	pubKeyBytes, err := encrypt.PemEncodePublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, err
	}
	pubKeyHash := sha256.Sum256(pubKeyBytes)
	req.PublicKeyHash = base64.StdEncoding.EncodeToString(pubKeyHash[:])

	// Send sends fund request to server
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	return http.Post(serverUrl+"/transfer-funds", jsonContentType, bytes.NewBuffer(body))
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
	var notifications []database.Transaction
	err = json.NewDecoder(resp.Body).Decode(&notifications)
	return notifications, err
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

func waitForNotifications(
	ctx context.Context,
	user User,
	serverUrl string,
	notifications chan<- database.Transaction,
) error {
	accessToken := requireLogin(user, serverUrl)

	log.Printf("%v waiting for transaction notifications\n", user.Username)

	u, err := url.Parse(serverUrl)
	if err != nil {
		return err
	}

	// Establish WebSocket connection
	rawUrl := "ws://" + u.Host + "/ws-notifications"
	header := http.Header{}
	header.Add("Authorization", fmt.Sprintf("Bearer %v", accessToken))

	conn, resp, err := websocket.DefaultDialer.Dial(rawUrl, header)
	if err != nil {
		if resp != nil {
			printResponse(resp, COLOR_RED)
		}
		return err
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var transaction database.Transaction

			err = conn.ReadJSON(&transaction)
			if err != nil {
				return fmt.Errorf("Expected message to be of type Transaction but found garbage data")
			}

			notifications <- transaction
		}
	}
}

func TestWsNotifyReceivedFunds(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Get one of tommy's active wallets
	tommysWallet, err := getUsersWallet(testServer.URL, tommy, func(w database.Wallet) bool {
		return w.IsActive
	})
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v\n", err)
	}

	// Get one of lee's active wallets
	leesWallet, err := getUsersWallet(testServer.URL, lee, func(w database.Wallet) bool {
		return w.IsActive
	})
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v\n", err)
	}

	// Lee waits for received transactions
	notifications := make(chan database.Transaction)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err = waitForNotifications(ctx, lee, testServer.URL, notifications)
		if err != nil {
			log.Fatalf("Error waiting for transaction notifications; %v\n", err)
		}
	}()

	// Sleep for few seconds to give time for lee to start
	// their notification listener
	time.Sleep(2 * time.Second)

	// Tommy sends money to lee
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

	// Lee should receive notification of transaction
	select {
	case <-time.After(20 * time.Second):
		cancel()
		t.Errorf("Tired of waiting for transaction notification")

	case notification := <-notifications:
		log.Printf("Received transaction notification from server %#v\n", notification)
	}
}
