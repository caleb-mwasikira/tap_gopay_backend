package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
)

func getAllTransactionFees(serverUrl string) ([]database.TransactionFee, error) {
	resp, err := http.Get(serverUrl + "/all-transaction-fees")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var fees []database.TransactionFee
	err = json.NewDecoder(resp.Body).Decode(&fees)
	return fees, err
}

func getTransactionFee(amount float64) (float64, error) {
	fee := utils.FindOne(
		cachedTransactionFees,
		func(fee database.TransactionFee) bool {
			return amount >= fee.MinAmount && amount <= fee.MaxAmount
		},
	)
	if fee != nil {
		return fee.Fee, nil
	}

	resp, err := http.Get(testServer.URL + fmt.Sprintf("/transaction-fees?amount=%.2f", amount))
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
	// Test setting up transaction fees from non-admin account
	requireLogin(lee)

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
	requireLogin(tommy)

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
