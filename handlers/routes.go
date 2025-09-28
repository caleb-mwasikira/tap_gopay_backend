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

		r.Get("/all-transaction-fees", GetAllTransactionFees)
		r.Get("/transaction-fees", GetTransactionFees)

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(RequireAdmin)

			r.Post("/transaction-fees", CreateTransactionFees)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(RequireAuthMiddleware)

			r.Get("/verify-login", VerifyLogin)
			r.HandleFunc("/subscribe-notifications", SubscribeNotifications)

			// Wallets
			r.Post("/new-wallet", CreateWallet)
			r.Get("/wallets", GetAllWallets)
			r.Post("/wallets/owned-by-phone", GetWalletsOwnedByPhoneNo)

			r.Group(func(r chi.Router) {
				r.Use(VerifyWalletOwnership)

				r.Get("/wallets/{wallet_address}", GetWallet)
				r.Post("/wallets/{wallet_address}/freeze", FreezeWallet)
				r.Post("/wallets/{wallet_address}/activate", ActivateWallet)
				r.Post("/wallets/{wallet_address}/limit", SetOrUpdateLimit)
				r.Post("/wallets/{wallet_address}/add-owner", AddWalletOwner)
				r.Post("/wallets/{wallet_address}/remove-owner", RemoveWalletOwner)
			})

			// Transactions
			r.Post("/send-money", SendMoney)
			r.Post("/request-funds", RequestFunds)
			r.Get("/recent-transactions/{wallet_address}", GetRecentTransactions)
			r.Get("/transactions/{transaction_code}", GetTransaction)
			r.Post("/transactions/{transaction_code}/sign-transaction", SignTransaction)

			// Cash Pools
			r.Post("/new-cash-pool", CreateCashPool)
			r.Get("/cash-pools/{wallet_address}", GetCashPool)

			go RefundExpiredCashPools()
		})
	})
	return r
}
