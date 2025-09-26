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

func waitForNotifications(
	ctx context.Context,
	user User,
	serverUrl string,
	notifications chan<- database.Transaction,
) error {
	accessToken := requireLogin(user)

	log.Printf("%v waiting for transaction notifications\n", user.Username)

	u, err := url.Parse(serverUrl)
	if err != nil {
		return err
	}

	// Establish WebSocket connection
	rawUrl := "ws://" + u.Host + "/subscribe-notifications"
	header := http.Header{}
	header.Add("AuthToken", fmt.Sprintf("Bearer %v", accessToken))

	conn, resp, err := websocket.DefaultDialer.Dial(rawUrl, header)
	if err != nil {
		printResponse(resp, http.StatusOK)
		return err
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var transaction database.Transaction

			err = conn.ReadJSON(&transaction)
			if err != nil {
				return fmt.Errorf("Expected message to be of type Transaction but found garbage data")
			}

			notifications <- transaction
		}
	}
}

func TestSubscribeNotifications(t *testing.T) {
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
	notifications := make(chan database.Transaction)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err = waitForNotifications(ctx, lee, testServer.URL, notifications)
		if err != nil {
			log.Fatalf("Error waiting for transaction notifications; %v\n", err)
		}
	}()

	// Sleep for few seconds to give time for lee to start
	// their notification listener
	time.Sleep(2 * time.Second)

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
