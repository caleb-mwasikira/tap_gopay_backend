package database

func AddSignature(
	userId int,
	transactionCode string,
	signature string,
	pubKeyHash string,
) error {
	var transactionId int

	query := "SELECT id FROM transactions WHERE transaction_code= ?"
	err := db.QueryRow(query, transactionCode).Scan(
		&transactionId,
	)
	if err != nil {
		return err
	}

	query = `
	INSERT INTO signatures(
		user_id,
		transaction_id,
		transaction_code,
		signature,
		public_key_hash
	) VALUES(?, ?, ?, ?, ?)`

	_, err = db.Exec(
		query,
		userId,
		transactionId,
		transactionCode,
		signature,
		pubKeyHash,
	)
	return err
}
