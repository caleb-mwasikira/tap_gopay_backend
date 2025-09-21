package database

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const (
	TRANSACTION_ID_LEN int = 12
)

type Account struct {
	Username string `json:"username"`
	Address  string `json:"wallet_address"`
	Phone    string `json:"phone_no"`
}

type Transaction struct {
	TransactionCode string  `json:"transaction_code"`
	Sender          Account `json:"sender"`
	Receiver        Account `json:"receiver"`
	Amount          float64 `json:"amount"`
	Fee             float64 `json:"fee"`

	// Time when transaction was initiated by client - signed by client
	Timestamp   string `json:"timestamp"`
	Signature   string `json:"signature"`
	PublicKeyId string `json:"public_key_hash"`

	// Time when record was saved to database
	CreatedAt string `json:"created_at"`
}

func generateRandomChar() string {
	// Generate a random integer between 65 and 90 (inclusive)
	randInt := 65 + rand.Intn(90-65+1)
	randomChar := rune(randInt)
	return fmt.Sprintf("%c", randomChar)
}

func generateTransactionCode() string {
	rand.NewSource(time.Now().UnixNano())

	str := strings.Builder{}
	var value string

	for range TRANSACTION_ID_LEN {
		if rand.Float32() > 0.3 {
			value = generateRandomChar()
		} else {
			value = fmt.Sprintf("%d", rand.Intn(9))
		}
		str.WriteString(value)
	}

	return str.String()
}

func CreateTransaction(
	userId int,
	sender, receiver string,
	amount, fee float64,
	timestamp, signature string,
	publicKeyHash string,
) (*Transaction, error) {
	transactionCode := "TX" + generateTransactionCode()

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	query := `
	INSERT INTO transactions(
		transaction_code,
		sender,
		receiver,
		amount,
		fee,
		timestamp
	) VALUES(?, ?, ?, ?, ?, ?)`
	_, err = tx.Exec(
		query,
		transactionCode,
		sender,
		receiver,
		amount,
		fee,
		timestamp,
	)
	if err != nil {
		return nil, err
	}

	// Add signature into signatures table
	query = `
	INSERT INTO signatures(
		transaction_code,
		user_id,
		signature,
		public_key_hash
	) VALUES(?, ?, ?, ?)`
	_, err = tx.Exec(
		query,
		transactionCode,
		userId,
		signature,
		publicKeyHash,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return GetTransaction(transactionCode)
}

func GetTransaction(transactionCode string) (*Transaction, error) {
	var t Transaction
	var sender Account
	var receiver Account

	query := `
		SELECT
			transaction_code,
			senders_username,
			senders_phone,
			senders_wallet_address,
			receivers_username,
			receivers_phone,
			receivers_wallet_address,
			amount,
			fee,
			timestamp,
			signature,
			public_key_hash,
			created_at
		FROM transaction_details
		WHERE transaction_code= ?
	`
	row := db.QueryRow(query, transactionCode)
	err := row.Scan(
		&t.TransactionCode,
		&sender.Username,
		&sender.Phone,
		&sender.Address,
		&receiver.Username,
		&receiver.Phone,
		&receiver.Address,
		&t.Amount,
		&t.Fee,
		&t.Timestamp,
		&t.Signature,
		&t.PublicKeyId,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	t.Sender = sender
	t.Receiver = receiver

	return &t, nil
}

func GetRecentTransactions(sendersAddress string) ([]*Transaction, error) {
	query := `
		SELECT
			transaction_code,
			senders_username,
			senders_phone,
			senders_wallet_address,
			receivers_username,
			receivers_phone,
			receivers_wallet_address,
			amount,
			fee,
			timestamp,
			signature,
			public_key_hash,
			created_at
		FROM transaction_details
		WHERE senders_wallet_address= ?
		LIMIT 50
	`
	rows, err := db.Query(query, sendersAddress)
	if err != nil {
		return nil, err
	}

	transactions := []*Transaction{}

	for rows.Next() {
		var t Transaction

		err := rows.Scan(
			&t.TransactionCode,
			&t.Sender.Username,
			&t.Sender.Phone,
			&t.Sender.Address,
			&t.Receiver.Username,
			&t.Receiver.Phone,
			&t.Receiver.Address,
			&t.Amount,
			&t.Fee,
			&t.Timestamp,
			&t.Signature,
			&t.PublicKeyId,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, &t)
	}
	return transactions, nil
}

type RequestFundsResult struct {
	TransactionCode string  `json:"transaction_code"`
	Sender          string  `json:"sender"`
	Receiver        string  `json:"receiver"`
	Amount          float64 `json:"amount"`

	// Time when transaction was initiated by client
	Timestamp   string `json:"timestamp"`
	Signature   string `json:"signature"`
	PublicKeyId string `json:"public_key_hash"`

	// Time when record was saved to database
	CreatedAt string `json:"created_at"`
}

func CreateRequestFunds(
	sender, receiver string,
	amount float64,
	timestamp, signature string,
	publicKeyHash string,
) (*RequestFundsResult, error) {
	transactionCode := "RX" + generateTransactionCode()

	t := RequestFundsResult{
		TransactionCode: transactionCode,
		Sender:          sender,
		Receiver:        receiver,
		Amount:          amount,
		Timestamp:       timestamp,
		Signature:       signature,
		PublicKeyId:     publicKeyHash,
	}

	query := `
	INSERT INTO request_funds(
		transaction_code,
		sender,
		receiver,
		amount,
		timestamp,
		signature,
		public_key_hash
	) VALUES(?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(
		query,
		t.TransactionCode,
		t.Sender,
		t.Receiver,
		t.Amount,
		t.Timestamp,
		t.Signature,
		t.PublicKeyId,
	)
	return &t, err
}
