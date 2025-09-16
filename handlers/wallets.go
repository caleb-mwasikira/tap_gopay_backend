package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/go-chi/chi/v5"
)

const (
	MIN_WALLET_ADDR_LEN int     = 16
	INITIAL_DEPOSIT     float64 = 100
)

// Wallet address will be in the format
//
//	1234 5678 8765 5432
func generateWalletAddress() string {
	str := strings.Builder{}
	index := 0

	for range MIN_WALLET_ADDR_LEN {
		if index != 0 && (index%4) == 0 {
			str.WriteString(" ")
		}

		num := rand.IntN(10)
		str.WriteString(fmt.Sprintf("%d", num))
		index++
	}
	walletAddress := str.String()
	return strings.TrimSpace(walletAddress)
}

func CreateWallet(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	walletAddress := generateWalletAddress()
	wallet, err := database.CreateWallet(
		user.Id, walletAddress, INITIAL_DEPOSIT,
	)
	if err != nil {
		api.Errorf(w, "Error creating wallet", err)
		return
	}

	api.OK2(w, wallet)
}

// Fetch all wallets associated with currently logged in user
func GetAllWallets(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	wallets, err := database.GetAllWallets(user.Id)
	if err != nil {
		api.Errorf(w, "Error fetching users wallets", err)
		return
	}

	api.OK2(w, wallets)
}

func GetWalletDetails(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	walletAddress := chi.URLParam(r, "wallet_address")
	if err := validateWalletAddress(walletAddress); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	wallet, err := database.GetWalletDetails(user.Id, walletAddress)
	if err != nil {
		api.Errorf(w, "Error fetching wallet details", err)
		return
	}

	api.OK2(w, wallet)
}

func FreezeWallet(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	walletAddress := chi.URLParam(r, "wallet_address")
	if err := validateWalletAddress(walletAddress); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	err := database.FreezeWallet(user.Id, walletAddress)
	if err != nil {
		api.Errorf(w, "Error freezing wallet account", err)
		return
	}

	api.OK(w, fmt.Sprintf("Wallet '%v' deactivated successfully", walletAddress))
}

func ActivateWallet(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	walletAddress := chi.URLParam(r, "wallet_address")
	if err := validateWalletAddress(walletAddress); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	err := database.ActivateWallet(user.Id, walletAddress)
	if err != nil {
		api.Errorf(w, "Error activating wallet account", err)
		return
	}

	api.OK(w, fmt.Sprintf("Wallet '%v' activated successfully", walletAddress))
}

type SetupLimitRequest struct {
	Period string  `json:"period" validate:"period"`
	Amount float64 `json:"amount" validate:"amount"`
}

func SetOrUpdateLimit(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	walletAddress := chi.URLParam(r, "wallet_address")
	if err := validateWalletAddress(walletAddress); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	var req SetupLimitRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	err = database.SetOrUpdateLimit(user.Id, walletAddress, req.Period, req.Amount)
	if err != nil {
		api.Errorf(w, "Error setting spending limits", err)
		return
	}

	api.OK(w, "Successfully setup new spending limit")
}
