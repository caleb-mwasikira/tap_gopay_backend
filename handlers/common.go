package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
	"github.com/golang-jwt/jwt/v5"
)

type key string

const (
	USER_CTX_KEY key = "USER_CTX_KEY"
)

var (
	SECRET_KEY      string
	ANDROID_API_KEY string
)

func init() {
	utils.LoadDotenv()

	SECRET_KEY = os.Getenv("SECRET_KEY")
	if strings.TrimSpace(SECRET_KEY) == "" {
		log.Fatalln("Missing SECRET_KEY environment variable")
	}

	ANDROID_API_KEY = os.Getenv("ANDROID_API_KEY")
}

func generateToken(user database.User) (string, error) {
	data, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	b64EncodedData := base64.StdEncoding.EncodeToString(data)
	now := time.Now()
	expiry := now.Add(72 * time.Hour)

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"iat": now.Unix(),
			"exp": expiry.Unix(),
			"iss": "fusion",
			"sub": b64EncodedData,
		},
	)
	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	return tokenString, err
}

// Verifies a json web token and returns the object stored
// in "sub" subject field. expects obj parameter to be a pointer of type T
func validToken(tokenString string, obj any) bool {
	token, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(SECRET_KEY), nil
		},
	)
	if err != nil {
		log.Printf("Error parsing jwt; %v\n", err)
		return false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		// Get subject - stored as base64 data
		b64EncodedData, ok := claims["sub"].(string)
		if !ok {
			log.Println("Unexpected \"sub\" type in jwt")
			return false
		}

		data, err := base64.StdEncoding.DecodeString(b64EncodedData)
		if err != nil {
			log.Printf("Error decoding \"sub\" value of jwt; %v\n", err)
			return false
		}

		// Unmarshal the data into the param object
		err = json.Unmarshal(data, obj)
		return err == nil
	}

	return false
}

// func hashUserPassword(password string) string {
// 	hash := hmac.New(sha256.New, []byte(SECRET_KEY))
// 	digest := hash.Sum([]byte(password))
// 	return fmt.Sprintf("%x", digest)
// }

func verifyPassword(dbPassword, password string) bool {
	hash := hmac.New(sha256.New, []byte(SECRET_KEY))
	mac2 := hash.Sum([]byte(password))
	hmacPassword, err := hex.DecodeString(dbPassword)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(hmacPassword), mac2)
}

func sendPasswordResetEmail(email, token string) {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD") // App Password (not actual Gmail password)

	to := []string{email}
	message := []byte(
		"Subject: Reset your password\r\n" +
			"MIME-version: 1.0;\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\";\r\n" +
			"\r\n" +
			"<html>" +
			"<body style='font-family: Arial, sans-serif;'>" +
			"<h2>Password Reset Request</h2>" +
			"<p>Hello, there</p>" +
			"<p>We received a request to reset your password on your TapGoPay account. Use the following One-Time Password (token) to continue:</p>" +
			"<div style='font-size: 24px; font-weight: bold; background:#f4f4f4; padding:10px; border-radius:5px; display:inline-block;'>" + token + "</div>" +
			"<p>This code will expire in <b>10 minutes</b>.</p>" +
			"<p>If you didn't request a password reset, you can safely ignore this email.</p>" +
			"<br>" +
			"<p>Best regards,<br>TapGoPay</p>" +
			"</body>" +
			"</html>",
	)

	auth := smtp.PlainAuth("", from, password, smtpHost)
	addr := net.JoinHostPort(smtpHost, smtpPort)
	err := smtp.SendMail(addr, auth, from, to, message)
	if err != nil {
		log.Printf("Error sending email; %v\n", err)
	}
}
