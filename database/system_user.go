package database

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	systemUserId *int         = nil
	mutex        sync.RWMutex = sync.RWMutex{}
)

// Signs blob of data using system user's SECRET_KEY.
// Returns HMAC signature, secret key hash and an error if any
func signPayload(data []byte) ([]byte, []byte, error) {
	secretKey := os.Getenv("SECRET_KEY")
	if isEmpty(secretKey) {
		return nil, nil, fmt.Errorf("SECRET_KEY is empty")
	}

	secretKeyHash := sha256.Sum256([]byte(secretKey))

	hash := hmac.New(sha256.New, []byte(secretKey))
	signature := hash.Sum(data)
	return signature, secretKeyHash[:], nil
}

func CreateSystemUser() error {
	log.Println("Creating system user...")

	secretKey := os.Getenv("SECRET_KEY")
	hashedPassword, err := hashPassword(secretKey)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	query := `
		INSERT IGNORE INTO users(
			username,
			email,
			password,
			role
		)
		VALUES('SYSTEM','tapgopay@gmail.com',?,'system')
	`
	result, err := tx.Exec(
		query,
		hashedPassword,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	userId, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	// Create system wallet
	walletAddress := generateWalletAddress(bankWallet)

	query = `
		INSERT INTO wallets(
			wallet_address,
			wallet_name,
			initial_deposit,
			total_owners,
			required_signatures
		) VALUES(?, ?, ?, ?, ?)`
	_, err = tx.Exec(
		query,
		walletAddress,
		"SYSTEM",
		0.0,
		1,
		1,
	)
	if err != nil {
		return err
	}

	// Add owner in wallet_owners table
	query = `
		INSERT INTO wallet_owners(wallet_address, user_id)
		VALUES(?, ?)
	`
	_, err = tx.Exec(query, walletAddress, userId)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func getSystemUserId() (*int, error) {
	mutex.RLock()
	sysUserId := systemUserId
	mutex.RUnlock()

	if sysUserId != nil {
		return sysUserId, nil
	}

	query := "SELECT id FROM users WHERE role= 'system'"
	err := db.QueryRow(query).Scan(&sysUserId)
	if err != nil {
		return nil, err
	}

	mutex.Lock()
	systemUserId = sysUserId
	mutex.Unlock()

	return sysUserId, err
}
