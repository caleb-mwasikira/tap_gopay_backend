package tests

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

var (
	testUsername string = "tommy"
	testEmail    string = "iamtommy@gmail.com"
	testPassword string = "tommyhasagun"
)

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

func TestRegister(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	username := testUsername
	email := testEmail
	phoneNo := gofakeit.Phone()
	password := testPassword

	argon2Key, err := encrypt.DeriveKey(password, nil)
	if err != nil {
		t.Fatalf("Error generating KDF password; %v\n", err)
	}

	var buff bytes.Buffer
	multipartWriter := multipart.NewWriter(&buff)

	multipartWriter.WriteField("username", username)
	multipartWriter.WriteField("email", email)
	multipartWriter.WriteField("phone_no", phoneNo)

	// Fetch user's public key and PEM encode it
	path := filepath.Join("keys", fmt.Sprintf("%v.key", email))
	privKey, err := getPrivateKey(path, argon2Key.Key)
	if err != nil {
		t.Fatalf("Error fetching user's private key; %v\n", err)
	}
	pubKeyBytes, err := encrypt.PemEncodePublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("Error PEM encoding public key; %v\n", err)
	}

	// Upload public key
	err = uploadFile(multipartWriter, handlers.PUBLIC_KEY, pubKeyBytes)
	if err != nil {
		t.Fatalf("Error uploading PUBLIC_KEY file; %v\n", err)
	}
	multipartWriter.Close()

	// Send request
	resp, err := http.Post(
		testServer.URL+"/auth/register",
		multipartWriter.FormDataContentType(),
		&buff,
	)
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}
	defer resp.Body.Close()

	printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)
}

func TestLogin(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	// Try accessing protected resource without logging in
	resp, err := http.Get(testServer.URL + "/verify-login")
	if err != nil {
		t.Fatalf("Error making request; %v\n", err)
	}

	printResponse(resp)
	resp.Body.Close()

	expectStatus(t, resp.StatusCode, http.StatusUnauthorized)

	// Login
	email := testEmail
	password := testPassword

	// Get user's password and generate longer password using KDF
	argon2Key, err := encrypt.DeriveKey(password, nil)
	if err != nil {
		t.Fatalf("Error generating KDF password; %v\n", err)
	}

	// Fetch or generate user's private key for signing
	path := filepath.Join("keys", fmt.Sprintf("%v.key", email))
	privKey, err := getPrivateKey(path, argon2Key.Key)
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

	printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)

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

	printResponse(resp)
	expectStatus(t, resp.StatusCode, http.StatusOK)
}

// Cannot test ForgotPassword and ResetPassword handlers as they require
// access to external resource (sending emails) that we do not control
