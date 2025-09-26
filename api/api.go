package api

import (
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
	"github.com/go-chi/chi/v5"
)

var (
	ANDROID_API_KEY string
)

func init() {
	utils.LoadDotenv()

	ANDROID_API_KEY = os.Getenv("ANDROID_API_KEY")
}

func GenerateAndroidApiKey() string {
	if strings.TrimSpace(ANDROID_API_KEY) == "" {
		return ""
	}
	b64EncodedData := base64.StdEncoding.EncodeToString([]byte(ANDROID_API_KEY))
	return b64EncodedData
}

func StartServer(address string, routes *chi.Mux) {
	androidApiKey := GenerateAndroidApiKey()
	log.Println("ANDROID_API_KEY: ", androidApiKey)

	log.Printf("Starting web server on http://%v\n", address)

	err := http.ListenAndServe(address, routes)
	if err != nil {
		log.Fatalf("Error starting web server; %v\n", err)
	}
}
