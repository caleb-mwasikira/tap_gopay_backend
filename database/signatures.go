package database

func AddSignature(
	userId int,
	transactionCode string,
	signature string,
	pubKeyHash string,
) error {
	query := `
	INSERT INTO signatures(
		user_id,
		transaction_code,
		signature,
		public_key_hash
	) VALUES(?, ?, ?, ?)`

	_, err := db.Exec(
		query,
		userId,
		transactionCode,
		signature,
		pubKeyHash,
	)
	return err
}
