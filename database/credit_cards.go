package database

import "time"

type CreditCard struct {
	UserId         int     `json:"user_id"`
	Username       string  `json:"username"`
	CardNo         string  `json:"card_no"`
	InitialDeposit float64 `json:"-"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	Balance        float64 `json:"balance"`
}

func CreateCreditCard(userId int, cardNo string, amount float64) (*CreditCard, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	query := "INSERT INTO credit_cards(user_id, card_no, initial_deposit, created_at) VALUES(?, ?, ?, ?)"
	_, err := db.Exec(
		query,
		userId,
		cardNo,
		amount,
		now,
	)
	if err != nil {
		return nil, err
	}
	return GetCreditCard(userId, cardNo)
}

func GetCreditCard(userId int, cardNo string) (*CreditCard, error) {
	query := `
	SELECT username, initial_deposit, is_active, created_at, balance
	FROM credit_card_details
	WHERE user_id= ? AND card_no= ?`
	row := db.QueryRow(query, userId, cardNo)

	cc := CreditCard{
		UserId: userId,
		CardNo: cardNo,
	}
	err := row.Scan(
		&cc.Username,
		&cc.InitialDeposit,
		&cc.IsActive,
		&cc.CreatedAt,
		&cc.Balance,
	)
	return &cc, err
}

func GetAllCreditCards(userId int) ([]*CreditCard, error) {
	query := `
	SELECT username, card_no, initial_deposit, is_active, created_at, balance
	FROM credit_card_details
	WHERE user_id= ?
	`
	rows, err := db.Query(query, userId)
	if err != nil {
		return nil, err
	}

	creditCards := []*CreditCard{}

	for rows.Next() {
		cc := CreditCard{
			UserId: userId,
		}
		err = rows.Scan(
			&cc.Username,
			&cc.CardNo,
			&cc.InitialDeposit,
			&cc.IsActive,
			&cc.CreatedAt,
			&cc.Balance,
		)
		if err != nil {
			return nil, err
		}
		creditCards = append(creditCards, &cc)
	}
	return creditCards, nil
}

func FreezeCreditCard(userId int, cardNo string) error {
	query := "UPDATE credit_cards SET is_active= 0 WHERE user_id= ? AND card_no= ?"
	_, err := db.Exec(query, userId, cardNo)
	return err
}

func ActivateCreditCard(userId int, cardNo string) error {
	query := "UPDATE credit_cards SET is_active= 1 WHERE user_id= ? AND card_no= ?"
	_, err := db.Exec(query, userId, cardNo)
	return err
}
