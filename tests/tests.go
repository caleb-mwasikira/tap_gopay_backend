package tests

import (
	"log"
	"net/http"
	"net/http/cookiejar"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/go-chi/chi/v5"
)

var (
	r *chi.Mux

	jsonContentType string = "application/json"
)

func init() {
	// Initialize cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("Error initializing cookie jar; %v\n", err)
	}
	http.DefaultClient.Jar = jar

	r = api.GetRoutes()
}
