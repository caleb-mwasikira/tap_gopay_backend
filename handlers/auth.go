package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	PhoneNo  string `json:"phone_no"`
}

func (req RegisterRequest) Validate() error {
	if err := validateName("username", req.Username); err != nil {
		return err
	}
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	if err := validatePassword(req.Password); err != nil {
		return err
	}
	if err := validatePhone(req.PhoneNo); err != nil {
		return err
	}
	return nil
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "username, email, password fields required",
		})
		return
	}

	if err = req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	userExists := database.UserExists(req.Email)
	if userExists {
		jsonResponse(w, http.StatusConflict, map[string]string{
			"message": "User account already exists",
		})
		return
	}

	err = database.CreateUser(req.Username, req.Email, req.Password, req.PhoneNo)
	if err != nil {
		log.Printf("Error creating user account; %v\n", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error creating user account",
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Created user account",
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req LoginRequest) Validate() error {
	err := validateEmail(req.Email)
	if err != nil {
		return err
	}

	err = validatePassword(req.Password)
	if err != nil {
		return err
	}
	return nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "username, password fields required",
		})
		return
	}

	if err = req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	user, err := database.GetUser(req.Email)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid username or password",
		})
		return
	}

	passwordMatch := verifyPassword(user.Password, req.Password)
	if !passwordMatch {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "Invalid username or password",
		})
		return
	}

	accessToken, err := generateToken(*user)
	if err != nil {
		log.Printf("Error generating JWT; %v\n", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error logging in user",
		})
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

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Login successful",
	})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

func (req forgotPasswordRequest) Validate() error {
	err := validateEmail(req.Email)
	return err
}

func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "email field required",
		})
		return
	}

	if err = req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	// If user does not exist we still send a 200 OK response.
	// this is done to prevent people from searching emails registered with
	// the system via this route
	userExists := database.UserExists(req.Email)
	if !userExists {
		jsonResponse(w, http.StatusOK, map[string]string{
			"message": "Password reset token has been sent to your email",
		})
		return
	}

	resetToken, err := database.CreatePasswordResetToken(req.Email, 72*time.Hour)
	if err != nil {
		log.Printf("Error creating password reset token; %v\n", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error creating password reset token",
		})
		return
	}

	// Launch this in goroutine so it doesn't delay our main request
	go sendPasswordResetEmail(req.Email, resetToken.Token)

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Password reset token has been sent to your email",
	})
}

type passwordResetRequest struct {
	Email       string `json:"email"`
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (req passwordResetRequest) Validate() error {
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	if err := validatePassword(req.NewPassword); err != nil {
		return err
	}
	return nil
}

func ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": "email, token and new_password fields required",
		})
		return
	}

	if err = req.Validate(); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
		return
	}

	passwordResetToken, err := database.GetPasswordResetToken(req.Email, req.Token)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{
			"message": "Invalid or expired token",
		})
		return
	}

	if passwordResetToken.Token != req.Token {
		jsonResponse(w, http.StatusNotFound, map[string]string{
			"message": "Invalid or expired token",
		})
		return
	}

	err = database.ChangePassword(req.Email, req.NewPassword)
	if err != nil {
		log.Printf("Error changing user password; %v\n", err)
		jsonResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "Error changing user password",
		})
		return
	}

	// Invalidate token to prevent re-use
	go database.DeletePasswordResetToken(req.Email, req.Token)

	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Password reset successful",
	})
}

func VerifyLoginHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Its Saul Good Man",
	})
}
