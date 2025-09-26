package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

var (
	tommy = NewUser("tommy", "iamtommy@gmail.com", "tommyhasagun")
	lee   = NewUser("leejohnson", "leejohnson@gmail.com", "johnsonsandjohnsons")
	bob   = NewUser("bobthebuilder", "canwefixit@gmail.com", "yeswecan")

	cookiesCache = map[string][]*http.Cookie{}
)

type User struct {
	Username string
	Email    string
	Password string
	PhoneNo  string
}

func randomPhoneNo() string {
	str := strings.Builder{}
	str.WriteString("07") // Kenyan phone number

	const phoneNoLength int = 8
	for range phoneNoLength {
		num := rand.IntN(10)
		str.WriteString(fmt.Sprintf("%d", num))
	}
	return str.String()
}

func NewRandomUser() User {
	return User{
		Username: gofakeit.Username(),
		Email:    gofakeit.Email(),
		Password: "2856",
		PhoneNo:  randomPhoneNo(),
	}
}

func NewUser(username, email, password string) User {
	return User{
		Username: username,
		Email:    email,
		Password: password,
		PhoneNo:  randomPhoneNo(),
	}
}

func createAccount(serverUrl string, user User) (*http.Response, error) {
	path := filepath.Join("keys", fmt.Sprintf("%v.key", user.Email))

	privKey, err := generatePrivateKey(path, user.Password)
	if err != nil {
		return nil, err
	}

	// Upload public key
	pubKeyBytes, err := encrypt.PemEncodePublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, err
	}

	req := handlers.RegisterRequest{
		Username:  user.Username,
		Email:     user.Email,
		Password:  user.Password,
		PhoneNo:   user.PhoneNo,
		PublicKey: base64.StdEncoding.EncodeToString(pubKeyBytes),
	}
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	// Send request
	resp, err := http.Post(serverUrl+"/auth/register", jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func TestRegister(t *testing.T) {
	user := NewRandomUser()

	resp, err := createAccount(testServer.URL, user)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)
}

func TestLogin(t *testing.T) {
	// Try accessing protected resource without logging in
	resp, err := http.Get(testServer.URL + "/verify-login")
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()

	email := tommy.Email
	password := tommy.Password

	// Fetch or generate user's private and public keys
	privKey, err := getPrivateKey(email)
	if err != nil {
		t.Fatalf("Error fetching user's private key; %v\n", err)
	}

	// Upload public key to server
	pubKeyBytes, err := encrypt.PemEncodePublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("Error PEM encoding public key; %v\n", err)
	}

	req := handlers.LoginRequest{
		Email:     email,
		Password:  password,
		PublicKey: base64.StdEncoding.EncodeToString(pubKeyBytes),
	}
	body, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling login request; %v\n", err)
	}

	resp, err = http.Post(testServer.URL+"/auth/login", jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making login request; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	cookies := resp.Cookies()
	var loginCookie *http.Cookie

	for _, cookie := range cookies {
		if cookie.Name == handlers.LOGIN_COOKIE {
			loginCookie = cookie
			break
		}
	}
	if loginCookie == nil {
		t.Fatalf("Expected %v in response but cookie NOT found\n", handlers.LOGIN_COOKIE)
	}

	// Set cookies
	url, err := url.Parse(testServer.URL)
	if err != nil {
		t.Fatalf("Error parsing test server's URL; %v\n", err)
	}
	http.DefaultClient.Jar.SetCookies(url, cookies)

	// Now we try accessing the same protected route, but this time
	// with the correct credentials
	resp, err = http.Get(testServer.URL + "/verify-login")
	if err != nil {
		t.Fatalf("Error making verify login request; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)
}

// Logs in to a users account.
// Sets any cookies returned from server to DefaultClient.Jar
// Returns access token to caller in case they require it
func requireLogin(user User) string {
	// Check if user logged in before
	cookies, ok := cookiesCache[user.Email]
	if ok {
		url, err := url.Parse(testServer.URL)
		if err != nil {
			log.Fatalf("Error parsing url; %v\n", err)
		}
		http.DefaultClient.Jar.SetCookies(url, cookies)

		for _, cookie := range cookies {
			if cookie.Name == handlers.LOGIN_COOKIE {
				return cookie.Value
			}
		}

		return ""
	}

	// Fetch or generate user's private and public keys
	privKey, err := getPrivateKey(user.Email)
	if err != nil {
		log.Fatalf("Error fetching user's private key; %v\n", err)
	}

	// Upload public key to server
	pubKeyBytes, err := encrypt.PemEncodePublicKey(&privKey.PublicKey)
	if err != nil {
		log.Fatalf("Error PEM encoding public key; %v\n", err)
	}

	req := handlers.LoginRequest{
		Email:     user.Email,
		Password:  user.Password,
		PublicKey: base64.StdEncoding.EncodeToString(pubKeyBytes),
	}
	body, err := json.Marshal(&req)
	if err != nil {
		log.Fatalf("Error marshalling login request; %v\n", err)
	}

	resp, err := http.Post(testServer.URL+"/auth/login", jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("Error making login request; %v\n", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Expected statusOK but got %v\n", resp.Status)
	}

	// Extract login cookies and set them in cookiejar
	cookies = resp.Cookies()

	url, err := url.Parse(testServer.URL)
	if err != nil {
		log.Fatalf("Error parsing url; %v\n", err)
	}
	http.DefaultClient.Jar.SetCookies(url, cookies)

	// Cache cookies
	cookiesCache[user.Email] = cookies

	for _, cookie := range cookies {
		if cookie.Name == handlers.LOGIN_COOKIE {
			return cookie.Value
		}
	}
	return ""
}

// Cannot test ForgotPassword and ResetPassword handlers as they require
// access to external resource (sending emails) that we do not control
