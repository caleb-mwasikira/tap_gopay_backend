package database

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
)

type User struct {
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

func NewUser(username, email, password, phone_no string) (*User, error) {
	return &User{
		Username: username,
		Email:    email,
		Password: hashPassword(password),
		PhoneNo:  phone_no,
	}, nil
}

type UserModel struct {
	db *sql.DB
}

func NewUserModel() *UserModel {
	return &UserModel{
		db: db,
	}
}

// Saves a user instance onto the database.
//
//	!! Make sure you create your user with NewUser() inorder
//	for it to do password hashing
func (m *UserModel) Insert(user User) (int64, error) {
	query := "INSERT INTO users(username, email, password, phone_no) VALUES(?, ?, ?, ?)"
	result, err := m.db.Exec(
		query,
		user.Username,
		user.Email,
		user.Password,
		user.PhoneNo,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Fetches user by their email
func (m *UserModel) Get(email string) (*User, error) {
	query := "SELECT username, email, password, phone_no FROM users WHERE email = ?"
	row := m.db.QueryRow(query, email)

	user := User{}
	err := row.Scan(
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

func (m *UserModel) Exists(email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`
	err := db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// Changes a user's password. Hashes the password for you; you can pass
// in the password as plaintext
func (m *UserModel) ChangePassword(email string, newPassword string) (int64, error) {
	query := "UPDATE users SET password = ? WHERE email = ?"
	result, err := m.db.Exec(
		query,
		hashPassword(newPassword),
		email,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
