package handlers

import (
	"log"
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
	subscribed = map[int]*websocket.Conn{} // Map of user id to *websocket.Conn
	mutex      = &sync.RWMutex{}           // Protect websocket subscribed
)

// Notifies subscribed users of received transactions
func SubscribeNotifications(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	// Upgrade HTTP connection to a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		api.Errorf(w, "Error setting up websocket connection", err)
		return
	}

	mutex.Lock()
	subscribed[user.Id] = conn
	mutex.Unlock()
}

// Sends notifications to receivers if they are subscribed to
// receiving notification messages.
// Receivers must either be valid emails, phone numbers, wallet addresses
// or a combination of both
func sendNotification[T any](message T, receivers ...string) {
	userIds, err := database.GetUserIds(receivers...)
	if err != nil {
		log.Printf("Error fetching receivers user ids; %v\n", err)
		return
	}

	conns := []*websocket.Conn{}

	mutex.RLock()
	for _, userId := range userIds {
		conn, ok := subscribed[userId]
		if ok {
			conns = append(conns, conn)
		}
	}
	mutex.RUnlock()

	for _, conn := range conns {
		err := conn.WriteJSON(&message)
		if err != nil {
			log.Printf("Error notifying wallet owners of transaction; %v\n", err)
		}
	}
}
