package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

func getTransactionFee(serverUrl string, amount float64) (float64, error) {
	resp, err := http.Get(serverUrl + fmt.Sprintf("/transaction-fees?amount=%.2f", amount))
	if err != nil {
		return 0, err
	}

	var transactionFee database.TransactionFee
	err = json.NewDecoder(resp.Body).Decode(&transactionFee)
	if err != nil {
		return 0, err
	}

	return transactionFee.Fee, nil
}

func TestCreateTransactionFees(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Test setting up transaction fees from non-admin account
	requireLogin(lee, testServer.URL)

	req := handlers.TransactionFeeRequest{
		MinAmount:     100,
		MaxAmount:     500,
		Fee:           7,
		EffectiveFrom: time.Now(),
	}
	body, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling request; %v\n", err)
	}

	resp, err := http.Post(
		testServer.URL+"/transaction-fees",
		jsonContentType,
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()

	// Test setting up transaction fees from admin account
	// Note: Make sure to setup tommy as an admin in the
	// database for this request to work
	requireLogin(tommy, testServer.URL)

	resp, err = http.Post(
		testServer.URL+"/transaction-fees",
		jsonContentType,
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestGetAllTransactionFees(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	resp, err := http.Get(testServer.URL + "/all-transaction-fees")
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	body := expectStatus(t, resp, http.StatusOK)

	var transactionFees []database.TransactionFee
	err = json.Unmarshal(body, &transactionFees)
	if err != nil {
		t.Error("Expected a list of transaction fees from GetAllTransactionFees")
	}
}
