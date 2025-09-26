package tests

import (
	"bytes"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
)

func createCashPool(
	serverUrl string,
	creator User,
	receiver *database.Wallet,
) (*database.CashPool, error) {
	expiresAt := time.Now().Add(5 * time.Minute)

	req := handlers.CashPoolRequest{
		Name:         "Fundraising",
		Description:  "Raising money for school fees",
		TargetAmount: 150,
		Receiver:     receiver.WalletAddress,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
	}
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	requireLogin(creator)

	resp, err := http.Post(
		serverUrl+"/new-cash-pool",
		jsonContentType,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Extract cash pool from response body
	var cashPool database.CashPool

	err = json.NewDecoder(resp.Body).Decode(&cashPool)
	return &cashPool, err
}

func fundCashPool(cashPool *database.CashPool) (float64, error) {
	var collectedAmount float64 = 0
	targetAmount := cashPool.TargetAmount
	users := []User{tommy, lee}

	for collectedAmount < targetAmount {
		amount := utils.RoundFloat(targetAmount*rand.Float64(), 2)

		user := randomChoice(users)

		usersWallet, err := createWallet(*user)
		if err != nil {
			return 0.0, err
		}

		resp, err := sendMoney(
			usersWallet.WalletAddress,
			cashPool.WalletAddress,
			*user,
			amount,
		)
		if err != nil {
			return 0.0, err
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			collectedAmount += amount
		}
	}
	return collectedAmount, nil
}

func TestCreateCashPool(t *testing.T) {
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	cashPool, err := createCashPool(testServer.URL, tommy, tommysWallet)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	_, err = fundCashPool(cashPool)
	if err != nil {
		t.Fatalf("Error funding cash pool; %v\n", err)
	}

	// With the target amount reached, we expect cash pool
	// to be in funded status
	resp, err := http.Get(
		testServer.URL + "/cash-pools/" + cashPool.WalletAddress,
	)
	if err != nil {
		t.Fatalf("Error fetching cash pool; %v\n", err)
	}
	defer resp.Body.Close()

	body := expectStatus(t, resp, http.StatusOK)

	var fetchedCashPool database.CashPool

	err = json.Unmarshal(body, &fetchedCashPool)
	if err != nil {
		t.Fatalf("Error unmarshalling response body; %v\n", err)
	}

	if fetchedCashPool.Status != "funded" {
		t.Fatalf("Expected cash pool to be in 'funded' status but got '%v' status", fetchedCashPool.Status)
	}

	if fetchedCashPool.CollectedAmount < cashPool.TargetAmount {
		t.Fatalf("Expected cash pool to have achieved its target amount")
	}
}

func TestCashPoolDeposit(t *testing.T) {
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	cashPool, err := createCashPool(testServer.URL, tommy, tommysWallet)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	_, err = fundCashPool(cashPool)
	if err != nil {
		t.Fatalf("Error funding cash pool; %v\n", err)
	}

	// Test sending more funds into cash pool after target amount is reached.
	// Should return error
	resp, err := sendMoney(
		tommysWallet.WalletAddress,
		cashPool.WalletAddress,
		tommy,
		10,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)
	resp.Body.Close()
}

func TestCashPoolWithdrawal(t *testing.T) {
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	cashPool, err := createCashPool(
		testServer.URL,
		tommy,
		tommysWallet,
	)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	collectedAmount, err := fundCashPool(cashPool)
	if err != nil {
		t.Fatalf("Error funding cash pool; %v\n", err)
	}

	// Test withdrawing from cash pool as non cash pool creator.
	// Should return an error
	leesWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	resp, err := sendMoney(
		cashPool.WalletAddress,
		leesWallet.WalletAddress,
		lee,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)
	resp.Body.Close()

	// Test sending of funds from cash pool to wrong receiver
	resp, err = sendMoney(
		cashPool.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		cashPool.TargetAmount,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)
	resp.Body.Close()

	// Test withdrawing more funds than collected amount.
	// Should return an error
	resp, err = sendMoney(
		cashPool.WalletAddress,
		cashPool.Receiver.WalletAddress,
		tommy,
		collectedAmount+1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)
	resp.Body.Close()
}
