package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

type RegisterRequest struct {
	Username string `json:"username" validate:"min=3,max=30"`
	Email    string `json:"email" validate:"email"`
	Password string `json:"password" validate:"password"`
	PhoneNo  string `json:"phone_no" validate:"phone_no"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		api.BadRequest(w, "username, email, password fields required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
		return
	}

	userExists := database.UserExists(req.Email)
	if userExists {
		api.Conflict(w, "User account already exists")
		return
	}

	err = database.CreateUser(req.Username, req.Email, req.Password, req.PhoneNo)
	if err != nil {
		api.Errorf(w, "Error creating user account", err)
		return
	}

	api.OK(w, "Created user account")
}

type LoginRequest struct {
	Email    string `json:"email" validate:"email"`
	Password string `json:"password" validate:"password"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		api.BadRequest(w, "username, password fields required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
		return
	}

	user, err := database.GetUser(req.Email)
	if err != nil {
		api.BadRequest(w, "Invalid username or password")
		return
	}

	passwordMatch := verifyPassword(user.Password, req.Password)
	if !passwordMatch {
		api.BadRequest(w, "Invalid username or password")
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
		api.BadRequest(w, "email field required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
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
	Email       string `json:"email" validate:"email"`
	Token       string `json:"token"`
	NewPassword string `json:"new_password" validate:"password"`
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		api.BadRequest(w, "email, token and new_password fields required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
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

	err = database.ChangePassword(req.Email, req.NewPassword)
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
