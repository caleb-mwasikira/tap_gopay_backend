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

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

func TestSetOrUpdateLimit(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(tommy, testServer.URL)

	// Create new credit card, so we can max out its limits
	resp, err := http.Post(
		testServer.URL+"/new-credit-card", jsonContentType, nil,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	body := expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	var tommysCreditCard database.CreditCard
	err = json.Unmarshal(body, &tommysCreditCard)
	if err != nil {
		t.Fatal("Expected credit card data type but got garbage data")
	}

	// Setup a limit on tommy's credit card
	const limit = 10.0

	req := handlers.SetupLimitRequest{
		Period: "week",
		Amount: limit,
	}
	body, err = json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling request; %v\n", err)
	}

	resp, err = http.Post(
		testServer.URL+fmt.Sprintf("/credit-cards/%v/limit", tommysCreditCard.CardNo),
		jsonContentType,
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Get one-of lee's credit cards
	leesCreditCard, err := getUsersCreditCard(
		testServer.URL,
		lee,
		func(cc database.CreditCard) bool {
			return cc.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's credit cards; %v\n", err)
	}

	// Test spending limit worked by sending amount > limit
	requireLogin(tommy, testServer.URL)

	resp, err = transferFunds(
		testServer.URL,
		tommysCreditCard.CardNo,
		leesCreditCard.CardNo,
		fmt.Sprintf("%v.key", tommy.Email),
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

		resp, err = transferFunds(
			testServer.URL,
			tommysCreditCard.CardNo,
			leesCreditCard.CardNo,
			fmt.Sprintf("%v.key", tommy.Email),
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
