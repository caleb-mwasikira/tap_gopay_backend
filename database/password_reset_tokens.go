package database

import (
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

func CreatePasswordResetToken(email string, duration time.Duration) (*PasswordResetToken, error) {
	now := time.Now()
	resetToken := PasswordResetToken{
		Email:     email,
		Token:     generatetoken(MIN_token_LEN),
		ExpiresAt: now.Add(duration),
	}

	query := "INSERT INTO password_reset_tokens(email, token, expires_at) VALUES(?, ?, ?)"
	_, err := db.Exec(
		query,
		email,
		resetToken.Token,
		resetToken.ExpiresAt,
	)
	return &resetToken, err
}

// Fetches password_reset_token by user's email and token
func GetPasswordResetToken(email, token string) (*PasswordResetToken, error) {
	query := "SELECT * FROM password_reset_tokens WHERE email= ? AND token= ? AND expires_at > ?"
	row := db.QueryRow(query, email, token, time.Now())

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

func DeletePasswordResetToken(email, token string) error {
	query := "DELETE password_reset_tokens WHERE email=? AND token=?"
	_, err := db.Exec(
		query,
		email,
		token,
	)
	return err
}
