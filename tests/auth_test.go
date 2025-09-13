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
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

var (
	phoneNumbers = []string{
		"+254 130761229", "+254 120760991",
		"+254 736414224", "+254 120754951",
		"+254 737635477", "+254 113216258",
		"+254 729982335", "+254 130132427",
		"+254 745985969", "+254 709367512",
	}
	tommy = NewUser("tommy", "iamtommy@gmail.com", "tommyhasagun")
	lee   = NewUser("leejohnson", "leejohnson@gmail.com", "johnsonsandjohnsons")

	cookiesCache = map[string][]*http.Cookie{}
)

type User struct {
	Username string
	Email    string
	Password string
	Phone    string
}

func NewUser(username, email, password string) User {
	return User{
		Username: username,
		Email:    email,
		Password: password,
		Phone:    *randomChoice(phoneNumbers),
	}
}

func uploadFile(
	multipartWriter *multipart.Writer, fieldName string,
	data []byte,
) error {
	writer, err := multipartWriter.CreateFormFile(fieldName, fieldName)
	if err != nil {
		return err
	}

	// Base64 encode data b4 uploading it
	b64EncodedData := base64.StdEncoding.EncodeToString(data)

	_, err = writer.Write([]byte(b64EncodedData))
	return err
}

func createAccount(testServerUrl string, user User) (*http.Response, error) {
	// First we create a stronger key from the user's password
	key, err := encrypt.DeriveKey(user.Password, nil)
	if err != nil {
		return nil, err
	}

	// Then we use the key to generate user's private key
	path := filepath.Join("keys", fmt.Sprintf("%v.key", user.Email))

	privKey, err := getOrGeneratePrivateKey(path, key.Key)
	if err != nil {
		return nil, err
	}

	// Create multipart/form-data request
	var buff bytes.Buffer
	multipartWriter := multipart.NewWriter(&buff)

	multipartWriter.WriteField("username", user.Username)
	multipartWriter.WriteField("email", user.Email)
	multipartWriter.WriteField("phone_no", user.Phone)

	// Upload public key
	pubKeyBytes, err := encrypt.PemEncodePublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, err
	}

	err = uploadFile(multipartWriter, handlers.PUBLIC_KEY, pubKeyBytes)
	if err != nil {
		return nil, err
	}
	multipartWriter.Close()

	// Send request
	resp, err := http.Post(
		testServerUrl+"/auth/register",
		multipartWriter.FormDataContentType(),
		&buff,
	)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func TestRegister(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	users := []User{tommy, lee}

	for _, user := range users {
		resp, err := createAccount(testServer.URL, user)
		if err != nil {
			t.Fatalf("Error making request; %v\n", err)
		}
		expectStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	}

}

func TestLogin(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Try accessing protected resource without logging in
	resp, err := http.Get(testServer.URL + "/verify-login")
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	expectStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()

	// Login
	email := tommy.Email
	password := tommy.Password

	// Get user's password and generate longer password using KDF
	key, err := encrypt.DeriveKey(password, nil)
	if err != nil {
		t.Fatalf("Error generating KDF password; %v\n", err)
	}

	// Fetch or generate user's private key for signing
	path := filepath.Join("keys", fmt.Sprintf("%v.key", email))
	privKey, err := getOrGeneratePrivateKey(path, key.Key)
	if err != nil {
		t.Fatalf("Error fetching user's private key; %v\n", err)
	}

	// Sign user's email
	digest := sha256.Sum256([]byte(email))
	signature, err := ecdsa.SignASN1(rand.Reader, privKey, digest[:])
	if err != nil {
		t.Fatalf("Error signing user's email; %v\n", err)
	}

	req := handlers.LoginRequest{
		Email:     email,
		Signature: base64.StdEncoding.EncodeToString(signature),
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
	key, err := encrypt.DeriveKey(user.Password, nil)
	if err != nil {
		log.Fatalf("Error generating KDF password; %v\n", err)
	}

	// Fetch or generate user's private key for signing
	path := filepath.Join("keys", fmt.Sprintf("%v.key", user.Email))
	privKey, err := getOrGeneratePrivateKey(path, key.Key)
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

// Cannot test ForgotPassword and ResetPassword handlers as they require
// access to external resource (sending emails) that we do not control
