package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

func TestSetOrUpdateLimit(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(tommy, testServer.URL)

	// Create new wallet, so we can max out its limits
	tommysWallet, err := createWallet(testServer.URL, tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Setup a limit on tommy's wallet
	const limit = 10.0

	req := handlers.SetupLimitRequest{
		Period: "week",
		Amount: limit,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling request; %v\n", err)
	}

	resp, err := http.Post(
		testServer.URL+fmt.Sprintf("/wallets/%v/limit", tommysWallet.WalletAddress),
		jsonContentType,
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Get one-of lee's wallets
	leesWallet, err := createWallet(testServer.URL, lee)
	if err != nil {
		t.Fatalf("Error fetching user's wallets; %v\n", err)
	}

	// Test spending limit worked by sending amount > limit
	resp, err = sendMoney(
		testServer.URL,
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		limit*2,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusConflict)
	resp.Body.Close()

	// Test spending limit by sending small amounts that add upto or are > limit
	totalAmountSpent := 0

	for range 10 {
		amount := 1 + rand.IntN(5)

		resp, err = sendMoney(
			testServer.URL,
			tommysWallet.WalletAddress,
			leesWallet.WalletAddress,
			tommy,
			float64(amount),
		)
		if err != nil {
			t.Fatalf("Error transferring funds; %v\n", err)
		}

		if (totalAmountSpent + amount) > limit {
			log.Printf("Total Amount Spent: %v\n", totalAmountSpent)
			log.Printf("Amount: %v\n", amount)
			expectStatus(t, resp, http.StatusConflict)
		} else {
			log.Printf("Total Amount Spent: %v\n", totalAmountSpent)
			log.Printf("Amount: %v\n", amount)
			expectStatus(t, resp, http.StatusOK)
			totalAmountSpent += amount
		}

		resp.Body.Close()
	}
}
