package database

import "github.com/nyaruka/phonenumbers"

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
) (*Wallet, error) {
	query := `
		INSERT INTO wallets(
			user_id,
			wallet_address,
			wallet_name,
			initial_deposit
		) VALUES(?, ?, ?, ?)`
	_, err := db.Exec(
		query,
		userId,
		walletAddress,
		walletName,
		amount,
	)
	if err != nil {
		return nil, err
	}
	return GetWalletDetails(userId, walletAddress)
}

func WalletExists(userId int, walletAddress string) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id= ? AND wallet_address= ?)"

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

func FreezeWallet(userId int, walletAddress string) error {
	query := "UPDATE wallets SET is_active= 0 WHERE user_id= ? AND wallet_address= ?"
	_, err := db.Exec(query, userId, walletAddress)
	return err
}

func ActivateWallet(userId int, walletAddress string) error {
	query := "UPDATE wallets SET is_active= 1 WHERE user_id= ? AND wallet_address= ?"
	_, err := db.Exec(query, userId, walletAddress)
	return err
}
