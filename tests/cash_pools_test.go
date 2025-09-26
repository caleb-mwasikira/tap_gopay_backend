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
	creator User,
	receiver *database.Wallet,
	targetAmount float64,
	expiresAt time.Time,
) (*database.CashPool, error) {
	req := handlers.CashPoolRequest{
		Name:         "Fundraising",
		Description:  "Raising money for school fees",
		TargetAmount: targetAmount,
		Receiver:     receiver.WalletAddress,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
	}
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	requireLogin(creator)

	resp, err := http.Post(
		testServer.URL+"/new-cash-pool",
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

func getCashPool(walletAddress string) (*database.CashPool, error) {
	resp, err := http.Get(testServer.URL + "/cash-pools/" + walletAddress)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cashPool database.CashPool

	err = json.NewDecoder(resp.Body).Decode(&cashPool)
	return &cashPool, err
}

type cashPoolDeposit struct {
	user          User
	walletAddress string
	original      float64
	amountSent    float64
}

// Deposits funds into a cash pool.
// Returns total amount deposited, all wallet addresses and the
// amounts they sent and an error if any exists
func fundCashPool(cashPool *database.CashPool, targetAmount float64) (float64, []cashPoolDeposit, error) {
	var depositedAmount float64 = 0
	users := []User{tommy, lee}
	deposits := []cashPoolDeposit{}
	numTries := 0

	for depositedAmount < targetAmount {
		if numTries >= 3 {
			break
		}

		amount := utils.RoundFloat(targetAmount*rand.Float64(), 2)

		user := randomChoice(users)

		wallet, err := createWallet(*user)
		if err != nil {
			return 0.0, nil, err
		}

		resp, err := sendMoney(
			wallet.WalletAddress,
			cashPool.WalletAddress,
			*user,
			amount,
		)
		if err != nil {
			return 0.0, nil, err
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			depositedAmount += amount
			deposits = append(deposits, cashPoolDeposit{
				user:          *user,
				walletAddress: wallet.WalletAddress,
				original:      wallet.Balance,
				amountSent:    amount,
			})
		} else {
			numTries++
		}
	}

	depositedAmount = utils.RoundFloat(depositedAmount, 2)

	return depositedAmount, deposits, nil
}

func TestCreateCashPool(t *testing.T) {
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	cashPool, err := createCashPool(
		tommy,
		tommysWallet,
		150,
		time.Now().Add(1*time.Minute),
	)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	_, _, err = fundCashPool(cashPool, cashPool.TargetAmount)
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

	cashPool, err := createCashPool(
		tommy,
		tommysWallet,
		150,
		time.Now().Add(1*time.Minute),
	)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	expectedAmount, _, err := fundCashPool(cashPool, cashPool.TargetAmount)
	if err != nil {
		t.Fatalf("Error funding cash pool; %v\n", err)
	}

	cashPool, err = getCashPool(cashPool.WalletAddress)
	if err != nil {
		t.Fatalf("Error fetching cash pool; %v\n", err)
	}

	if cashPool.CollectedAmount != expectedAmount {
		t.Fatalf("Cash pool collected amount KSH %v does not equal expected amount KSH %v\n", cashPool.CollectedAmount, expectedAmount)
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
		tommy,
		tommysWallet,
		150,
		time.Now().Add(1*time.Minute),
	)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	collectedAmount, _, err := fundCashPool(cashPool, cashPool.TargetAmount)
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

func TestCashPoolRefund(t *testing.T) {
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	var targetAmount float64 = 150

	cashPool, err := createCashPool(
		tommy,
		tommysWallet,
		targetAmount,
		time.Now().Add(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	// Fund cash pool halfway, then wait for it to expire.
	_, cashPoolDeposits, err := fundCashPool(cashPool, targetAmount/2)
	if err != nil {
		t.Fatalf("Error funding cash pool; %v\n", err)
	}

	// Wait for cash pool to expire.
	select {
	case <-time.After(25 * time.Second):
		t.Fatalf("Tired of waiting")

	case <-time.After(20 * time.Second):
		// All users who sent funds to the cash pool
		// should be refunded.
		for _, deposit := range cashPoolDeposits {
			expectedBalance := deposit.original

			wallet, err := getWallet(deposit.user, deposit.walletAddress)
			if err != nil {
				t.Errorf("Error fetching wallet; %v\n", err)
				return
			}

			if wallet.Balance != expectedBalance {
				t.Errorf("%v Wallet '%v' was never refunded KSH %v deposited into expired cash pool %v\n", COLOR_RED, wallet.WalletAddress, deposit.amountSent, COLOR_RESET)
			}
		}
	}

}
