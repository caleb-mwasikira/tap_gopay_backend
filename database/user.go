package database

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type User struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	PhoneNo  string `json:"phone_no"`
}

func hashPassword(password string) string {
	hash := hmac.New(sha256.New, []byte(SECRET_KEY))
	digest := hash.Sum([]byte(password))
	return hex.EncodeToString(digest)
}

// Creates a new user and saves them to the database.
// Hashes the password before saving it to the database.
func CreateUser(username, email, password, phoneNo string) error {
	query := "INSERT INTO users(username, email, password, phone_no) VALUES(?, ?, ?, ?)"
	_, err := db.Exec(
		query,
		username,
		email,
		hashPassword(password),
		phoneNo,
	)
	return err
}

// Fetches user by their email
func GetUser(email string) (*User, error) {
	query := "SELECT id, username, email, password, phone_no FROM users WHERE email = ?"
	row := db.QueryRow(query, email)

	user := User{}
	err := row.Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.PhoneNo,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UserExists(email string) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`
	err := db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

// Changes a user's password. Hashes the password for you; you can pass
// in the password as plaintext
func ChangePassword(email string, newPassword string) error {
	query := "UPDATE users SET password = ? WHERE email = ?"
	_, err := db.Exec(
		query,
		hashPassword(newPassword),
		email,
	)
	return err
}
