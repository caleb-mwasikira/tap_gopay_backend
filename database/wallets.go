package database

import (
	"github.com/nyaruka/phonenumbers"
)

type Wallet struct {
	UserId         int     `json:"user_id"`
	Username       string  `json:"username"`
	Phone          string  `json:"phone_no"`
	Address        string  `json:"wallet_address"`
	Name           string  `json:"wallet_name"`
	InitialDeposit float64 `json:"initial_deposit"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	Balance        float64 `json:"balance"`
}

func CreateWallet(
	userId int,
	walletAddress string,
	walletName string,
	amount float64,
	requiredSignatures uint,
) (*Wallet, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO wallets(
			wallet_address,
			wallet_name,
			initial_deposit,
			required_signatures
		) VALUES(?, ?, ?, ?)`
	_, err = tx.Exec(
		query,
		walletAddress,
		walletName,
		amount,
		requiredSignatures,
	)
	if err != nil {
		return nil, err
	}

	// Add owner in wallet_owners table
	query = `
		INSERT INTO wallet_owners(wallet_address, user_id)
		VALUES(?, ?)
	`
	_, err = tx.Exec(query, walletAddress, userId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return GetWalletDetails(userId, walletAddress)
}

func WalletExists(userId int, walletAddress string) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM wallet_owners WHERE user_id= ? AND wallet_address= ?)"

	row := db.QueryRow(query, userId, walletAddress)
	err := row.Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func GetWalletDetails(userId int, walletAddress string) (*Wallet, error) {
	wallet := Wallet{
		UserId:  userId,
		Address: walletAddress,
	}

	query := `
		SELECT
			username,
			phone_no,
			wallet_name,
			is_active,
			created_at,
			balance
		FROM wallet_details
		WHERE user_id= ? AND wallet_address= ?
	`
	row := db.QueryRow(query, userId, walletAddress)
	err := row.Scan(
		&wallet.Username,
		&wallet.Phone,
		&wallet.Name,
		&wallet.IsActive,
		&wallet.CreatedAt,
		&wallet.Balance,
	)
	return &wallet, err
}

func GetAllWalletsOwnedBy(phone string, filter func(*Wallet) bool) ([]*Wallet, error) {
	num, err := phonenumbers.Parse(phone, "KE")
	if err != nil {
		return nil, err
	}
	phone = phonenumbers.Format(num, phonenumbers.INTERNATIONAL)

	query := `
		SELECT
			user_id,
			username,
			wallet_address,
			wallet_name,
			is_active,
			created_at,
			balance
		FROM wallet_details
		WHERE phone_no= ?
	`
	rows, err := db.Query(query, phone)
	if err != nil {
		return nil, err
	}

	wallets := []*Wallet{}

	for rows.Next() {
		wallet := Wallet{
			Phone: phone,
		}
		err := rows.Scan(
			&wallet.UserId,
			&wallet.Username,
			&wallet.Address,
			&wallet.Name,
			&wallet.IsActive,
			&wallet.CreatedAt,
			&wallet.Balance,
		)
		if err != nil {
			return nil, err
		}

		if filter == nil {
			wallets = append(wallets, &wallet)
			continue
		}

		if filter(&wallet) {
			wallets = append(wallets, &wallet)
		}
	}

	return wallets, err
}

func GetAllWallets(userId int) ([]*Wallet, error) {
	query := `
		SELECT
			username,
			phone_no,
			wallet_address,
			wallet_name,
			initial_deposit,
			is_active,
			created_at,
			balance
		FROM wallet_details
		WHERE user_id= ?
	`
	rows, err := db.Query(query, userId)
	if err != nil {
		return nil, err
	}

	wallets := []*Wallet{}

	for rows.Next() {
		wallet := Wallet{
			UserId: userId,
		}
		err = rows.Scan(
			&wallet.Username,
			&wallet.Phone,
			&wallet.Address,
			&wallet.Name,
			&wallet.InitialDeposit,
			&wallet.IsActive,
			&wallet.CreatedAt,
			&wallet.Balance,
		)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, &wallet)
	}
	return wallets, nil
}

func ownsWallet(userId int, walletAddress string) bool {
	var exists bool

	query := "SELECT EXISTS(SELECT 1 FROM wallet_owners WHERE user_id= ? AND wallet_address= ?)"

	err := db.QueryRow(
		query,
		userId,
		walletAddress,
	).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func FreezeWallet(userId int, walletAddress string) error {
	var err error

	if ownsWallet(userId, walletAddress) {
		query := "UPDATE wallets SET is_active= 0 WHERE wallet_address= ?"
		_, err = db.Exec(query, walletAddress)
	}

	return err
}

func ActivateWallet(userId int, walletAddress string) error {
	var err error

	if ownsWallet(userId, walletAddress) {
		query := "UPDATE wallets SET is_active= 1 WHERE wallet_address= ?"
		_, err = db.Exec(query, walletAddress)
	}

	return err
}
