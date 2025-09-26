package database

import "time"

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
	creatorUserId int,
	poolName string,
	description string,
	targetAmount float64,
	receiver string,
	expiresAt string,
) (*CashPool, error) {
	walletAddress := generateWalletAddress(cashPool)

	query := `
		INSERT INTO cash_pools(
			creator_user_id,
			pool_name,
			description,
			target_amount,
			wallet_address,
			receiver,
			expires_at
		) VALUES(?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(
		query,
		creatorUserId,
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
