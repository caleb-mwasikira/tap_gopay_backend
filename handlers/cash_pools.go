package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

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

func RefundExpiredCashPools() {
	for {
		// TODO: Set longer time delay in production eg. 15 * time.Minute
		<-time.After(5 * time.Second)

		expiredCashPools, err := database.GetExpiredCashPools()
		if err != nil {
			log.Printf("Error fetching expired cash pools; %v\n", err)
			continue
		}

		if len(expiredCashPools) == 0 {
			continue
		}

		for _, cashPool := range expiredCashPools {
			go func(pool string) {
				failedRefunds, err := database.RefundExpiredCashPool(pool)
				if err != nil {
					log.Printf("Error refunding cash pool; %v\n", err)
				}

				log.Printf("%v failed refunds\n", len(failedRefunds))
				log.Println(failedRefunds)
			}(cashPool)
		}
	}
}

func RemoveCashPool(w http.ResponseWriter, r *http.Request) {
	walletAddress := chi.URLParam(r, "wallet_address")

	err := validateWalletAddress(walletAddress)
	if err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	err = database.RemoveCashPool(walletAddress)
	if err != nil {
		api.Errorf(w, "Error removing cash pool", err)
		return
	}

	api.OK(w, "Cash pool expired successfully. Waiting to refund deposits before deletion")
}
