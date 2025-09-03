package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	h "github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

var (
	testEmail    string = "calebmwasikira@gmail.com"
	testPassword string = "password420"
	testPhoneNo  string = gofakeit.Phone()
)

func TestRegister(t *testing.T) {
	testServer := httptest.NewServer(r)
	defer testServer.Close()

	req := h.RegisterRequest{
		Username: "fake_" + gofakeit.Name(),
		Email:    gofakeit.Email(),
		Password: "password1234",
		PhoneNo:  testPhoneNo,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("Error marshalling register request; %v\n", err)
	}

	resp, err := http.Post(testServer.URL+"/auth/register", jsonContentType, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Error making register request; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)
}

// Prints the response to stdout.
// And since we can't read the response body more than once,
// we return the body
func printResponse(resp *http.Response) []byte {
	fmt.Println()
	fmt.Printf("%v\n", resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body; %v\n", err)
	}
	fmt.Printf("Body:\n%s\n", string(body))

	return body
}

// Checks the response for expected status code.
// Fails if response status code does NOT match expected status code.
// Returns response body for further processing
func expectStatus(t *testing.T, resp *http.Response, expectedStatusCode int) []byte {
	body := printResponse(resp)

	if resp.StatusCode != expectedStatusCode {
		t.Fatalf("Expected statusCode %v but got %v\n", expectedStatusCode, resp.StatusCode)
	}
	return body
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
	req := h.LoginRequest{
		Email:    testEmail,
		Password: testPassword,
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
		if cookie.Name == h.LOGIN_COOKIE {
			loginCookie = cookie
			break
		}
	}
	if loginCookie == nil {
		t.Fatalf("Expected %v in response but cookie NOT found\n", h.LOGIN_COOKIE)
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

// Cannot test ForgotPassword and ResetPassword handlers as they require
// access to external resource (sending emails) that we do not control
