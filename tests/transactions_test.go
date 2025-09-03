package tests

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	mrand "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	h "github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

func TestSendFundsHandler(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	email := testEmail
	password := testPassword
	requireLogin(email, password, testServer.URL)

	// Get logged in user's credit cards
	creditCards, err := getCreditCards(testServer.URL)
	if err != nil {
		t.Fatalf("Error fetching user's credit cards; %v\n", err)
	}

	// Send funds from one credit card to another.
	if len(creditCards) < 2 {
		t.Fatalf("Minimum of 2 credit cards required for testing")
	}

	sender := creditCards[0].CardNo
	receiver := creditCards[1].CardNo
	amount := math.Round(mrand.Float64() * 100)

	req := h.SendFundsRequest{
		Sender:    sender,
		Receiver:  receiver,
		Amount:    amount,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}

	log.Printf("Sending funds from %v to %v\n", sender, receiver)

	// Load senders private key from file
	filename := fmt.Sprintf("%v.key", sender)
	privKeyPath := filepath.Join("keys", filename)
	privKey, err := encrypt.LoadPrivateKeyFromFile(privKeyPath)
	if err != nil {
		t.Fatalf("Error loading private key from file; %v\n", err)
	}

	// Sign send funds request
	digest := req.Hash()
	hexDigest := hex.EncodeToString(digest)
	log.Println("Hash client side: ", hexDigest)

	signature, err := ecdsa.SignASN1(rand.Reader, privKey, digest)
	if err != nil {
		t.Fatalf("Error signing send funds request; %v\n", err)
	}
	req.Signature = hex.EncodeToString(signature)

	// Send sends fund request to server
	body, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling send funds request; %v\n", err)
	}

	resp, err := http.Post(testServer.URL+"/send-funds", jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)
}
