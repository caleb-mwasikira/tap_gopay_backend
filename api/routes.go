package api

import (
	h "github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func GetRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		// // Uncomment in production
		// r.Use(h.RequireAndroidApiKeyMiddleware)
		r.Use(middleware.Logger)

		r.Post("/auth/register", h.RegisterHandler)
		r.Post("/auth/login", h.LoginHandler)
		r.Post("/auth/forgot-password", h.ForgotPasswordHandler)
		r.Post("/auth/reset-password", h.ResetPasswordHandler)

		r.Group(func(r chi.Router) {
			r.Use(h.RequireAuthMiddleware)

			// Protected routes
			r.Get("/credit-cards", h.GetCreditCardsHandler)
			r.Post("/credit-cards", h.NewCreditCardHandler)
			r.Post("/credit-cards/freeze", h.FreezeCreditCardHandler)
			r.Post("/credit-cards/activate", h.ActivateCreditCardHandler)
			r.Post("/send-funds", h.SendFundsHandler)
			r.Post("/request-funds", h.RequestFundsHandler)

			r.Get("/verify-login", h.VerifyLoginHandler)
		})
	})
	return r
}
