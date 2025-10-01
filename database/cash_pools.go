package database

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"sync"
	"time"
)

type CashPoolType string

const (
	SplitBill          CashPoolType = "split_bill"
	Chama              CashPoolType = "chama"
	BusinessInvestment CashPoolType = "business_investment"
)

type CashPool struct {
	Creator         WalletOwner  `json:"creator"`
	PoolName        string       `json:"pool_name"`
	PoolType        CashPoolType `json:"cash_pool_type"`
	Description     string       `json:"description"`
	WalletAddress   string       `json:"wallet_address"`
	TargetAmount    float64      `json:"target_amount"`
	Receiver        WalletOwner  `json:"receiver"`
	ExpiresAt       string       `json:"expires_at"`
	Status          string       `json:"status"`
	CollectedAmount float64      `json:"collected_amount"`
	CreatedAt       time.Time    `json:"created_at"`
}

func CreateCashPool(
	cashPoolCreator int,
	poolName string,
	poolType CashPoolType,
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
			pool_type,
			description,
			target_amount,
			wallet_address,
			receiver,
			expires_at
		) VALUES(?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(
		query,
		poolName,
		poolType,
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
			pool_type,
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
		&p.PoolType,
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
		AND expires_at < NOW() AND status <> 'refunded'
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

type transaction struct {
	transactionCode string
	sender          string
	receiver        string
	amount          float64
}

type failedRefund struct {
	transaction
	err error
}

func refundTransaction(wg *sync.WaitGroup, t transaction, failedRefundsChan chan<- failedRefund) {
	defer wg.Done()

	// 🎵 Raindrops keep falling on my head,
	// but just like the guy whose feet are too big for his bed,
	// nothing seems to fit...
	// Those raindrops keep falling on my head,
	// they keep falling...

	systemUserId, err := getSystemUserId()
	if err != nil {
		failedRefundsChan <- failedRefund{
			transaction: t,
			err:         err,
		}
		return
	}

	var (
		refundTransactionCode string  = t.transactionCode
		sender                string  = t.receiver // Flip sender and receiver to reverse funds
		receiver              string  = t.sender
		amount                float64 = t.amount
		fee                   float64 = 0.0
		timestamp             string  = time.Now().Format(time.RFC3339)
	)

	// Sign payload
	payload := fmt.Sprintf("%s|%s|%.2f|%.2f|%s", sender, receiver, amount, fee, timestamp)
	payloadHash := sha256.Sum256([]byte(payload))
	hmacSignature, secretKeyHash, err := signPayload(payloadHash[:])
	if err != nil {
		failedRefundsChan <- failedRefund{
			transaction: t,
			err:         err,
		}
		return
	}

	_, err = CreateRefundTransaction(
		*systemUserId,
		refundTransactionCode,
		sender,
		receiver,
		t.amount,
		fee,
		timestamp,
		base64.StdEncoding.EncodeToString(hmacSignature),
		base64.StdEncoding.EncodeToString(secretKeyHash),
	)
	if err != nil {
		failedRefundsChan <- failedRefund{
			transaction: t,
			err:         err,
		}
	}
}

func RefundExpiredCashPool(cashPoolAddress string) ([]failedRefund, error) {
	log.Println("Refunding cash pool; ", cashPoolAddress)

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
		return nil, err
	}

	wg := sync.WaitGroup{}
	failedRefundsChan := make(chan failedRefund, 10)

	for rows.Next() {
		var t transaction

		err = rows.Scan(
			&t.transactionCode,
			&t.sender,
			&t.receiver,
			&t.amount,
		)
		if err != nil {
			log.Printf("Error scanning database row; %v\n", err)
			break
		}

		wg.Add(1)
		go refundTransaction(&wg, t, failedRefundsChan)
	}

	wg.Wait()
	close(failedRefundsChan)

	// Collect failed refunds
	failedRefunds := []failedRefund{}

	for failedRefund := range failedRefundsChan {
		failedRefunds = append(failedRefunds, failedRefund)
	}

	if len(failedRefunds) == 0 {
		query = "UPDATE cash_pools SET status= 'refunded' WHERE wallet_address= ?"
		_, err = db.Exec(query, cashPoolAddress)
	}
	return failedRefunds, err
}

func RemoveCashPool(cashPoolAddress string) error {
	query := "UPDATE cash_pools SET expires_at= NOW() WHERE wallet_address= ?"
	_, err := db.Exec(query, cashPoolAddress)
	return err
}
