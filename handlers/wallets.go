package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/go-chi/chi/v5"
)

const (
	INITIAL_DEPOSIT float64 = 100
)

type CreateWalletRequest struct {
	WalletName string `json:"wallet_name" validate:"min=4"`

	TotalOwners uint `json:"total_owners" validate:"min=1,max=10"`

	// Number of signatures required for wallet to complete transaction
	NumSignatures uint `json:"num_signatures" validate:"min=1,max=10"`
}

func CreateWallet(w http.ResponseWriter, r *http.Request) {
	user, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	var req CreateWalletRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err = validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	if req.NumSignatures > req.TotalOwners {
		api.BadRequest(w, "Number of signatures required cannot exceed total number of owners", nil)
		return
	}

	wallet, err := database.CreateWallet(
		user.Id,
		req.WalletName,
		INITIAL_DEPOSIT,
		req.TotalOwners,
		req.NumSignatures,
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

func GetWallet(w http.ResponseWriter, r *http.Request) {
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

	wallet, err := database.GetWallet(user.Id, walletAddress)
	if err != nil {
		api.Errorf(w, "Error fetching wallet details", err)
		return
	}

	api.OK2(w, wallet)
}

type PhoneNoRequest struct {
	PhoneNo string `json:"phone_no" validate:"phone_no"`
}

func GetWalletsOwnedByPhoneNo(w http.ResponseWriter, r *http.Request) {
	var req PhoneNoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err = validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	wallets, err := database.GetWalletsOwnedByPhoneNo(
		req.PhoneNo,
		func(wallet *database.Wallet) bool {
			return wallet.IsActive
		},
	)
	if err != nil || len(wallets) == 0 {
		message := fmt.Sprintf("Error fetching wallets owned by '%v'", req.PhoneNo)
		api.Errorf(w, message, nil)
		return
	}

	api.OK2(w, wallets)
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

// A user can add a wallet owner by providing the
// counterparts email or phone number
type WalletOwnerRequest struct {
	Email   string `json:"email"`
	PhoneNo string `json:"phone_no"`
}

func AddWalletOwner(w http.ResponseWriter, r *http.Request) {
	loggedInUser, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	var req WalletOwnerRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	walletAddress := chi.URLParam(r, "wallet_address")

	err = validateWalletAddress(walletAddress)
	if err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	isValidEmail := validateEmail(req.Email) == nil
	isValidPhoneNo := validatePhoneNumber(req.PhoneNo) == nil

	if !isValidEmail && !isValidPhoneNo {
		api.BadRequest(w, "Please enter a valid email or phone number", nil)
		return
	}

	// Fetch id of user to add
	newUser, err := database.GetUserByEmailOrPhoneNo(req.Email, req.PhoneNo)
	if err != nil {
		message := fmt.Sprintf("User '%v' or '%v' not found", req.Email, req.PhoneNo)
		api.NotFound(w, message)
		return
	}

	err = database.AddWalletOwner(loggedInUser.Id, newUser.Id, walletAddress)
	if err != nil {
		api.Errorf(w, "Error adding wallet owner", err)
		return
	}

	message := fmt.Sprintf("User '%v' or '%v' added as owner of wallet '%v'",
		req.Email, req.PhoneNo, walletAddress)
	api.OK(w, message)
}

func RemoveWalletOwner(w http.ResponseWriter, r *http.Request) {
	loggedInUser, ok := getAuthUser(r)
	if !ok {
		api.Unauthorized(w, "Access to this route requires user login")
		return
	}

	var req WalletOwnerRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	walletAddress := chi.URLParam(r, "wallet_address")

	err = validateWalletAddress(walletAddress)
	if err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	isValidEmail := validateEmail(req.Email) == nil
	isValidPhoneNo := validatePhoneNumber(req.PhoneNo) == nil

	if !isValidEmail && !isValidPhoneNo {
		api.BadRequest(w, "Please enter a valid email or phone number", nil)
		return
	}

	// Fetch id of user to remove
	userToRemove, err := database.GetUserByEmailOrPhoneNo(req.Email, req.PhoneNo)
	if err != nil {
		message := fmt.Sprintf("User with email '%v' or phone number '%v' not found", req.Email, req.PhoneNo)
		api.NotFound(w, message)
		return
	}

	err = database.RemoveWalletOwner(loggedInUser.Id, userToRemove.Id, walletAddress)
	if err != nil {
		api.Errorf(w, "Error removing wallet owner", err)
		return
	}

	message := fmt.Sprintf("User '%v' or '%v' removed as owner of wallet '%v'",
		req.Email, req.PhoneNo, walletAddress)
	api.OK(w, message)
}
