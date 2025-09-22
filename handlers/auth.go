package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"golang.org/x/crypto/bcrypt"
)

const (
	ENCRYPTED_SEED_PHRASE string = "encrypted_seed_phrase"
)

// No json tags as request will be multipart/form-data
type RegisterRequest struct {
	Username  string `json:"username" validate:"min=3,max=30"`
	Email     string `json:"email" validate:"email"`
	Password  string `json:"password" validate:"password"`
	PhoneNo   string `json:"phone_no" validate:"phone_no"`
	PublicKey string `json:"public_key" validate:"public_key"` // Base64 encoded public key in PEM format
}

func Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	userExists := database.UserExists(req.Email)
	if userExists {
		api.Conflict(w, "User account already exists")
		return
	}

	err = database.CreateUser(
		req.Username, req.Email,
		req.Password, req.PhoneNo,
		req.PublicKey,
	)
	if err != nil {
		api.Errorf(w, "Error creating user account", err)
		return
	}

	api.OK(w, "Created user account")
}

func verifyPassword(dbPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
	return err == nil
}

type LoginRequest struct {
	Email     string `json:"email" validate:"email"`
	Password  string `json:"password" validate:"password"`
	PublicKey string `json:"public_key" validate:"public_key"` // Base64 encoded public key in PEM format
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err = validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	user, err := database.GetUser(req.Email)
	if err != nil {
		api.BadRequest(w, "Invalid username or password", nil)
		return
	}

	passwordMatch := verifyPassword(user.Password, req.Password)
	if !passwordMatch {
		api.BadRequest(w, "Invalid username or password", nil)
		return
	}

	err = database.CreatePublicKey(req.Email, req.PublicKey)
	if err != nil {
		api.Errorf(w, "Error logging in user", fmt.Errorf("error creating public key; %v", err))
		return
	}

	accessToken, err := generateToken(*user)
	if err != nil {
		api.Errorf(w, "Error logging in user", fmt.Errorf("error generating JWT; %v", err))
		return
	}

	// Set access token in user's cookies
	now := time.Now()
	expires := now.Add(72 * time.Hour)

	http.SetCookie(w, &http.Cookie{
		Name:    LOGIN_COOKIE,
		Value:   accessToken,
		Expires: expires,
	})

	api.OK(w, "Login successful")
}

type forgotPasswordRequest struct {
	Email string `json:"email" validate:"email"`
}

func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	// If user does not exist we still send a 200 OK response.
	// this is done to prevent people from searching emails registered with
	// the system via this route
	userExists := database.UserExists(req.Email)
	if !userExists {
		api.OK(w, "Password reset token has been sent to your email")
		return
	}

	resetToken, err := database.CreatePasswordResetToken(req.Email, 72*time.Hour)
	if err != nil {
		api.Errorf(w, "Error creating password reset token", err)
		return
	}

	// Launch this in goroutine so it doesn't delay our main request
	go sendPasswordResetEmail(req.Email, resetToken.Token)

	api.OK(w, "Password reset token has been sent to your email")
}

type passwordResetRequest struct {
	Email    string `json:"email" validate:"email"`
	Token    string `json:"token"`
	Password string `json:"password" validate:"password"`
}

// WARN: This implementation is risky. If user ever loses access to their
// email, then its game over for them.
func ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		api.BadRequest(w, "Error parsing request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		api.BadRequest(w, err.Error(), nil)
		return
	}

	passwordResetToken, err := database.GetPasswordResetToken(req.Email, req.Token)
	if err != nil {
		api.NotFound(w, "Invalid or expired token")
		return
	}

	if passwordResetToken.Token != req.Token {
		api.NotFound(w, "Invalid or expired token")
		return
	}

	err = database.ChangePassword(req.Email, req.Password)
	if err != nil {
		api.Errorf(w, "Error changing user password", err)
		return
	}

	// Invalidate token to prevent re-use
	go database.DeletePasswordResetToken(req.Email, req.Token)

	api.OK(w, "Password reset successful")
}

func VerifyLogin(w http.ResponseWriter, r *http.Request) {
	api.OK(w, "Its Saul Good Man")
}
