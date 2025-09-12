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
			r.Post("/new-credit-card", NewCreditCard)
			r.Get("/credit-cards", GetAllCreditCards)
			r.Get("/credit-cards/{card_no}", GetCreditCardDetails)
			r.Post("/credit-cards/{card_no}/freeze", FreezeCreditCard)
			r.Post("/credit-cards/{card_no}/activate", ActivateCreditCard)
			r.Post("/credit-cards/{card_no}/limit", SetOrUpdateLimit)
			r.Post("/transfer-funds", TransferFunds)
			r.Post("/request-funds", RequestFunds)
			r.Get("/recent-transactions/{card_no}", GetRecentTransactions)
			r.Get("/transactions/{transaction_id}", GetTransaction)

			// TODO: Implement require ownership middleware that checks if
			// the credit card a user is requesting action on belongs to them.
			// Affected routes */{card_no}/*

			r.Get("/verify-login", VerifyLogin)
		})
	})
	return r
}
