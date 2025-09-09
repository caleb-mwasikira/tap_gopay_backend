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
	CardNo   string `json:"card_no"`
	PhoneNo  string `json:"phone_no"`
}

type Transaction struct {
	TransactionId string  `json:"transaction_id"`
	Sender        Account `json:"sender"`
	Receiver      Account `json:"receiver"`
	Amount        float64 `json:"amount"`
	CreatedAt     string  `json:"created_at"` // RFC3339 formatted string
	Signature     string  `json:"signature"`
}

func generateRandomChar() string {
	// Generate a random integer between 65 and 90 (inclusive)
	randInt := 65 + rand.Intn(90-65+1)
	randomChar := rune(randInt)
	return fmt.Sprintf("%c", randomChar)
}

func generateTransactionId() string {
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
	sender, receiver string,
	amount float64,
	created_at, signature string,
) (*Transaction, error) {
	transactionId := generateTransactionId()
	now := time.Now().Format(time.RFC3339)

	query := `
	INSERT INTO transactions(
		transaction_id, 
		sender, receiver, 
		amount, created_at, 
		signature
	) VALUES(?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(
		query,
		transactionId,
		sender,
		receiver,
		amount,
		now,
		signature,
	)
	if err != nil {
		return nil, err
	}

	return GetTransaction(transactionId)
}

func GetTransaction(transactionId string) (*Transaction, error) {
	var t Transaction
	var sender Account
	var receiver Account

	query := `
		SELECT
			transaction_id,
			senders_username,
			senders_phone,
			senders_card_no,
			receivers_username,
			receivers_phone,
			receivers_card_no,
			amount,
			signature,
			created_at
		FROM transaction_details
		WHERE transaction_id= ?
	`
	row := db.QueryRow(query, transactionId)
	err := row.Scan(
		&t.TransactionId,
		&sender.Username,
		&sender.PhoneNo,
		&sender.CardNo,
		&receiver.Username,
		&receiver.PhoneNo,
		&receiver.CardNo,
		&t.Amount,
		&t.Signature,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	t.Sender = sender
	t.Receiver = receiver

	return &t, nil
}

func GetRecentTransactions(sendersCardNo string) ([]*Transaction, error) {
	query := `
		SELECT
			transaction_id,
			senders_username,
			senders_phone,
			senders_card_no,
			receivers_username,
			receivers_phone,
			receivers_card_no,
			amount,
			signature,
			created_at
		FROM transaction_details
		WHERE senders_card_no= ?
		LIMIT 50
	`
	rows, err := db.Query(query, sendersCardNo)
	if err != nil {
		return nil, err
	}

	transactions := []*Transaction{}

	for rows.Next() {
		var t Transaction

		err := rows.Scan(
			&t.TransactionId,
			&t.Sender.Username,
			&t.Sender.PhoneNo,
			&t.Sender.CardNo,
			&t.Receiver.Username,
			&t.Receiver.PhoneNo,
			&t.Receiver.CardNo,
			&t.Amount,
			&t.Signature,
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
	TransactionId string  `json:"transaction_id"`
	Sender        string  `json:"sender"`
	Receiver      string  `json:"receiver"`
	Amount        float64 `json:"amount"`
	CreatedAt     string  `json:"created_at"` // RFC3339 formatted string
	Signature     string  `json:"signature"`
}

func CreateRequestFunds(
	sender, receiver string,
	amount float64,
	created_at, signature string,
) (*RequestFundsResult, error) {
	t := RequestFundsResult{
		TransactionId: generateTransactionId(),
		Sender:        sender,
		Receiver:      receiver,
		Amount:        amount,
		CreatedAt:     created_at,
		Signature:     signature,
	}

	query := `
	INSERT INTO request_funds(
		transaction_id,
		sender, receiver, 
		amount, created_at, 
		signature
	) VALUES(?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(
		query,
		t.TransactionId,
		t.Sender,
		t.Receiver,
		t.Amount,
		t.CreatedAt,
		t.Signature,
	)
	return &t, err
}
