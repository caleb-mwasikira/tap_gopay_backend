package handlers

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/database"
)

const (
	LOGIN_COOKIE string = "LOGIN_COOKIE"
)

func GenerateAndroidApiKey() string {
	if strings.TrimSpace(ANDROID_API_KEY) == "" {
		return ""
	}
	b64EncodedData := base64.StdEncoding.EncodeToString([]byte(ANDROID_API_KEY))
	return b64EncodedData
}

func RequireAndroidApiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fetch ANDROID_API_KEY from Authorization headers
		authHeader := r.Header.Get("Authorization")
		fields := strings.Split(authHeader, " ")
		if len(fields) != 2 {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"message": "Invalid Authorization header format"})
			return
		}

		// We receive ANDROID_API_KEY as b64-encoded string
		b64EncodedData := fields[1]
		androidApiKey, err := base64.StdEncoding.DecodeString(b64EncodedData)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, map[string]string{"message": "Invalid Authorization value. Expected base64-encoded string"})
			return
		}

		if ANDROID_API_KEY != string(androidApiKey) {
			jsonResponse(w, http.StatusUnauthorized, map[string]string{"message": "You are not authorized to access this resource"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fetch access token from user's cookies
		cookie, err := r.Cookie(LOGIN_COOKIE)
		if err != nil {
			jsonResponse(w, http.StatusUnauthorized, map[string]string{"message": "Access to this route requires user login"})
			return
		}

		accessToken := cookie.Value

		var user database.User
		if !validToken(accessToken, &user) {
			jsonResponse(w, http.StatusUnauthorized, map[string]string{"message": "Access to this route requires user login"})
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
