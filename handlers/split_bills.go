package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

const (
	MIN_SPLIT_BILL_AMOUNT float64 = 100.0
)

type Contribution struct {
	Account string  `json:"participant" validate:"account"`
	Amount  float64 `json:"amount" validate:"min=1"`
}

type SplitBillNotification struct {
	BillName             string               `json:"bill_name"`
	Creator              database.WalletOwner `json:"wallet_owner"`
	WalletAddress        string               `json:"wallet_address"`
	Receiver             database.WalletOwner `json:"receiver"`
	BillAmount           float64              `json:"bill_amount"`
	ExpectedContribution Contribution         `json:"expected_contribution"`
	CollectedAmount      float64              `json:"collected_amount"`
	CreatedAt            string               `json:"created_at"`
}

type SplitBillRequest struct {
	BillName      string         `json:"bill_name" validate:"min=4"`
	Description   string         `json:"description"` // Optional
	BillAmount    float64        `json:"bill_amount" validate:"min=100"`
	Contributions []Contribution `json:"contributions" validate:"contributions"`
	Receiver      string         `json:"receiver" validate:"account"`
}

func CreateSplitBill(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	var req SplitBillRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err = validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	// Verify that sum of contributions add upto bill amount
	var totalContributions float64
	for _, c := range req.Contributions {
		totalContributions += c.Amount
	}

	if totalContributions < req.BillAmount {
		message := fmt.Sprintf("Sum of contributions KSH %f MUST match bill amount KSH %f", totalContributions, req.BillAmount)
		api.BadRequest(w, message, nil)
		return
	}

	// Internally, a split bill is just a cash pool
	// that lasts for max 1 hour
	expiresAt := time.Now().Add(1 * time.Hour)

	cashPool, err := database.CreateCashPool(
		user.Id,
		req.BillName,
		database.SplitBill,
		req.Description,
		req.BillAmount,
		req.Receiver,
		expiresAt.Format(time.RFC3339),
	)
	if err != nil {
		api.Errorf(w, "Error splitting bill", err)
		return
	}

	// When a bill is split, we send notifications to all
	// its contributors
	splitBillNotification := SplitBillNotification{
		BillName:        req.BillName,
		Creator:         cashPool.Creator,
		WalletAddress:   cashPool.WalletAddress,
		Receiver:        cashPool.Receiver,
		BillAmount:      cashPool.TargetAmount,
		CollectedAmount: cashPool.CollectedAmount,
		CreatedAt:       cashPool.CreatedAt.Format(time.RFC3339),
	}

	for _, c := range req.Contributions {
		splitBillNotification.ExpectedContribution = c
		go sendNotification(splitBillNotification, c.Account)
	}

	api.OK2(w, cashPool)
}
