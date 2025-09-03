package database

// Handles operations on both the transactions table
// and request_funds table

func CreateTransaction(
	sender, receiver string,
	amount float64,
	created_at, signature string,
) error {
	query := "INSERT INTO transactions(sender, receiver, amount, created_at, signature) VALUES(?, ?, ?, ?, ?)"
	_, err := db.Exec(
		query,
		sender,
		receiver,
		amount,
		created_at,
		signature,
	)
	return err
}

func CreateRequestFunds(
	sender, receiver string,
	amount float64,
	created_at, signature string,
) error {
	query := "INSERT INTO request_funds(sender, receiver, amount, created_at, signature) VALUES(?, ?, ?, ?, ?)"
	_, err := db.Exec(
		query,
		sender,
		receiver,
		amount,
		created_at,
		signature,
	)
	return err
}
