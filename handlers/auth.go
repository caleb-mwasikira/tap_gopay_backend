package handlers

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/encrypt"
)

const (
	PUBLIC_KEY            string = "public_key"
	ENCRYPTED_SEED_PHRASE string = "encrypted_seed_phrase"
)

// No json tags as request will be multipart/form-data
type RegisterRequest struct {
	Username  string `validate:"min=3,max=30"`
	Email     string `validate:"email"`
	PhoneNo   string `validate:"phone_no"`
	PublicKey []byte `validate:"public_key"` // Base64-encoded public key data
}

func readFormFile(r *http.Request, fieldName string) ([]byte, error) {
	file, _, err := r.FormFile(fieldName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Data is in base64-encoded form
	b64EncodedData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(string(b64EncodedData))
}

func Register(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		api.BadRequest(w, "Error parsing multipart/form-data")
		return
	}

	// Read public key file
	pubKeyBytes, err := readFormFile(r, PUBLIC_KEY)
	if err != nil {
		api.BadRequest(w, "Error reading uploaded file %v", PUBLIC_KEY)
		return
	}

	req := RegisterRequest{
		Username:  r.FormValue("username"),
		Email:     r.FormValue("email"),
		PhoneNo:   r.FormValue("phone_no"),
		PublicKey: pubKeyBytes,
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

	err = database.CreateUser(
		req.Username, req.Email,
		req.PhoneNo,
		req.PublicKey,
	)
	if err != nil {
		api.Errorf(w, "Error creating user account", err)
		return
	}

	api.OK(w, "Created user account")
}

type LoginRequest struct {
	Email     string `json:"email" validate:"email"`
	Signature string `json:"signature" validate:"signature"` // Base64 encoded signature of user's email
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		api.BadRequest(w, "email, signature fields required")
		return
	}

	if errs := validateStruct(req); len(errs) > 0 {
		api.BadRequest2(w, errs)
		return
	}

	user, err := database.GetUser(req.Email)
	if err != nil {
		api.BadRequest(w, "Invalid email or signature")
		return
	}

	// Load users public key
	pubKey, err := encrypt.LoadPublicKeyFromBytes(user.PublicKey)
	if err != nil {
		api.Errorf(w, "Error loading user's public key; %v", err)
		return
	}

	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		api.BadRequest(w, "Expected base64 encoded signature")
		return
	}

	digest := sha256.Sum256([]byte(req.Email))
	ok := ecdsa.VerifyASN1(pubKey, digest[:], signature)
	if !ok {
		api.BadRequest(w, "Error verifiying users signature")
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
	Email     string `json:"email" validate:"email"`
	Token     string `json:"token"`
	PublicKey []byte `json:"public_key" validate:"public_key"`
}

// WARN: This implementation is risky. If user ever loses access to their
// email, then its game over for them.
func ResetPassword(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		api.BadRequest(w, "Error parsing multipart/form-data")
		return
	}

	// Read public key file
	pubKeyBytes, err := readFormFile(r, PUBLIC_KEY)
	if err != nil {
		api.BadRequest(w, "Error reading uploaded file %v", PUBLIC_KEY)
		return
	}

	req := passwordResetRequest{
		Email:     r.FormValue("email"),
		Token:     r.FormValue("token"),
		PublicKey: pubKeyBytes,
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

	err = database.ChangePublicKey(req.Email, req.PublicKey)
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
