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

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

func sendFunds(
	testServerUrl string,
	sender, receiver, sendersEmail string,
	amount float64,
) (*http.Response, error) {
	req := handlers.SendFundsRequest{
		Sender:    sender,
		Receiver:  receiver,
		Amount:    amount,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}

	log.Printf("Sending funds from %v to %v\n", sender, receiver)

	// Load user's private key from file
	privKeyPath := filepath.Join("keys", fmt.Sprintf("%v.key", sendersEmail))
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

	return http.Post(testServerUrl+"/send-funds", jsonContentType, bytes.NewBuffer(body))
}

func TestSendFunds(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	email := testEmail
	password := testPassword
	requireLogin(email, password, testServer.URL)

	// Get logged in user's credit cards
	creditCards, err := getAllCreditCards(testServer.URL)
	if err != nil {
		t.Fatalf("Error fetching user's credit cards; %v\n", err)
	}

	// Send funds from one credit card to another.
	if len(creditCards) < 2 {
		t.Fatalf("Minimum of 2 credit cards required for testing")
	}

	sender := creditCards[0].CardNo
	receiver := creditCards[1].CardNo
	amount := handlers.MIN_AMOUNT

	resp, err := sendFunds(testServer.URL, sender, receiver, email, amount)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)
}
