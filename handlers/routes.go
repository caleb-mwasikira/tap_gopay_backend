package handlers

import (
	"net/http"
	"sync"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// For dev, allow all origins; in prod, validate origin
			return true
		},
	}
	clients = map[string]*websocket.Conn{} // Map of wallet address to *websocket.Conn
	mutex   = &sync.RWMutex{}              // Protect websocket clients
)

// Notifies subscribed users of received transactions
func wsNotifyReceivedFunds(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w)
		return
	}

	// Get all wallets tied to this user
	wallets, err := database.GetAllWallets(user.Id)
	if err != nil {
		api.Errorf(w, "Error fetching user's wallets", err)
		return
	}

	// Upgrade HTTP connection to a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		api.Errorf(w, "Error setting up websocket connection", err)
		return
	}

	mutex.Lock()
	for _, wallet := range wallets {
		clients[wallet.Address] = conn
	}
	mutex.Unlock()
}

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

			r.HandleFunc("/ws-notifications", wsNotifyReceivedFunds)

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
