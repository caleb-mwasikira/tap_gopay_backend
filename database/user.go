package database

type User struct {
	Id            int    `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	PhoneNo       string `json:"phone_no"`
	EmailVerified bool   `json:"email_verified"`
	PublicKey     []byte `json:"public_key"`
}

func CreateUser(
	username, email, phoneNo string,
	pubKey []byte,
) error {
	query := `
		INSERT INTO users(
			username, email, phone_no, public_key
		)
		VALUES(?, ?, ?, ?)
	`
	_, err := db.Exec(
		query,
		username,
		email,
		phoneNo,
		pubKey,
	)
	return err
}

// Fetches user by their email
func GetUser(email string) (*User, error) {
	query := `
		SELECT id, username, email, phone_no, public_key
		FROM users WHERE email = ?
	`
	row := db.QueryRow(query, email)

	user := User{}
	err := row.Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.PhoneNo,
		&user.PublicKey,
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

func ChangePublicKey(email string, pubKeyBytes []byte) error {
	query := "UPDATE users SET public_key= ? WHERE email= ?"
	_, err := db.Exec(query, pubKeyBytes, email)
	return err
}
