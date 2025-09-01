package database

import (
	"database/sql"
	"fmt"
	"math/rand/v2"
	"time"
)

const (
	MIN_token_LEN int = 6
)

type PasswordResetToken struct {
	Id        int       `json:"id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func generatetoken(length int) string {
	token := ""
	for range length {
		rand_digit := rand.IntN(10)
		token += fmt.Sprint(rand_digit)
	}
	return token
}

func NewPasswordResetToken(email string, duration time.Duration) *PasswordResetToken {
	now := time.Now()
	expiry_time := now.Add(duration)
	token := generatetoken(MIN_token_LEN)

	return &PasswordResetToken{
		Email:     email,
		Token:     token,
		CreatedAt: now,
		ExpiresAt: expiry_time,
	}
}

type PasswordResetModel struct {
	db *sql.DB
}

func NewPasswordResetModel() *PasswordResetModel {
	return &PasswordResetModel{
		db: db,
	}
}

func (m *PasswordResetModel) Insert(passwordResetToken PasswordResetToken) (int64, error) {
	query := "INSERT INTO password_reset_tokens(email, token, expires_at) VALUES(?, ?, ?)"
	result, err := m.db.Exec(
		query,
		passwordResetToken.Email,
		passwordResetToken.Token,
		passwordResetToken.ExpiresAt,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Fetches password_reset_token by user's email and token
func (m *PasswordResetModel) Get(email, token string) (*PasswordResetToken, error) {
	query := "SELECT * FROM password_reset_tokens WHERE email= ? AND token= ? AND expires_at > ?"
	row := m.db.QueryRow(query, email, token, time.Now())

	passwordResetToken := PasswordResetToken{}
	err := row.Scan(
		&passwordResetToken.Id,
		&passwordResetToken.Email,
		&passwordResetToken.Token,
		&passwordResetToken.ExpiresAt,
		&passwordResetToken.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &passwordResetToken, nil
}

func (m *PasswordResetModel) Delete(email, token string) (int64, error) {
	query := "DELETE password_reset_tokens WHERE email=? AND token=?"
	result, err := m.db.Exec(
		query,
		email,
		token,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
