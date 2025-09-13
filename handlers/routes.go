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
			r.Post("/new-wallet", CreateWallet)
			r.Get("/wallets", GetAllWallets)
			r.Get("/wallets/{wallet_address}", GetWalletDetails)
			r.Post("/wallets/{wallet_address}/freeze", FreezeWallet)
			r.Post("/wallets/{wallet_address}/activate", ActivateWallet)
			r.Post("/wallets/{wallet_address}/limit", SetOrUpdateLimit)
			r.Post("/transfer-funds", TransferFunds)
			r.Post("/request-funds", RequestFunds)
			r.Get("/recent-transactions/{wallet_address}", GetRecentTransactions)
			r.Get("/transactions/{transaction_id}", GetTransaction)

			// TODO: Implement require ownership middleware that checks if
			// the wallet a user is requesting action on belongs to them.
			// Affected routes */{wallet_address}/*

			r.Get("/verify-login", VerifyLogin)
		})
	})
	return r
}
