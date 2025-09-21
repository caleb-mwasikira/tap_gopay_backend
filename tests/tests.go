package tests

import (
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/go-chi/chi/v5"
)

var (
	r *chi.Mux

	jsonContentType string = "application/json"
)

const (
	COLOR_RED   string = "\033[31m"
	COLOR_GREEN string = "\033[32m"
	COLOR_RESET string = "\033[0m"
)

func init() {
	// Initialize cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Error initializing cookie jar; %v\n", err)
	}
	http.DefaultClient.Jar = jar

	r = handlers.GetRoutes()
}

func printResponse(resp *http.Response, expectedStatusCode int) []byte {
	if resp == nil {
		fmt.Println(COLOR_RED, "nil response", COLOR_RESET)
		return nil
	}

	colorCode := COLOR_RED
	if resp.StatusCode == expectedStatusCode {
		colorCode = COLOR_GREEN
	}

	fmt.Println(colorCode)
	fmt.Printf("%v %v %v\n", resp.Request.Method, resp.Request.URL, resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body; %v\n", err)
	}
	fmt.Printf("Body:\n%s\n", string(body))
	fmt.Println(COLOR_RESET)

	return body
}

// Checks the response for expected status code.
// Fails if response status code does NOT match expected status code.
func expectStatus(t *testing.T, resp *http.Response, expectedStatusCode int) []byte {
	body := printResponse(resp, expectedStatusCode)

	// Check status code
	if resp.StatusCode != expectedStatusCode {
		t.Errorf("Expected statusCode %v but got %v\n", expectedStatusCode, resp.StatusCode)
	}

	return body
}

func randomString(length uint) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	var sb strings.Builder
	sb.Grow(int(length))

	for i := 0; i < int(length); i++ {
		num := rand.IntN(len(charset))
		sb.WriteByte(charset[num])
	}
	return sb.String()
}
