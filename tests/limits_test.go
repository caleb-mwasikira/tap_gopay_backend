package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
)

func TestSetOrUpdateLimit(t *testing.T) {
	requireLogin(tommy)

	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Setup a spending limit on tommy's wallet
	limit := handlers.INITIAL_DEPOSIT * rand.Float64()

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
	leesWallet, err := createWallet(lee)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Test spending limit is not exceeded by sending amount > limit
	resp, err = sendMoney(
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		limit+1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusConflict)
	resp.Body.Close()

	// Test spending limit is not exceeded by sending small amounts that are > limit
	var totalAmountSpent float64 = 0

	for totalAmountSpent < limit {
		amount := 1 + utils.RoundFloat(10*rand.Float64(), 2)

		resp, err = sendMoney(
			tommysWallet.WalletAddress,
			leesWallet.WalletAddress,
			tommy,
			float64(amount),
		)
		if err != nil {
			t.Fatalf("Error transferring funds; %v\n", err)
		}

		if (totalAmountSpent + amount) > limit {
			expectStatus(t, resp, http.StatusConflict)
			break
		} else {
			expectStatus(t, resp, http.StatusOK)
			totalAmountSpent += amount
		}
		resp.Body.Close()
	}
}
