package database

import (
	"strings"

	"github.com/nyaruka/phonenumbers"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id            int    `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	PhoneNo       string `json:"phone_no"`
	EmailVerified bool   `json:"email_verified"`
	Role          string `json:"role"`
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

// Inserts a new user into the database and saves
// their registered public key.
// You can pass in password in plaintext, the function
// hashes the password for you.
func CreateUser(
	username,
	email,
	password,
	phone,
	b64EncodedPubKey string,
) error {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return err
	}

	// Parse phone number
	num, err := phonenumbers.Parse(phone, "KE")
	if err != nil {
		return err
	}
	phone = phonenumbers.Format(num, phonenumbers.INTERNATIONAL)

	query := `
		INSERT INTO users(
			username,
			email,
			password,
			phone_no
		)
		VALUES(?, ?, ?, ?)
	`
	_, err = db.Exec(
		query,
		username,
		email,
		hashedPassword,
		phone,
	)
	if err != nil {
		return err
	}

	// Save public key in database
	return CreatePublicKey(email, b64EncodedPubKey)
}

// Fetches user by their email
func GetUser(email string) (*User, error) {
	query := `
		SELECT
			id,
			username,
			email,
			password,
			phone_no,
			role
		FROM users WHERE email = ?
	`
	row := db.QueryRow(query, email)

	var user User

	err := row.Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.PhoneNo,
		&user.Role,
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

func ChangePassword(email, password string) error {
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return err
	}

	query := "UPDATE users SET password= ? WHERE email= ?"
	_, err = db.Exec(query, hashedPassword, email)
	return err
}

func isEmpty(value string) bool {
	return strings.TrimSpace(value) == ""
}

func GetUserIdFromEmailOrPhoneNo(email, phone string) (int, error) {
	if !isEmpty(phone) {
		// Format phone number into INTERNATIONAL format.
		// This makes search for phone numbers more effective.
		// Without this, searching the phone numbers 0789689745, +254789689745 and 0789 689 745
		// will yield different results even though they are basically the same phone numbers.
		number, err := phonenumbers.Parse(phone, "KE")
		if err != nil {
			return 0, err
		}
		phone = phonenumbers.Format(number, phonenumbers.INTERNATIONAL)
	}

	query := `
		SELECT id
		FROM users WHERE email = ? OR phone_no= ?
		LIMIT 1
	`

	var userId int
	err := db.QueryRow(query, email, phone).Scan(&userId)
	return userId, err
}
