package handlers

import (
	"net/http"
	"sync"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
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
		api.Unauthorized(w, "Access to this route requires user login")
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
