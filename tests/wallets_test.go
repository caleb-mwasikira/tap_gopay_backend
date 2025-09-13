package tests

import (
	"encoding/json"
	"fmt"
	"log"
	mrand "math/rand/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

func TestCreateWallet(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	users := []User{tommy, lee}

	for _, user := range users {
		requireLogin(user, testServer.URL)

		resp, err := http.Post(
			testServer.URL+"/new-wallet", jsonContentType, nil,
		)
		if err != nil {
			t.Fatalf("Error making request; %v\n", err)
		}
		defer resp.Body.Close()

		body := expectStatus(t, resp, http.StatusOK)

		// Check if request body contains created wallet
		var wallet database.Wallet
		err = json.Unmarshal(body, &wallet)
		if err != nil {
			t.Fatalf("Expected response body to be Wallet but got garbage data")
		}
	}
}

func TestGetAllWallets(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(tommy, testServer.URL)

	url := testServer.URL + "/wallets"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	body :=
		expectStatus(t, resp, http.StatusOK)

	var results []*database.Wallet
	err = json.Unmarshal(body, &results)
	if err != nil {
		t.Fatalf("Expected an []Wallet in response body but got garbage data")
	}
}

// Selects random items from list of items.
// If len(items) == 0, returns nil pointer
func randomChoice[T any](items []T) *T {
	if len(items) == 0 {
		return nil
	}

	index := mrand.IntN(len(items))
	item := items[index]
	return &item
}

func getAllWallets(serverUrl string, filter func(database.Wallet) bool) ([]database.Wallet, error) {
	log.Println("Fetching user's wallets...")

	url := serverUrl + "/wallets"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var wallets []database.Wallet
	err = json.NewDecoder(resp.Body).Decode(&wallets)
	if err != nil {
		return nil, err
	}

	if filter == nil {
		return wallets, nil
	}

	filtered := []database.Wallet{}

	for _, wallet := range wallets {
		if filter(wallet) {
			filtered = append(filtered, wallet)
		}
	}

	return filtered, err
}

func TestGetWallet(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(tommy, testServer.URL)

	wallets, err := getAllWallets(testServer.URL, nil)
	if err != nil {
		t.Fatalf("Error fetching user's wallets; %v\n", err)
	}
	// We are going to fetch this one wallet from server
	wallet := randomChoice(wallets)
	if wallet == nil {
		t.Logf("Logged in user has no wallet in their name")
		return
	}

	resp, err := http.Get(
		testServer.URL + fmt.Sprintf("/wallets/%v", wallet.Address),
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	body :=
		expectStatus(t, resp, http.StatusOK)

	// Check if request body contains fetched wallet
	var fetchedWallet database.Wallet
	err = json.Unmarshal(body, &fetchedWallet)
	if err != nil {
		t.Fatalf("Expected response body to be Wallet but got garbage data")
	}

	if fetchedWallet.Address != wallet.Address {
		t.Fatalf("Expected to fetch wallet with walletAddress %v but got %v\n", fetchedWallet.Address, wallet.Address)
	}
}

func getUsersWallet(
	serverUrl string,
	user User,
	filter func(database.Wallet) bool,
) (*database.Wallet, error) {
	requireLogin(user, serverUrl)

	wallets, err := getAllWallets(serverUrl, filter)
	if err != nil {
		return nil, err
	}
	wallet := randomChoice(wallets)
	if wallet == nil {
		return nil, fmt.Errorf("no wallets found")
	}
	return wallet, nil
}

func TestFreezeWallet(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Get one of tommy's active wallets
	tommysWallet, err := getUsersWallet(
		testServer.URL,
		tommy,
		func(wallet database.Wallet) bool {
			return wallet.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v\n", err)
	}

	// Get one of lee's wallets
	leesWallet, err := getUsersWallet(
		testServer.URL,
		lee,
		func(wallet database.Wallet) bool {
			return wallet.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v\n", err)
	}

	// Freeze tommys wallet
	requireLogin(tommy, testServer.URL)

	log.Printf("Freezing wallet %v\n", tommysWallet.Address)

	url := testServer.URL + fmt.Sprintf("/wallets/%v/freeze", tommysWallet.Address)
	resp, err := http.Post(url, jsonContentType, nil)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Attempt to send money using frozen wallet
	// NOTE: If test fails please ensure you have created the verifyTransaction TRIGGER
	// in your database. Check database/sql/transactions.sql file for TRIGGER value
	frozenWallet := tommysWallet

	resp, err = transferFunds(
		testServer.URL,
		frozenWallet.Address,
		leesWallet.Address,
		fmt.Sprintf("%v.key", tommy.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)

	// Test: Attempt to send money to frozen wallet
	requireLogin(lee, testServer.URL)

	resp, err = transferFunds(
		testServer.URL,
		leesWallet.Address,
		frozenWallet.Address,
		fmt.Sprintf("%v.key", lee.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)

}

func TestActivateWallet(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Fetch one of tommy's frozen wallets
	tommysWallet, err := getUsersWallet(
		testServer.URL,
		tommy,
		func(wallet database.Wallet) bool {
			return !wallet.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v", err)
	}

	// Fetch one of lee's active wallets
	leesWallet, err := getUsersWallet(
		testServer.URL,
		lee,
		func(wallet database.Wallet) bool {
			return wallet.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's wallet; %v", err)
	}

	// Activate tommy's frozen wallet
	requireLogin(tommy, testServer.URL)

	resp, err := http.Post(
		testServer.URL+fmt.Sprintf("/wallets/%v/activate", tommysWallet.Address),
		jsonContentType,
		nil,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Attempt to send money using activated wallet
	resp, err = transferFunds(
		testServer.URL,
		tommysWallet.Address,
		leesWallet.Address,
		fmt.Sprintf("%v.key", tommy.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Attempt to send money to activated wallet
	requireLogin(lee, testServer.URL)

	resp, err = transferFunds(
		testServer.URL,
		leesWallet.Address,
		tommysWallet.Address,
		fmt.Sprintf("%v.key", lee.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

}
