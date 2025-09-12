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

var (
	cookiesCache = map[string][]*http.Cookie{}
)

func requireLogin(user User, serverUrl string) {
	// Check if user logged in before
	cookies, ok := cookiesCache[user.Email]
	if ok {
		url, err := url.Parse(serverUrl)
		if err != nil {
			log.Fatalf("Error parsing url; %v\n", err)
		}
		http.DefaultClient.Jar.SetCookies(url, cookies)
		return
	}

	// Get user's password and generate longer password using KDF
	argon2Key, err := encrypt.DeriveKey(user.Password, nil)
	if err != nil {
		log.Fatalf("Error generating KDF password; %v\n", err)
	}

	// Fetch or generate user's private key for signing
	path := filepath.Join("keys", fmt.Sprintf("%v.key", user.Email))
	privKey, err := getPrivateKey(path, argon2Key.Key)
	if err != nil {
		log.Fatalf("Error fetching user's private key; %v\n", err)
	}

	// Sign user's email
	digest := sha256.Sum256([]byte(user.Email))
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, digest[:])
	if err != nil {
		log.Fatalf("Error signing user's email; %v\n", err)
	}

	req := handlers.LoginRequest{
		Email:     user.Email,
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

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Expected statusOK but got %v\n", resp.Status)
	}

	// Extract login cookies and set them in cookiejar
	cookies = resp.Cookies()

	url, err := url.Parse(serverUrl)
	if err != nil {
		log.Fatalf("Error parsing url; %v\n", err)
	}
	http.DefaultClient.Jar.SetCookies(url, cookies)

	// Cache cookies
	cookiesCache[user.Email] = cookies
}

func TestNewCreditCard(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Select random user
	users := []User{tommy, lee}
	user := randomChoice(users)

	requireLogin(*user, testServer.URL)

	resp, err := http.Post(
		testServer.URL+"/new-credit-card", jsonContentType, nil,
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
}

func TestGetAllCreditCards(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(tommy, testServer.URL)

	url := testServer.URL + "/credit-cards"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	body :=
		expectStatus(t, resp, http.StatusOK)

	var results []*database.CreditCard
	err = json.Unmarshal(body, &results)
	if err != nil {
		t.Fatalf("Expected an []CreditCard in response body but got garbage data")
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

func getAllCreditCards(serverUrl string, filter func(database.CreditCard) bool) ([]database.CreditCard, error) {
	log.Println("Fetching user's credit cards...")

	url := serverUrl + "/credit-cards"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var creditCards []database.CreditCard
	err = json.NewDecoder(resp.Body).Decode(&creditCards)
	if err != nil {
		return nil, err
	}

	if filter == nil {
		return creditCards, nil
	}

	filtered := []database.CreditCard{}

	for _, cc := range creditCards {
		if filter(cc) {
			filtered = append(filtered, cc)
		}
	}

	return filtered, err
}

func TestGetCreditCard(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	requireLogin(tommy, testServer.URL)

	creditCards, err := getAllCreditCards(testServer.URL, nil)
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

	body :=
		expectStatus(t, resp, http.StatusOK)

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

func getUsersCreditCard(
	serverUrl string,
	user User,
	filter func(database.CreditCard) bool,
) (*database.CreditCard, error) {
	requireLogin(user, serverUrl)

	creditCards, err := getAllCreditCards(serverUrl, filter)
	if err != nil {
		return nil, err
	}
	creditCard := randomChoice(creditCards)
	if creditCard == nil {
		return nil, fmt.Errorf("no credit cards found")
	}
	return creditCard, nil
}

func TestFreezeCreditCard(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Get one of tommy's active credit cards
	tommysCreditCard, err := getUsersCreditCard(
		testServer.URL,
		tommy,
		func(cc database.CreditCard) bool {
			return cc.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's credit card; %v\n", err)
	}

	// Get one of lee's credit cards
	leesCreditCard, err := getUsersCreditCard(
		testServer.URL,
		lee,
		func(cc database.CreditCard) bool {
			return cc.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's credit card; %v\n", err)
	}

	// Freeze tommys credit card
	requireLogin(tommy, testServer.URL)

	log.Printf("Freezing credit card %v\n", tommysCreditCard.CardNo)

	url := testServer.URL + fmt.Sprintf("/credit-cards/%v/freeze", tommysCreditCard.CardNo)
	resp, err := http.Post(url, jsonContentType, nil)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Attempt to send money using frozen credit card
	frozenCreditCard := tommysCreditCard

	resp, err = transferFunds(
		testServer.URL,
		frozenCreditCard.CardNo,
		leesCreditCard.CardNo,
		fmt.Sprintf("%v.key", tommy.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)

	// Test: Attempt to send money to frozen credit card
	requireLogin(lee, testServer.URL)

	resp, err = transferFunds(
		testServer.URL,
		leesCreditCard.CardNo,
		frozenCreditCard.CardNo,
		fmt.Sprintf("%v.key", lee.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusInternalServerError)

}

func TestActivateCreditCard(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Fetch one of tommy's frozen credit cards
	tommysCreditCard, err := getUsersCreditCard(
		testServer.URL,
		tommy,
		func(cc database.CreditCard) bool {
			return !cc.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's credit card; %v", err)
	}

	// Fetch one of lee's active credit cards
	leesCreditCard, err := getUsersCreditCard(
		testServer.URL,
		lee,
		func(cc database.CreditCard) bool {
			return cc.IsActive
		},
	)
	if err != nil {
		t.Fatalf("Error fetching user's credit card; %v", err)
	}

	// Activate tommy's frozen credit card
	requireLogin(tommy, testServer.URL)

	resp, err := http.Post(
		testServer.URL+fmt.Sprintf("/credit-cards/%v/activate", tommysCreditCard.CardNo),
		jsonContentType,
		nil,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Attempt to send money using activated credit card
	resp, err = transferFunds(
		testServer.URL,
		tommysCreditCard.CardNo,
		leesCreditCard.CardNo,
		fmt.Sprintf("%v.key", tommy.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Test: Attempt to send money to activated credit card
	requireLogin(lee, testServer.URL)

	resp, err = transferFunds(
		testServer.URL,
		leesCreditCard.CardNo,
		tommysCreditCard.CardNo,
		fmt.Sprintf("%v.key", lee.Email),
		1,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

}
