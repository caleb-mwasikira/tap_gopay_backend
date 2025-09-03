package database

import "time"

type CreditCard struct {
	UserId         int     `json:"user_id"`
	CardNo         string  `json:"card_no"`
	InitialDeposit float64 `json:"initial_deposit"`
	IsActive       bool    `json:"is_active"`
	PublicKey      []byte  `json:"public_key"`
	CreatedAt      string  `json:"created_at"`
}

func CreateCreditCard(userId int, cardNo string, amount float64, pubKeyBytes []byte) (*CreditCard, error) {
	cc := CreditCard{
		UserId:         userId,
		CardNo:         cardNo,
		InitialDeposit: amount,
		IsActive:       true,
		PublicKey:      pubKeyBytes,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339Nano),
	}

	query := "INSERT INTO credit_cards(user_id, card_no, initial_deposit, public_key, created_at) VALUES(?, ?, ?, ?, ?)"
	_, err := db.Exec(
		query,
		cc.UserId,
		cc.CardNo,
		cc.InitialDeposit,
		cc.PublicKey,
		cc.CreatedAt,
	)
	return &cc, err
}

func GetCreditCard(userId int, cardNo string, isActive bool) (*CreditCard, error) {
	query := "SELECT initial_deposit, is_active, public_key FROM credit_cards WHERE user_id= ? AND card_no= ? AND is_active= ?"
	row := db.QueryRow(query, userId, cardNo, isActive)

	cc := CreditCard{
		UserId: userId,
		CardNo: cardNo,
	}
	err := row.Scan(
		&cc.InitialDeposit,
		&cc.IsActive,
		&cc.PublicKey,
	)
	return &cc, err
}

func GetAllCreditCards(userId int) ([]*CreditCard, error) {
	query := "SELECT card_no, initial_deposit, public_key, is_active, created_at FROM credit_cards WHERE user_id= ?"
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
			&cc.CardNo,
			&cc.InitialDeposit,
			&cc.PublicKey,
			&cc.IsActive,
			&cc.CreatedAt,
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
