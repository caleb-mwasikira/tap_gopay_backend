package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/handlers"
	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
)

type notificationResult struct {
	message  *handlers.SplitBillNotification
	receiver string
	err      error
}

func waitForSplitBillNotification(
	ctx context.Context,
	user User,
	notificationsChan chan<- notificationResult,
) {
	notifications, err := waitForNotifications[handlers.SplitBillNotification](ctx, user)
	if err != nil {
		notificationsChan <- notificationResult{
			receiver: user.Username,
			err:      err,
		}
		return
	}

	log.Printf("%v is waiting for split bill notification\n", user.Username)

	select {
	case <-ctx.Done():
		notificationsChan <- notificationResult{
			receiver: user.Username,
			err:      fmt.Errorf("%v got tired of waiting for split bill notification", user.Username),
		}
		return
	case message, ok := <-notifications:
		if !ok {
			notificationsChan <- notificationResult{
				receiver: user.Username,
				err:      fmt.Errorf("notifications channel closed for %v", user.Username),
			}
			return
		}

		notificationsChan <- notificationResult{
			message:  &message,
			receiver: user.Username,
		}
		return
	}
}

func createSplitBill(
	user User, // Logged in user
	name string,
	description string,
	amount float64,
	contributions []handlers.Contribution,
	receiver string,
) (*http.Response, error) {
	requireLogin(user)

	req := handlers.SplitBillRequest{
		BillName:      name,
		Description:   description,
		BillAmount:    amount,
		Contributions: contributions,
		Receiver:      receiver,
	}
	body, err := json.Marshal(&req)
	if err != nil {
		return nil, err
	}

	return http.Post(
		testServer.URL+"/new-split-bill",
		jsonContentType,
		bytes.NewBuffer(body),
	)
}

func getAlmostEqualContributions(totalAmount float64, contributors ...string) []handlers.Contribution {
	if len(contributors) == 0 {
		return nil
	}

	contributions := []handlers.Contribution{}
	var totalContributions float64

	for _, c := range contributors {
		amount := totalAmount / float64(len(contributors))
		amount = utils.RoundFloat(amount, 2)
		contributions = append(contributions, handlers.Contribution{
			Account: c,
			Amount:  amount,
		})

		totalContributions += amount
	}

	// Fix rounding error by
	// assigning leftover difference to the first contributor
	diff := utils.RoundFloat(totalAmount-totalContributions, 2)
	if math.Abs(diff) > 0.0 {
		contributions[0].Amount = utils.RoundFloat(contributions[0].Amount+diff, 2)
	}

	return contributions
}

func TestCreateSplitBill(t *testing.T) {
	// Test creating a split bill with diff ∑(contributions) and bill amount.
	// Should return error

	chaoMinsRestaurant, err := createWallet(chaoMin)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}

	halfBakedAmount := rand.Float64() * handlers.INITIAL_DEPOSIT
	resp, err := createSplitBill(
		tommy,
		"Broken bills",
		"",
		200,
		[]handlers.Contribution{
			{
				Account: lee.PhoneNo,
				Amount:  utils.RoundFloat(halfBakedAmount, 2),
			},
		},
		chaoMinsRestaurant.WalletAddress,
	)
	if err != nil {
		t.Fatalf("Error splitting bill; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusBadRequest)
}

func TestSplitBillNotifications(t *testing.T) {
	users := []User{lee, bob}
	notificationsChan := make(chan notificationResult, len(users))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}

	for _, user := range users {
		wg.Add(1)
		go func(u User) {
			defer wg.Done()
			waitForSplitBillNotification(ctx, u, notificationsChan)
		}(user)
	}

	go func() {
		wg.Wait()
		close(notificationsChan)
	}()

	// Bob and lee are going to share billAmount equally
	billAmount := handlers.MIN_SPLIT_BILL_AMOUNT + (100 * rand.Float64())
	billAmount = utils.RoundFloat(billAmount, 2)

	chaoMinsRestaurant, err := createWallet(chaoMin)
	if err != nil {
		t.Fatalf("Error creating wallet; %v\n", err)
	}
	contributions := getAlmostEqualContributions(billAmount, bob.PhoneNo, lee.PhoneNo)

	resp, err := createSplitBill(
		tommy,
		"Fine dining",
		"Chapo 3, Beans",
		billAmount,
		contributions,
		chaoMinsRestaurant.WalletAddress,
	)
	if err != nil {
		t.Fatalf("Error splitting bill; %v\n", err)
	}
	defer resp.Body.Close()

	expectStatus(t, resp, http.StatusOK)

	// Collect notifications until timeout
	receivedNotifications := []notificationResult{}

	for notification := range notificationsChan {
		if notification.err != nil {
			t.Errorf("%v notification error; %v %v\n", COLOR_RED, notification.err, COLOR_RESET)
		} else {
			receivedNotifications = append(receivedNotifications, notification)
		}
	}

	// Verify users received split bill notifications
	for _, user := range users {
		notification := utils.FindOne(receivedNotifications, func(notification notificationResult) bool {
			return notification.receiver == user.Username
		})
		if notification == nil {
			t.Errorf("%v User '%v' NEVER received their split bill notification %v\n", COLOR_RED, user.Username, COLOR_RESET)
			continue
		}

		t.Logf("\n%v User %v received notification; %#v %v\n", COLOR_GREEN, user.Username, notification.message, COLOR_RESET)

		// TODO: Have user fulfill split bill request

	}
}
