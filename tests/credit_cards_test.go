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
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

func requireLogin(email, password, serverUrl string) {
	// Get user's password and generate longer password using KDF
	argon2Key, err := encrypt.DeriveKey(password, nil)
	if err != nil {
		log.Fatalf("Error generating KDF password; %v\n", err)
	}

	// Fetch or generate user's private key for signing
	path := filepath.Join("keys", fmt.Sprintf("%v.key", email))
	privKey, err := getPrivateKey(path, argon2Key.Key)
	if err != nil {
		log.Fatalf("Error fetching user's private key; %v\n", err)
	}

	// Sign user's email
	digest := sha256.Sum256([]byte(email))
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, digest[:])
	if err != nil {
		log.Fatalf("Error signing user's email; %v\n", err)
	}

	req := handlers.LoginRequest{
		Email:     email,
		Signature: base64.StdEncoding.EncodeToString(signature),
	}
	body, err := json.Marshal(&req)
	if err != nil {
		log.Fatalf("Error marshalling login request; %v\n", err)
	}

	resp, err := http.Post(serverUrl+"/auth/login", jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("Error making login request; %v\n", err)
	}
	defer resp.Body.Close()

	printResponse(resp)

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Expected statusOK but got %v\n", resp.Status)
	}

	// Extract login cookies and set them in cookiejar
	url, err := url.Parse(serverUrl)
	if err != nil {
		log.Fatalf("Error parsing url; %v\n", err)
	}
	http.DefaultClient.Jar.SetCookies(url, resp.Cookies())
}

func TestNewCreditCard(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	email := testEmail
	password := testPassword
	requireLogin(email, password, testServer.URL)

	// Send public key to server
	resp, err := http.Post(
		testServer.URL+"/new-credit-card", jsonContentType, nil,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	body := printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)

	// Check if request body contains created credit card
	var creditCard database.CreditCard
	err = json.Unmarshal(body, &creditCard)
	if err != nil {
		t.Fatalf("Expected response body to be CreditCard but got garbage data")
	}
}

func TestGetAllCreditCards(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(testEmail, testPassword, testServer.URL)

	url := testServer.URL + "/credit-cards"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	body := printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)

	var results []*database.CreditCard
	err = json.Unmarshal(body, &results)
	if err != nil {
		t.Fatalf("Expected an []CreditCard in response body but got garbage data")
	}
}

func randomChoice[T any](items []T) *T {
	if len(items) == 0 {
		return nil
	}
	index := mrand.IntN(len(items))
	item := items[index]
	return &item
}

func getAllCreditCards(serverUrl string) ([]database.CreditCard, error) {
	log.Println("Fetching user's credit cards...")

	url := serverUrl + "/credit-cards"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var creditCards []database.CreditCard
	err = json.NewDecoder(resp.Body).Decode(&creditCards)

	return creditCards, err
}

func TestGetCreditCard(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(testEmail, testPassword, testServer.URL)

	creditCards, err := getAllCreditCards(testServer.URL)
	if err != nil {
		t.Fatalf("Error fetching user's credit cards; %v\n", err)
	}
	// We are going to fetch this one credit card from server
	creditCard := randomChoice(creditCards)
	if creditCard == nil {
		t.Logf("Logged in user has no credit card in their name")
		return
	}

	resp, err := http.Get(
		testServer.URL + fmt.Sprintf("/credit-cards/%v", creditCard.CardNo),
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	body := printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)

	// Check if request body contains fetched credit card
	var fetchedCreditCard database.CreditCard
	err = json.Unmarshal(body, &fetchedCreditCard)
	if err != nil {
		t.Fatalf("Expected response body to be CreditCard but got garbage data")
	}

	if fetchedCreditCard.CardNo != creditCard.CardNo {
		t.Fatalf("Expected to fetch credit card with cardNo %v but got %v\n", fetchedCreditCard.CardNo, creditCard.CardNo)
	}
}

func TestFreezeCreditCard(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(testEmail, testPassword, testServer.URL)

	// Get logged in user's credit cards
	creditCards, err := getAllCreditCards(testServer.URL)
	if err != nil {
		t.Fatalf("Error fetching user's credit cards; %v\n", err)
	}

	if len(creditCards) < 2 {
		t.Fatalf("Minimum of 2 credit cards required for this test")
	}

	// Freeze one of them
	creditCard := randomChoice(creditCards)
	if creditCard == nil {
		t.Logf("Logged in user has no credit card in their name")
		return
	}

	log.Printf("Freezing credit card %v\n", creditCard.CardNo)

	url := testServer.URL + fmt.Sprintf("/credit-cards/%v/freeze", creditCard.CardNo)
	resp, err := http.Post(url, jsonContentType, nil)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp.StatusCode, http.StatusOK)
	printResponse(resp)
	resp.Body.Close()

	// Attempt to make transaction on frozen credit card
	frozenCreditCard := creditCard
	var otherCreditCard *database.CreditCard

	for _, cc := range creditCards {
		if cc.CardNo != frozenCreditCard.CardNo {
			otherCreditCard = &cc
			break
		}
	}

	if otherCreditCard == nil {
		t.Fatalf("Minimum of 2 credit cards required for this test")
	}

	amount := handlers.MIN_AMOUNT
	resp, err = transferFunds(
		testServer.URL,
		frozenCreditCard.CardNo, otherCreditCard.CardNo,
		testEmail, amount,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusInternalServerError)
}

func TestActivateCreditCard(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(testEmail, testPassword, testServer.URL)

	creditCards, _ := getAllCreditCards(testServer.URL)
	var frozenCreditCard *database.CreditCard

	for _, cc := range creditCards {
		if !cc.IsActive {
			frozenCreditCard = &cc
			break
		}
	}
	if frozenCreditCard == nil {
		t.Fatalf("This test requires a frozen credit card")
	}

	resp, err := http.Post(
		testServer.URL+fmt.Sprintf("/credit-cards/%v/activate", frozenCreditCard.CardNo),
		jsonContentType,
		nil,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)

	// Attempt to make transaction on activated credit card
	activatedCreditCard := frozenCreditCard
	var otherCreditCard *database.CreditCard

	for _, cc := range creditCards {
		if cc.CardNo != activatedCreditCard.CardNo {
			otherCreditCard = &cc
			break
		}
	}

	if otherCreditCard == nil {
		t.Fatalf("Minimum of 2 credit cards required for this test")
	}

	amount := handlers.MIN_AMOUNT
	resp, err = transferFunds(
		testServer.URL,
		activatedCreditCard.CardNo, otherCreditCard.CardNo,
		testEmail, amount,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)
}
