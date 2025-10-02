package database

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

type Wallet struct {
	UserId         int     `json:"user_id"`
	Username       string  `json:"username"`
	PhoneNo        string  `json:"phone_no"`
	WalletAddress  string  `json:"wallet_address"`
	WalletName     string  `json:"wallet_name"`
	InitialDeposit float64 `json:"initial_deposit"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	Balance        float64 `json:"balance"`
}

type walletType string

const (
	WALLET_ADDR_LEN int = 12

	bankWallet     walletType = "00"
	individual     walletType = "11"
	multiSignature walletType = "22"
	cashPool       walletType = "33"
)

func generateWalletAddress(walletTyp walletType) string {
	str := []string{}

	// prefix
	for _, char := range walletTyp {
		str = append(str, fmt.Sprintf("%c", char))
	}

	remainingLength := WALLET_ADDR_LEN - len(walletTyp)
	numDigits := len(str)

	for range remainingLength {
		if numDigits%4 == 0 {
			str = append(str, " ")
		}

		num := rand.IntN(10)
		str = append(str, fmt.Sprintf("%d", num))
		numDigits++
	}
	return strings.Join(str, "")
}

func CreateWallet(
	userId int,
	walletName string,
	initialDeposit float64,
	totalOwners uint,
	requiredSignatures uint,
) (*Wallet, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	walletTyp := individual
	if totalOwners > 1 {
		walletTyp = multiSignature
	}
	walletAddress := generateWalletAddress(walletTyp)

	query := `
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
		walletName,
		initialDeposit,
		totalOwners,
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

	return GetWallet(userId, walletAddress)
}

func GetWallet(userId int, walletAddress string) (*Wallet, error) {
	wallet := Wallet{
		UserId:        userId,
		WalletAddress: walletAddress,
	}

	query := `
		SELECT
			username,
			phone_no,
			wallet_name,
			initial_deposit,
			is_active,
			created_at,
			balance
		FROM wallet_details
		WHERE user_id= ? AND wallet_address= ?
	`
	row := db.QueryRow(query, userId, walletAddress)
	err := row.Scan(
		&wallet.Username,
		&wallet.PhoneNo,
		&wallet.WalletName,
		&wallet.InitialDeposit,
		&wallet.IsActive,
		&wallet.CreatedAt,
		&wallet.Balance,
	)
	return &wallet, err
}

func GetWalletsOwnedByPhoneNo(phone string, filter func(*Wallet) bool) ([]*Wallet, error) {
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
			PhoneNo: phone,
		}
		err := rows.Scan(
			&wallet.UserId,
			&wallet.Username,
			&wallet.WalletAddress,
			&wallet.WalletName,
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
			&wallet.PhoneNo,
			&wallet.WalletAddress,
			&wallet.WalletName,
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

func OwnsWallet(userId int, walletAddress string) bool {
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

func FreezeWallet(walletAddress string) error {
	query := "UPDATE wallets SET is_active= 0 WHERE wallet_address= ?"
	_, err := db.Exec(query, walletAddress)
	return err
}

func ActivateWallet(walletAddress string) error {
	query := "UPDATE wallets SET is_active= 1 WHERE wallet_address= ?"
	_, err := db.Exec(query, walletAddress)
	return err
}

func formatPhoneNumber(phone string) (string, bool) {
	if isEmpty(phone) {
		return "", false
	}

	number, err := phonenumbers.Parse(phone, "KE")
	if err != nil {
		return "", false
	}

	ok := phonenumbers.IsValidNumber(number)
	if ok {
		phone = phonenumbers.Format(number, phonenumbers.INTERNATIONAL)
		return phone, true
	}
	return "", false
}

// Aliases must either be valid emails, phone numbers, wallet addresses
// or a combination of both
func GetUserIds(aliases ...string) ([]int, error) {
	if len(aliases) == 0 {
		return nil, nil
	}

	// build placeholders (?, ?, ?)
	placeholders := strings.Repeat("?,", len(aliases))
	placeholders = placeholders[:len(placeholders)-1] // drop last comma

	// build query
	query := fmt.Sprintf(`
		SELECT id AS user_id
		FROM users
		WHERE email IN (%s) OR phone_no IN (%s)

		UNION ALL

		SELECT user_id
		FROM wallet_owners
		WHERE wallet_address IN (%s)
	`, placeholders, placeholders, placeholders)

	// args = aliases x3 (for email, phone_no and wallet_address)
	values := append(aliases, append(aliases, aliases...)...)
	args := []any{}

	for _, value := range values {
		phone, ok := formatPhoneNumber(value)
		if ok {
			value = phone
		}
		args = append(args, value)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIds []int

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		userIds = append(userIds, id)
	}

	return userIds, nil
}

// Adds user as wallet owner.
// A user can only be added as a wallet owner by another wallet owner.
func AddWalletOwner(
	loggedInUser int,
	userId int,
	walletAddress string,
) error {
	query := "INSERT INTO wallet_owners(user_id, wallet_address) VALUES(?, ?)"
	_, err := db.Exec(query, userId, walletAddress)
	return err
}

func isOriginalOwner(userId int, walletAddress string) bool {
	var exists bool

	query := `
		SELECT EXISTS(
			SELECT 1 FROM wallet_owners
			WHERE user_id= ?
			AND wallet_address= ?
			AND is_original_owner= TRUE
		)`

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

func RemoveWalletOwner(loggedInUser int, userId int, walletAddress string) error {
	// Only original wallet creator can remove wallet owners
	if !isOriginalOwner(loggedInUser, walletAddress) {
		return fmt.Errorf("user does not have permission to remove wallet owner")
	}

	query := "DELETE FROM wallet_owners WHERE user_id= ? AND wallet_address= ?"
	_, err := db.Exec(query, userId, walletAddress)
	return err
}
