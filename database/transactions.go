package database

import (
	"math/rand"
	"strings"
	"time"
)

const (
	TRANSACTION_ID_LEN int = 12
)

type Transaction struct {
	TransactionId string  `json:"transaction_id"`
	Sender        string  `json:"sender"`
	Receiver      string  `json:"receiver"`
	Amount        float64 `json:"amount"`
	CreatedAt     string  `json:"created_at"` // RFC3339 formatted string

	// Base64 encoded signature
	// Request signed by sender
	Signature string `json:"signature"`
}

func generateTransactionId() string {
	rand.NewSource(time.Now().UnixNano())

	str := strings.Builder{}

	for range TRANSACTION_ID_LEN {
		// Generate a random integer between 65 and 90 (inclusive)
		randInt := 65 + rand.Intn(90-65+1)
		randomChar := rune(randInt)
		str.WriteRune(randomChar)
	}

	return str.String()
}

func CreateTransaction(
	sender, receiver string,
	amount float64,
	created_at, signature string,
) (*Transaction, error) {
	t := Transaction{
		TransactionId: generateTransactionId(),
		Sender:        sender,
		Receiver:      receiver,
		Amount:        amount,
		CreatedAt:     created_at,
		Signature:     signature,
	}

	query := `
	INSERT INTO transactions(
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

func CreateRequestFunds(
	sender, receiver string,
	amount float64,
	created_at, signature string,
) (*Transaction, error) {
	t := Transaction{
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
