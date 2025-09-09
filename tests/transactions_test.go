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
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
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

	tommysCreditCard, err := getUsersCreditCard(
		testServer.URL,
		tommy,
		func(cc database.CreditCard) bool {
			return cc.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching users credit card; %v\n", err)
	}

	leesCreditCard, err := getUsersCreditCard(
		testServer.URL,
		lee,
		func(cc database.CreditCard) bool {
			return cc.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching users credit card; %v\n", err)
	}

	// Test: Transfer funds from one credit card to another
	requireLogin(tommy, testServer.URL)

	resp, err := transferFunds(
		testServer.URL,
		tommysCreditCard.CardNo,
		leesCreditCard.CardNo,
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
		tommysCreditCard.PhoneNo,
		leesCreditCard.PhoneNo,
		fmt.Sprintf("%v.key", tommy.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)
}
