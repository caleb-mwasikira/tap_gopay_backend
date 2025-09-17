package database

import (
	"time"
)

type TransactionFee struct {
	MinAmount float64 `json:"min_amount"`
	MaxAmount float64 `json:"max_amount"`
	Fee       float64 `json:"fee"`
}

func CreateTransactionFees(
	minAmount, maxAmount, fee float64,
	effectiveFrom time.Time,
	effectiveTo *time.Time, // Nullable
) error {
	query := `
		INSERT IGNORE INTO transaction_fees(
			min_amount,
			max_amount,
			fee,
			effective_from,
			effective_to
		)
		VALUES(?, ?, ?, ?, ?)
	`
	_, err := db.Exec(
		query,
		minAmount,
		maxAmount,
		fee,
		effectiveFrom,
		effectiveTo,
	)
	if err != nil {
		return err
	}

	return nil
}

func GetAllTransactionFees() ([]TransactionFee, error) {
	query := `
		SELECT min_amount, max_amount, fee
		FROM transaction_fees
		WHERE effective_from <= NOW()
		AND (effective_to > NOW() OR effective_to IS NULL)
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	var newFees []TransactionFee

	for rows.Next() {
		var fee TransactionFee
		err = rows.Scan(
			&fee.MinAmount,
			&fee.MaxAmount,
			&fee.Fee,
		)
		if err != nil {
			return nil, err
		}

		newFees = append(newFees, fee)
	}

	return newFees, nil
}

// Fetches transaction fees by amount from database.
// Error returned might be [sql.ErrNoRows]
func GetTransactionFees(amount float64) (*TransactionFee, error) {
	var t TransactionFee

	query := `
		SELECT min_amount, max_amount, fee
		FROM transaction_fees
		WHERE ? BETWEEN min_amount AND max_amount
		AND NOW() BETWEEN effective_from AND COALESCE(effective_to, NOW())
		LIMIT 1
	`
	err := db.QueryRow(query, amount).Scan(
		&t.MinAmount,
		&t.MaxAmount,
		&t.Fee,
	)
	return &t, err
}
