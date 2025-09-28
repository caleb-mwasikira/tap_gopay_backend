package database

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"time"
)

type CashPool struct {
	Creator         WalletOwner `json:"creator"`
	PoolName        string      `json:"pool_name"`
	Description     string      `json:"description"`
	WalletAddress   string      `json:"wallet_address"`
	TargetAmount    float64     `json:"target_amount"`
	Receiver        WalletOwner `json:"receiver"`
	ExpiresAt       string      `json:"expires_at"`
	Status          string      `json:"status"`
	CollectedAmount float64     `json:"collected_amount"`
	CreatedAt       time.Time   `json:"created_at"`
}

func CreateCashPool(
	cashPoolCreator int,
	poolName string,
	description string,
	targetAmount float64,
	receiver string,
	expiresAt string,
) (*CashPool, error) {
	walletAddress := generateWalletAddress(cashPool)

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO cash_pools(
			pool_name,
			description,
			target_amount,
			wallet_address,
			receiver,
			expires_at
		) VALUES(?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(
		query,
		poolName,
		description,
		targetAmount,
		walletAddress,
		receiver,
		expiresAt,
	)
	if err != nil {
		return nil, err
	}

	query = "INSERT INTO wallet_owners(user_id, wallet_address) VALUES(?, ?)"
	_, err = tx.Exec(query, cashPoolCreator, walletAddress)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return nil, err
	}

	return GetCashPool(walletAddress)
}

func GetCashPool(walletAddress string) (*CashPool, error) {
	var p CashPool

	query := `
		SELECT
			creators_username,
			creators_email,
			pool_name,
			description,
			wallet_address,
			target_amount,
			receivers_username,
			receivers_email,
			receivers_wallet_address,
			expires_at,
			status,
			collected_amount,
			created_at
		FROM cash_pool_details
		WHERE wallet_address= ? LIMIT 1
	`
	err := db.QueryRow(query, walletAddress).Scan(
		&p.Creator.Username,
		&p.Creator.Email,
		&p.PoolName,
		&p.Description,
		&p.WalletAddress,
		&p.TargetAmount,
		&p.Receiver.Username,
		&p.Receiver.Email,
		&p.Receiver.WalletAddress,
		&p.ExpiresAt,
		&p.Status,
		&p.CollectedAmount,
		&p.CreatedAt,
	)
	return &p, err
}

// Returns wallet addresses of all expired cash pools
// that did not reach their funding goal.
func GetExpiredCashPools() ([]string, error) {
	query := `
		SELECT wallet_address
		FROM cash_pool_details
		WHERE collected_amount < target_amount
		AND expires_at < NOW()
		AND status <> 'refunded'
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	cashPools := []string{}
	var cashPool string

	for rows.Next() {
		err = rows.Scan(&cashPool)
		if err != nil {
			return nil, err
		}
		cashPools = append(cashPools, cashPool)
	}
	return cashPools, nil
}

type refundRequest struct {
	transactionCode string
	sender          string
	receiver        string
	amount          float64
}

func handleRefundRequest(request refundRequest) {
	// 🎵 Raindrops keep falling on my head,
	// but just like the guy whose feet are too big for his bed,
	// nothing seems to fit...
	// Those raindrops keep falling on my head,
	// they keep falling...

	sysUserId, err := getSystemUserId()
	if err != nil {
		log.Printf("Error fetching system user; %v\n", err)
		return
	}

	var (
		refundTransactionCode string  = request.transactionCode
		sender                string  = request.sender
		receiver              string  = request.receiver
		amount                float64 = request.amount
		fee                   float64 = 0.0
		timestamp             string  = time.Now().Format(time.RFC3339)
	)

	// Sign payload
	payload := fmt.Sprintf("%s|%s|%.2f|%.2f|%s", sender, receiver, amount, fee, timestamp)
	payloadHash := sha256.Sum256([]byte(payload))
	hmacSignature, secretKeyHash, err := signPayload(payloadHash[:])
	if err != nil {
		log.Printf("Error signing refund transaction; %v\n", err)
		return
	}

	_, err = CreateRefundTransaction(
		*sysUserId,
		refundTransactionCode,
		sender,
		receiver,
		request.amount,
		fee,
		timestamp,
		base64.StdEncoding.EncodeToString(hmacSignature),
		base64.StdEncoding.EncodeToString(secretKeyHash),
	)
	if err != nil {
		log.Printf("Error creating refund transaction; %v\n", err)
	}
}

func RefundExpiredCashPool(cashPoolAddress string) error {
	query := `
		SELECT
			transaction_code,
			sender,
			receiver,
			amount
		FROM transactions
		WHERE receiver= ?
	`
	rows, err := db.Query(query, cashPoolAddress)
	if err != nil {
		return err
	}

	for rows.Next() {
		var refund refundRequest

		err = rows.Scan(
			&refund.transactionCode,
			&refund.receiver, // Flip sender and receiver to reverse funds
			&refund.sender,
			&refund.amount,
		)
		if err != nil {
			return err
		}

		handleRefundRequest(refund)
	}

	query = "UPDATE cash_pools SET status= 'refunded' WHERE wallet_address= ?"
	_, err = db.Exec(query, cashPoolAddress)
	return err
}
