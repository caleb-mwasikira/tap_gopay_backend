package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"mime/multipart"
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
	req := handlers.LoginRequest{
		Email:    email,
		Password: password,
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

func TestNewCreditCardHandler(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	email := testEmail
	password := testPassword
	requireLogin(email, password, testServer.URL)

	// Creating a new credit card requires sending an ecdsa.PublicKey
	// to the server
	privKey, pubKey, err := encrypt.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Unexpected error generating key pair; %v", err)
	}

	// Upload public key to multipart form
	var buff bytes.Buffer
	multiPartWriter := multipart.NewWriter(&buff)

	writer, err := multiPartWriter.CreateFormFile(handlers.PUB_KEY_FIELD, ".pub")
	if err != nil {
		t.Fatalf("Error creating multipart form file; %v\n", err)
	}

	pubKeyBytes, err := encrypt.PemEncodePublicKey(pubKey)
	if err != nil {
		t.Fatalf("Error PEM encoding public key; %v\n", err)
	}

	_, err = writer.Write(pubKeyBytes)
	if err != nil {
		t.Fatalf("Error writing public key data to multipart form file; %v\n", err)
	}
	multiPartWriter.Close()

	// Send public key to server
	resp, err := http.Post(
		testServer.URL+"/credit-cards",
		multiPartWriter.FormDataContentType(),
		&buff,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	body := expectStatus(t, resp, http.StatusOK)

	// Check if request body contains created credit card
	var creditCard database.CreditCard
	err = json.Unmarshal(body, &creditCard)
	if err != nil {
		t.Fatalf("Expected response body to be CreditCard but got garbage data")
	}

	// We save key pair to a file as we will need it in future requests
	privKeyFilename := fmt.Sprintf("%v.key", creditCard.CardNo)
	pubKeyFilename := fmt.Sprintf("%v.pub", creditCard.CardNo)

	privKeyPath := filepath.Join("keys", privKeyFilename)
	pubKeyPath := filepath.Join("keys", pubKeyFilename)

	err = encrypt.SavePrivateKeyToFile(privKey, privKeyPath, false)
	if err != nil {
		t.Fatalf("Unexpected error saving EC private key to file; %v\n", err)
	}

	err = encrypt.SavePublicKeyToFile(pubKey, pubKeyPath, false)
	if err != nil {
		t.Errorf("Unexpected error saving EC public key to file; %v\n", err)
	}
}

func TestGetCreditCardsHandler(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(testEmail, testPassword, testServer.URL)

	url := testServer.URL + "/credit-cards"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	body := expectStatus(t, resp, http.StatusOK)

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
	index := rand.IntN(len(items))
	item := items[index]
	return &item
}

func getCreditCards(serverUrl string) ([]database.CreditCard, error) {
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

func TestFreezeCreditCardHandler(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(testEmail, testPassword, testServer.URL)

	// Get logged in user's credit cards
	creditCards, err := getCreditCards(testServer.URL)
	if err != nil {
		t.Fatalf("Error fetching user's credit cards; %v\n", err)
	}

	// Freeze one of them
	creditCard := randomChoice(creditCards)
	if creditCard == nil {
		t.Logf("Logged in user has no credit card in their name")
		return
	}

	log.Printf("Freezing credit card %v\n", creditCard.CardNo)

	req := handlers.CreditCardRequest{
		CardNo: creditCard.CardNo,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling request; %v\n", err)
	}

	url := testServer.URL + "/credit-cards/freeze"
	resp, err := http.Post(url, jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	// TODO: Attempt to make transaction on frozen credit card

}
