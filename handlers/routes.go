package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func GetRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		// // Uncomment in production
		// r.Use(RequireAndroidApiKeyMiddleware)
		r.Use(middleware.Logger)

		r.Post("/auth/register", Register)
		r.Post("/auth/login", Login)
		r.Post("/auth/forgot-password", ForgotPassword)
		r.Post("/auth/reset-password", ResetPassword)

		r.Group(func(r chi.Router) {
			r.Use(RequireAuthMiddleware)

			// Protected routes
			r.Get("/credit-cards", GetCreditCards)
			r.Post("/credit-cards", NewCreditCard)
			r.Post("/credit-cards/freeze", FreezeCreditCard)
			r.Post("/credit-cards/activate", ActivateCreditCard)
			r.Post("/send-funds", SendFunds)
			r.Post("/request-funds", RequestFunds)

			r.Get("/verify-login", VerifyLogin)
		})
	})
	return r
}
