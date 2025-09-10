package database

type CreditCard struct {
	UserId         int     `json:"user_id"`
	Username       string  `json:"username"`
	PhoneNo        string  `json:"phone_no"`
	CardNo         string  `json:"card_no"`
	InitialDeposit float64 `json:"-"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	Balance        float64 `json:"balance"`
}

func CreateCreditCard(userId int, cardNo string, amount float64) (*CreditCard, error) {
	query := "INSERT INTO credit_cards(user_id, card_no, initial_deposit) VALUES(?, ?, ?)"
	_, err := db.Exec(
		query,
		userId,
		cardNo,
		amount,
	)
	if err != nil {
		return nil, err
	}
	return GetCreditCardDetails(userId, cardNo)
}

func GetCreditCardDetails(userId int, cardNo string) (*CreditCard, error) {
	cc := CreditCard{
		UserId: userId,
		CardNo: cardNo,
	}

	query := `
		SELECT username, phone_no, is_active, created_at, balance
		FROM credit_card_details
		WHERE user_id= ? AND card_no= ?
	`
	row := db.QueryRow(query, userId, cardNo)
	err := row.Scan(
		&cc.Username,
		&cc.PhoneNo,
		&cc.IsActive,
		&cc.CreatedAt,
		&cc.Balance,
	)
	return &cc, err
}

func GetAllCreditCardsOwnedBy(phone string, filter func(*CreditCard) bool) ([]*CreditCard, error) {
	query := `
		SELECT user_id, username, card_no, is_active, created_at, balance
		FROM credit_card_details
		WHERE phone_no= ?
	`
	rows, err := db.Query(query, phone)
	if err != nil {
		return nil, err
	}

	creditCards := []*CreditCard{}

	for rows.Next() {
		cc := CreditCard{
			PhoneNo: phone,
		}
		err := rows.Scan(
			&cc.UserId,
			&cc.Username,
			&cc.CardNo,
			&cc.IsActive,
			&cc.CreatedAt,
			&cc.Balance,
		)
		if err != nil {
			return nil, err
		}

		if filter == nil {
			creditCards = append(creditCards, &cc)
			continue
		}

		if filter(&cc) {
			creditCards = append(creditCards, &cc)
		}
	}

	return creditCards, err
}

func GetAllCreditCards(userId int) ([]*CreditCard, error) {
	query := `
	SELECT username, phone_no, card_no, initial_deposit, is_active, created_at, balance
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
			&cc.PhoneNo,
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
