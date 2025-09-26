package tests

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	mrand "math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/nyaruka/phonenumbers"
)

func createWallet(user User) (*database.Wallet, error) {
	requireLogin(user)

	req := handlers.CreateWalletRequest{
		WalletName:    user.Username + randomString(6),
		TotalOwners:   1,
		NumSignatures: 1,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		testServer.URL+"/new-wallet", jsonContentType, bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if request body contains created wallet
	var wallet database.Wallet

	err = json.NewDecoder(resp.Body).Decode(&wallet)
	return &wallet, err
}

func freezeWallet(serverUrl string, user User, walletAddress string) error {
	requireLogin(user)

	url := serverUrl + fmt.Sprintf("/wallets/%v/freeze", walletAddress)
	resp, err := http.Post(url, jsonContentType, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func TestCreateWallet(t *testing.T) {
	users := []User{tommy, lee}

	for _, user := range users {
		_, err := createWallet(user)
		if err != nil {
			t.Fatalf("Error creating wallet; %v\n", err)
		}
	}
}

func TestGetAllWallets(t *testing.T) {
	requireLogin(tommy)

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

func getAllWallets(
	serverUrl string,
	user User,
	filter func(database.Wallet) bool,
) ([]database.Wallet, error) {
	requireLogin(user)

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
	wallets, err := getAllWallets(
		testServer.URL,
		tommy,
		nil,
	)
	if err != nil {
		t.Fatalf("Error fetching user's wallets; %v\n", err)
	}

	// Select random wallet
	originalWallet := randomChoice(wallets)
	if originalWallet == nil {
		t.Logf("Logged in user has no wallet in their name")
		return
	}

	// Fetch same wallet from server
	resp, err := http.Get(
		testServer.URL + fmt.Sprintf("/wallets/%v", originalWallet.WalletAddress),
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	body := expectStatus(t, resp, http.StatusOK)

	// Check if request body contains fetched wallet
	var fetchedWallet database.Wallet

	err = json.Unmarshal(body, &fetchedWallet)
	if err != nil {
		t.Fatalf("Expected response body to be Wallet but got garbage data")
	}

	if fetchedWallet.WalletAddress != originalWallet.WalletAddress {
		t.Fatalf("Expected to fetch wallet with walletAddress %v but got %v\n", fetchedWallet.WalletAddress, originalWallet.WalletAddress)
	}
}

func TestFreezeWallet(t *testing.T) {
	// Create new wallet for tommy
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Create new wallet for lee
	leesWallet, err := createWallet(lee)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Freeze tommys wallet
	err = freezeWallet(testServer.URL, tommy, tommysWallet.WalletAddress)
	if err != nil {
		t.Fatalf("Error freezing user's wallet; %v\n", err)
	}

	// Test: Attempt to send money using frozen wallet
	// NOTE: If test fails please ensure you have created the verifyTransaction TRIGGER
	// in your database. Check database/sql/transactions.sql file for TRIGGER value
	frozenWallet := tommysWallet

	resp, err := sendMoney(
		frozenWallet.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)

	// Test: Attempt to send money to frozen wallet
	resp, err = sendMoney(
		leesWallet.WalletAddress,
		frozenWallet.WalletAddress,
		lee,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)

}

func TestActivateWallet(t *testing.T) {
	// Create wallet
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v", err)
	}

	// Freeze tommy's wallet
	err = freezeWallet(testServer.URL, tommy, tommysWallet.WalletAddress)
	if err != nil {
		t.Fatalf("Error freezing user's wallet; %v", err)
	}

	// Fetch one of lee's active wallets
	leesWallet, err := createWallet(lee)
	if err != nil {
		t.Fatalf("Error creating wallet; %v", err)
	}

	// Activate tommy's frozen wallet
	requireLogin(tommy)

	resp, err := http.Post(
		testServer.URL+fmt.Sprintf("/wallets/%v/activate", tommysWallet.WalletAddress),
		jsonContentType,
		nil,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Attempt to send money using activated wallet
	resp, err = sendMoney(
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Attempt to send money to activated wallet
	resp, err = sendMoney(
		leesWallet.WalletAddress,
		tommysWallet.WalletAddress,
		lee,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

}

func TestGetWalletsOwnedByPhoneNo(t *testing.T) {
	// Create account for random user
	user := NewRandomUser()
	resp, err := createAccount(testServer.URL, user)
	if err != nil {
		t.Fatalf("Error creating user account; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Create wallet for random user
	originalWallet, err := createWallet(user)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Get wallet tied to user's phone number
	fetchedWallets, err := database.GetWalletsOwnedByPhoneNo(user.PhoneNo, nil)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	// Check if original wallet is in fetched wallets
	ok := slices.ContainsFunc(fetchedWallets, func(w *database.Wallet) bool {
		return w.WalletAddress == originalWallet.WalletAddress
	})
	if !ok {
		t.Fatalf("GetWalletsOwnedByPhoneNo returns invalid data; %v\n", err)
	}

	// Randomize phone number input
	dirtyPhone, err := dirtifyPhoneInput(user.PhoneNo)
	if err != nil {
		t.Fatalf("Error dirtying phone input; %v\n", err)
	}

	// Fetch wallet tield to dirtied phone number
	fetchedWallets, err = database.GetWalletsOwnedByPhoneNo(dirtyPhone, nil)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	// Check if original wallet is in fetched wallets
	ok = slices.ContainsFunc(fetchedWallets, func(w *database.Wallet) bool {
		return w.WalletAddress == originalWallet.WalletAddress
	})
	if !ok {
		t.Fatalf("GetWalletsOwnedByPhoneNo returns invalid data; %v\n", err)
	}

}

// Changes phone numbers format just a little bit
// eg from international (+254) to national(07) format
// or add random spaces in between numbers
func dirtifyPhoneInput(phone string) (string, error) {
	num, err := phonenumbers.Parse(phone, "KE")
	if err != nil {
		return "", err
	}

	formats := []phonenumbers.PhoneNumberFormat{
		phonenumbers.E164,
		phonenumbers.NATIONAL,
		phonenumbers.INTERNATIONAL,
	}
	format := randomChoice(formats)
	phone = phonenumbers.Format(num, *format)

	// Randomly add spaces in between numbers
	str := strings.Builder{}

	for _, char := range phone {
		addSpace := mrand.Float64() > 0.5
		if addSpace {
			str.WriteString(" ")
		}
		str.WriteRune(char)
	}

	return str.String(), nil
}

func createMultiSigWallet(
	serverUrl string,
	user User,
	totalOwners, numSignatures uint,
) (*database.Wallet, error) {
	requireLogin(user)

	req := handlers.CreateWalletRequest{
		WalletName:    user.Username + randomString(6),
		TotalOwners:   totalOwners,
		NumSignatures: numSignatures,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		serverUrl+"/new-wallet", jsonContentType, bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body = printResponse(resp, http.StatusOK)

	// Check if request body contains created wallet
	var wallet database.Wallet

	err = json.Unmarshal(body, &wallet)
	return &wallet, err
}

func signTransaction(
	serverUrl string,
	user User,
	transaction database.Transaction,
) (*http.Response, error) {
	requireLogin(user)

	// Load user's private key from file
	privKey, err := getPrivateKey(user.Email)
	if err != nil {
		return nil, err
	}

	// Sign transaction
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, transaction.Hash())
	if err != nil {
		return nil, err
	}

	pubKey := privKey.PublicKey
	pubKeyBytes, err := encrypt.PemEncodePublicKey(&pubKey)
	if err != nil {
		return nil, err
	}
	pubKeyHash := sha256.Sum256(pubKeyBytes)

	req := handlers.SignTransactionRequest{
		Signature:     base64.StdEncoding.EncodeToString(signature),
		PublicKeyHash: base64.StdEncoding.EncodeToString(pubKeyHash[:]),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return http.Post(
		serverUrl+"/transactions/"+transaction.TransactionCode+"/sign-transaction",
		jsonContentType,
		bytes.NewBuffer(body),
	)
}

func TestCreateMultiSigWallet(t *testing.T) {
	const totalOwners uint = 2
	const numSignatures uint = 2
	tommysWallet, err := createMultiSigWallet(
		testServer.URL,
		tommy,
		totalOwners,
		numSignatures,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	// Add lee as owner of tommy's wallet
	rawUrl := testServer.URL + "/wallets/" + tommysWallet.WalletAddress + "/add-owner"
	req := handlers.WalletOwnerRequest{
		Email: lee.Email,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling request body; %v\n", err)
	}

	resp, err := http.Post(rawUrl, jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Send money from tommy's wallet -> lee's wallet as lee
	leesWallet, err := createWallet(lee)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	resp, err = sendMoney(
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		lee,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	body = expectStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	// Extract transaction from response body
	var transaction database.Transaction

	err = json.Unmarshal(body, &transaction)
	if err != nil {
		t.Fatalf("Error unmarshalling response body; %v\n", err)
	}

	if transaction.Status == "confirmed" {
		t.Errorf("Transaction completed illegitimately without signatures from other owners")
	}

	// Transaction requires more signatures
	// So we have tommy also sign transaction
	resp, err = signTransaction(testServer.URL, tommy, transaction)
	if err != nil {
		t.Fatalf("Error signing transaction; %v", err)
	}

	body = expectStatus(t, resp, http.StatusOK)

	// Extract transaction from response body
	err = json.Unmarshal(body, &transaction)
	if err != nil {
		t.Fatalf("Error unmarshalling response body; %v\n", err)
	}

	if transaction.Status != "confirmed" {
		t.Errorf("Expected confirmed transaction status but got '%v' status", transaction.Status)
	}
}

func TestRemoveWalletOwner(t *testing.T) {
	const totalOwners uint = 2
	const numSignatures uint = 2
	tommysWallet, err := createMultiSigWallet(
		testServer.URL,
		tommy,
		totalOwners,
		numSignatures,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	// Add lee as owner of tommy's wallet
	rawUrl := testServer.URL + "/wallets/" + tommysWallet.WalletAddress + "/add-owner"
	req := handlers.WalletOwnerRequest{
		Email: lee.Email,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling request body; %v\n", err)
	}

	resp, err := http.Post(rawUrl, jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Remove lee as owner of tommy's wallet
	resp, err = http.Post(
		testServer.URL+"/wallets/"+tommysWallet.WalletAddress+"/remove-owner",
		jsonContentType,
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	lessWallet, err := createWallet(lee)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Try sending funds from tommy's wallet to lee's wallet
	resp, err = sendMoney(
		tommysWallet.WalletAddress,
		lessWallet.WalletAddress,
		lee,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)
	resp.Body.Close()
}

func TestWalletOwnership(t *testing.T) {
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	leesWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Try sending funds from tommysWallet -> leesWallet as lee
	// Basically stealing funds
	resp, err := sendMoney(
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		lee,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusInternalServerError)

}
