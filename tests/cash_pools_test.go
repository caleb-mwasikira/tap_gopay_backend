package tests

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
)

func createNewChama(
	creator User,
	name string,
	description string,
	targetAmount float64,
	expiresAt *time.Time,
) (*database.CashPool, error) {
	var expiresAtStr string
	if expiresAt != nil {
		expiresAtStr = expiresAt.Format(time.RFC3339)
	}

	req := handlers.CashPoolRequest{
		Name:         name,
		Description:  description,
		TargetAmount: targetAmount,
		ExpiresAt:    expiresAtStr,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	requireLogin(creator)

	resp, err := http.Post(
		testServer.URL+"/new-chama",
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
			log.Println("Failed multiple times to fund cash pool")
			break
		}

		amount := utils.RoundFloat(handlers.INITIAL_DEPOSIT*rand.Float64(), 2)
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

func TestCreateNewChama(t *testing.T) {
	cashPool, err := createNewChama(
		tommy,
		"Fundraising",
		"Raising money for fun and profit",
		150,
		nil,
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

	expectedPoolType := database.Chama
	if fetchedCashPool.PoolType != expectedPoolType {
		t.Fatalf("Expected '%v' cash pool type but got '%v'\n", expectedPoolType, fetchedCashPool.PoolType)
	}
}

func TestCashPoolDeposit(t *testing.T) {
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	cashPool, err := createNewChama(
		tommy,
		"Fundraising",
		"Raising money for fun and profit",
		150,
		nil,
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

func TestChamaWithdrawal(t *testing.T) {
	cashPool, err := createNewChama(
		tommy,
		"Fundraising",
		"Raising money for fun and profit",
		150,
		nil,
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

	// For other cash pools apart from chama,
	// we will have to test if sending to the wrong receiver
	// fails

	// Test withdrawing more funds than collected amount.
	// Should return an error
	resp, err = sendMoney(
		cashPool.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		collectedAmount+1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)
	resp.Body.Close()
}

func checkRefunds(t *testing.T, deposits []cashPoolDeposit, duration time.Duration) {
	maxDuration := duration + (5 * time.Second)

	select {
	case <-time.After(maxDuration):
		t.Fatalf("Tired of waiting")

	case <-time.After(duration):
		// All users who sent funds to the cash pool
		// should be refunded.
		for _, deposit := range deposits {
			expectedBalance := deposit.original

			wallet, err := getWallet(deposit.user, deposit.walletAddress)
			if err != nil {
				t.Errorf("Error fetching wallet; %v\n", err)
			}

			if wallet.Balance != expectedBalance {
				t.Errorf("%v Wallet '%v' was never refunded KSH %v deposited into expired cash pool %v\n", COLOR_RED, wallet.WalletAddress, deposit.amountSent, COLOR_RESET)
			} else {
				log.Printf("%v Wallet '%v' refunded KSH %v deposited into expired cash pool %v\n", COLOR_GREEN, wallet.WalletAddress, deposit.amountSent, COLOR_RESET)
			}
		}
	}
}

func TestChamaRefund(t *testing.T) {
	var targetAmount float64 = 150

	// Create chama with expiry time to check if funds
	// are refunded to users after chama expires
	expiresAt := time.Now().Add(5 * time.Second)

	cashPool, err := createNewChama(
		tommy,
		"Fundraising",
		"Raising money for fun and profit",
		150,
		&expiresAt,
	)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	// Fund cash pool halfway, then wait for it to expire.
	_, deposits, err := fundCashPool(cashPool, targetAmount/2)
	if err != nil {
		t.Fatalf("Error funding cash pool; %v\n", err)
	}

	checkRefunds(t, deposits, 15*time.Second)
}

func TestRemoveCashPool(t *testing.T) {
	// Create cash pool, delete it and check if users who
	// deposited into the cash pool are refunded

	cashPool, err := createNewChama(
		tommy,
		"Fundraising",
		"Raising money for fun and profit",
		150,
		nil,
	)
	if err != nil {
		t.Fatalf("Error creating cash pool; %v\n", err)
	}

	_, deposits, err := fundCashPool(cashPool, 100)
	if err != nil {
		t.Fatalf("Error funding cash pool; %v\n", err)
	}

	requireLogin(tommy)

	req, err := http.NewRequest(
		http.MethodDelete,
		testServer.URL+"/cash-pools/"+cashPool.WalletAddress,
		nil,
	)
	if err != nil {
		t.Fatalf("Error creating request; %v\n", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Error removing cash pool; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	checkRefunds(t, deposits, 15*time.Second)
}
