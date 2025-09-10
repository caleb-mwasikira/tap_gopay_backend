package main

import (
	"log"
	"net/http"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
)

func main() {
	androidApiKey := api.GenerateAndroidApiKey()
	log.Println("ANDROID_API_KEY: ", androidApiKey)

	address := "0.0.0.0:5000"
	log.Printf("Starting web server on http://%v\n", address)

	r := handlers.GetRoutes()
	err := http.ListenAndServe(address, r)
	if err != nil {
		log.Fatalf("Error starting web server; %v\n", err)
	}
}
