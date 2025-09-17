package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/api"
	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

const (
	LOGIN_COOKIE string = "LOGIN_COOKIE"
)

// Restricts access to a route unless client request has embedded
// ANDROID_API_KEY in their Authorization header
func RequireAndroidApiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		fields := strings.Split(authHeader, " ")
		if len(fields) != 2 {
			api.BadRequest(w, "Invalid Authorization header format", nil)
			return
		}

		// We receive ANDROID_API_KEY as b64-encoded string
		b64EncodedData := fields[1]
		androidApiKey, err := base64.StdEncoding.DecodeString(b64EncodedData)
		if err != nil {
			api.BadRequest(w, "Invalid Authorization value. Expected base64-encoded string", nil)
			return
		}

		if ANDROID_API_KEY != string(androidApiKey) {
			api.Unauthorized(w, "Invalid ANDROID_API_KEY in Authorization headers")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getAccessTokenFromCookies(r *http.Request) (string, error) {
	cookie, err := r.Cookie(LOGIN_COOKIE)
	if err != nil {
		return "", err
	}

	accessToken := cookie.Value
	return accessToken, nil
}

func getAccessTokenFromHeaders(r *http.Request) (string, error) {
	auth := r.Header.Get("AuthToken")
	fields := strings.Split(auth, " ")
	if len(fields) != 2 {
		return "", fmt.Errorf("expected AuthToken header to use format Bearer <token>")
	}

	accesToken := fields[1]
	return accesToken, nil
}

func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieToken, err := getAccessTokenFromCookies(r)
		headerToken, err2 := getAccessTokenFromHeaders(r)

		if err != nil && err2 != nil {
			api.Unauthorized(w, "Access to this resource requires user login")
			return
		}

		var user database.User

		if !validToken(cookieToken, &user) && !validToken(headerToken, &user) {
			api.Unauthorized(w, "Access to this resource requires user login")
			return
		}

		// Embed user into context
		newCtx := context.WithValue(r.Context(), USER_CTX_KEY, &user)
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieToken, err := getAccessTokenFromCookies(r)
		headerToken, err2 := getAccessTokenFromHeaders(r)

		if err != nil && err2 != nil {
			api.Unauthorized(w, "Access to this resource requires user login")
			return
		}

		var user database.User

		if !validToken(cookieToken, &user) && !validToken(headerToken, &user) {
			api.Unauthorized(w, "Access to this resource requires user login")
			return
		}

		// Check users role
		if user.Role != "admin" {
			api.Unauthorized(w, "You are not authorized to access this resource")
			return
		}

		// Embed user into context
		newCtx := context.WithValue(r.Context(), USER_CTX_KEY, &user)
		next.ServeHTTP(w, r.WithContext(newCtx))
	})
}

func getAuthUser(r *http.Request) (*database.User, bool) {
	value := r.Context().Value(USER_CTX_KEY)
	user, ok := value.(*database.User)
	return user, ok
}
