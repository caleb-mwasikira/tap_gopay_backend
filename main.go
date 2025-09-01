package main

import (
	"log"
	"net/http"

	h "github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		// Protect all routes from access by other devices
		// except our Android app
		r.Use(h.RequireAndroidApiKeyMiddleware)
		r.Use(middleware.Logger)

		r.Post("/auth/register", h.RegisterHandler)
		r.Post("/auth/login", h.LoginHandler)
		r.Post("/auth/forgot-password", h.ForgotPasswordHandler)
		r.Post("/auth/reset-password", h.ResetPasswordHandler)

		r.Group(func(r chi.Router) {
			r.Use(h.RequireAuthMiddleware)

			// Protected routes
		})
	})

	androidApiKey := h.GenerateAndroidApiKey()
	log.Println("ANDROID_API_KEY: ", androidApiKey)

	address := "127.0.0.1:5000"
	log.Printf("Starting web server on http://%v\n", address)
	err := http.ListenAndServe(address, r)
	if err != nil {
		log.Fatalf("Error starting web server; %v\n", err)
	}
}
