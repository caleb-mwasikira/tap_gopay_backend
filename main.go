package main

import (
	"log"
	"net/http"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
)

func main() {
	androidApiKey := api.GenerateAndroidApiKey()
	log.Println("ANDROID_API_KEY: ", androidApiKey)

	address := "127.0.0.1:5000"
	log.Printf("Starting web server on http://%v\n", address)

	r := api.GetRoutes()
	err := http.ListenAndServe(address, r)
	if err != nil {
		log.Fatalf("Error starting web server; %v\n", err)
	}
}
