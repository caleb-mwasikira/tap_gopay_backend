package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/go-chi/chi/v5"
)

type CashPoolRequest struct {
	Name         string  `json:"name" validate:"min=4"`
	Description  string  `json:"description"`
	TargetAmount float64 `json:"target_amount" validate:"min=100"`
	Receiver     string  `json:"receiver" validate:"wallet_address"`
	ExpiresAt    string  `json:"expires_at" validate:"expiry"`
}

func CreateCashPool(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	var req CashPoolRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err = validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	cashPool, err := database.CreateCashPool(
		user.Id,
		req.Name,
		req.Description,
		req.TargetAmount,
		req.Receiver,
		req.ExpiresAt,
	)
	if err != nil {
		api.Errorf(w, "Error creating cash pool", err)
		return
	}

	api.OK2(w, cashPool)
}

func GetCashPool(w http.ResponseWriter, r *http.Request) {
	walletAddress := chi.URLParam(r, "wallet_address")

	err := validateWalletAddress(walletAddress)
	if err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	cashPool, err := database.GetCashPool(walletAddress)
	if err != nil {
		api.Errorf(w, "Error fetching cash pool", err)
		return
	}

	api.OK2(w, cashPool)
}
