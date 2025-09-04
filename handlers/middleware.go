package handlers

import (
	"context"
	"encoding/base64"
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
			api.BadRequest(w, "Invalid Authorization header format")
			return
		}

		// We receive ANDROID_API_KEY as b64-encoded string
		b64EncodedData := fields[1]
		androidApiKey, err := base64.StdEncoding.DecodeString(b64EncodedData)
		if err != nil {
			api.BadRequest(w, "Invalid Authorization value. Expected base64-encoded string")
			return
		}

		if ANDROID_API_KEY != string(androidApiKey) {
			api.Unauthorized(w)
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
			api.Unauthorized2(w, "Access to this route requires user login")
			return
		}

		accessToken := cookie.Value

		var user database.User
		if !validToken(accessToken, &user) {
			api.Unauthorized2(w, "Access to this route requires user login")
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
