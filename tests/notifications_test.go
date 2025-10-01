package tests

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/gorilla/websocket"
)

func waitForNotifications[T any](
	ctx context.Context,
	user User,
	serverUrl string,
) (<-chan T, error) {
	accessToken := requireLogin(user)

	log.Printf("%v waiting for transaction notifications\n", user.Username)

	u, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}

	// Establish WebSocket connection
	rawUrl := "ws://" + u.Host + "/subscribe-notifications"
	header := http.Header{}
	header.Add("AuthToken", fmt.Sprintf("Bearer %v", accessToken))

	conn, resp, err := websocket.DefaultDialer.Dial(rawUrl, header)
	if err != nil {
		printResponse(resp, http.StatusOK)
		return nil, err
	}
	defer conn.Close()

	notifications := make(chan T, 10)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("notifications channel closed by caller")
				close(notifications)
				return
			default:
				var message T

				err = conn.ReadJSON(&message)
				if err != nil {
					log.Printf("error reading notification message; %v\n", err)
					close(notifications)
					return
				}

				notifications <- message
			}
		}
	}()

	return notifications, nil
}

func TestNotifications(t *testing.T) {
	// Get one of tommy's active wallets
	tommysWallet, err := createWallet(tommy)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Get one of lee's active wallets
	leesWallet, err := createWallet(lee)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	// Lee waits for received transactions
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	notifications, err := waitForNotifications[database.Transaction](ctx, lee, testServer.URL)
	if err != nil {
		log.Fatalf("Error establishing notifications channel; %v\n", err)
	}

	// Tommy sends money to lee
	resp, err := sendMoney(
		tommysWallet.WalletAddress,
		leesWallet.WalletAddress,
		tommy,
		1,
	)
	if err != nil {
		t.Fatalf("Error transferring funds; %v\n", err)
	}

	expectStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// Lee should receive notification of transaction
	select {
	case <-time.After(10 * time.Second):
		cancel()
		t.Errorf("Tired of waiting for transaction notification")

	case <-notifications:
		log.Println("Received transaction notification from server")
	}
}
