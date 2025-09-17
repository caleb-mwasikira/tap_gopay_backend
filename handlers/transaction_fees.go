package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

var (
	transactionFeesCache []database.TransactionFee
)

type TransactionFeeRequest struct {
	MinAmount     float64    `json:"min_amount" validate:"min=0"`
	MaxAmount     float64    `json:"max_amount" validate:"min=0"`
	Fee           float64    `json:"fee" validate:"min=0"`
	EffectiveFrom time.Time  `json:"effective_from"`
	EffectiveTo   *time.Time `json:"effective_to,omitempty"`
}

func CreateTransactionFees(w http.ResponseWriter, r *http.Request) {
	var req TransactionFeeRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", nil)
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	var result *database.TransactionFee

	// Check cache
	mutex.RLock()
	for _, t := range transactionFeesCache {
		if t.MinAmount == req.MinAmount && t.MaxAmount == t.MaxAmount && t.Fee == req.Fee {
			result = &t
			break
		}
	}
	mutex.RUnlock()

	if result != nil {
		api.OK(w, "Transaction fees setup correctly")
		return
	}

	err = database.CreateTransactionFees(
		req.MinAmount, req.MaxAmount, req.Fee,
		req.EffectiveFrom, req.EffectiveTo,
	)
	if err != nil {
		api.Errorf(w, "Error setting transaction fees", err)
		return
	}

	// Update cache
	mutex.Lock()
	transactionFeesCache = append(transactionFeesCache, database.TransactionFee{
		MinAmount: req.MinAmount,
		MaxAmount: req.MaxAmount,
		Fee:       req.Fee,
	})
	mutex.Unlock()

	api.OK(w, "Transaction fees setup correctly")
}

func GetAllTransactionFees(w http.ResponseWriter, r *http.Request) {
	mutex.RLock()
	cached := transactionFeesCache
	mutex.RUnlock()

	if len(cached) != 0 {
		api.OK2(w, cached)
		return
	}

	transactionFees, err := database.GetAllTransactionFees()
	if err != nil {
		api.Errorf(w, "Error fetching current transaction fees", err)
		return
	}

	// Update cache
	mutex.Lock()
	transactionFeesCache = transactionFees
	mutex.Unlock()

	api.OK2(w, transactionFees)
}

// Fetches transaction fees by amount from cache or database.
// Error returned might be [sql.ErrNoRows]
func getTransactionFees(amount float64) (*database.TransactionFee, error) {
	var transactionFee *database.TransactionFee

	// Check cache
	mutex.RLock()
	for _, t := range transactionFeesCache {
		withinRange := amount > t.MinAmount && amount <= t.MaxAmount
		if withinRange {
			transactionFee = &t
			break
		}
	}
	mutex.RUnlock()

	if transactionFee != nil {
		return transactionFee, nil
	}

	// Fetch from database
	transactionFee, err := database.GetTransactionFees(amount)
	if err != nil {
		return nil, err
	}

	// Update cache
	mutex.Lock()
	transactionFeesCache = append(transactionFeesCache, *transactionFee)
	mutex.Unlock()

	return transactionFee, nil
}

func GetTransactionFees(w http.ResponseWriter, r *http.Request) {
	str := r.URL.Query().Get("amount")
	amount, err := strconv.ParseFloat(str, 64)
	if err != nil {
		api.BadRequest(w, "Expected query parameter ?amount=<value>", err)
		return
	}

	if amount < MIN_AMOUNT {
		api.BadRequest(w, "Minimum transferrable amount is KSH 1.0", nil)
		return
	}

	transactionFee, err := getTransactionFees(amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No transaction fee found for this amount
			// return zero transaction fee
			var transactionFee database.TransactionFee

			api.OK2(w, transactionFee)
			return
		}

		api.Errorf(w, "Error fetching transaction fees", err)
		return
	}

	api.OK2(w, transactionFee)
}
