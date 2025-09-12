package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
)

// Gets total amount spent on a credit card for the past week, month and year.
func getTotalAmountSpent(cardNo string, period string) (float64, error) {
	query := "CALL getTotalAmountSpent(?)"
	rows, err := db.Query(query, cardNo)
	if err != nil {
		return 0, err
	}

	var (
		dbPeriod string
		amount   float64
	)

	for rows.Next() {
		err = rows.Scan(
			&dbPeriod,
			&amount,
		)
		if err != nil {
			return 0, err
		}

		if dbPeriod == period {
			return amount, nil
		}
	}
	return 0, fmt.Errorf("period %v not found", period)
}

func IsWithinSpendingLimits(cardNo string, newAmount float64) bool {
	var (
		period      string
		limitAmount float64
	)

	query := "SELECT period, amount FROM limits WHERE card_no= ?"
	row := db.QueryRow(query, cardNo)
	err := row.Scan(
		&period,
		&limitAmount,
	)
	if err != nil {
		noLimitsSet := errors.Is(err, sql.ErrNoRows)
		if noLimitsSet {
			return true
		}

		log.Printf("Error checking spending limits; %v\n", err)
		return false
	}

	amountSpent, err := getTotalAmountSpent(cardNo, period)
	if err != nil {
		log.Printf("Error fetching total amount spent; %v\n", err)
		return false
	}

	withinLimits := (amountSpent + newAmount) <= limitAmount
	return withinLimits
}

func SetOrUpdateLimit(cardNo string, period string, amount float64) error {
	query := `
		INSERT INTO limits(card_no, period, amount)
		VALUES(?, ?, ?)
		ON DUPLICATE KEY UPDATE
			period = VALUES(period),
			amount = VALUES(amount);
	`
	_, err := db.Exec(query, cardNo, period, amount)
	return err
}
