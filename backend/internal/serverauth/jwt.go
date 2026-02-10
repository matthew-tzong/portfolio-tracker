package serverauth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// JWTAuth is middleware that validates a Supabase JWT (HS256) from the Authorization header.
// It expects: Authorization: Bearer <access_token> and extracts the user ID (sub claim) from the token.
// It adds the request context with the user ID; if the token is valid it calls the next handler, else returns 401.
func JWTAuth(next http.Handler) http.Handler {
	secret := os.Getenv("SUPABASE_JWT_SECRET")
	allowedEmail := os.Getenv("ALLOWED_USER_EMAIL")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := extractBearerToken(r.Header.Get("Authorization"))

		// if the token is missing or invalid, return 401
		if err != nil || tokenString == "" {
			http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
			return
		}

		// if the secret is not set, return 500
		if secret == "" {
			http.Error(w, "server auth not configured", http.StatusInternalServerError)
			return
		}

		// parse the token
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			// Supabase JWTs are signed with HS256 by default
			_, ok := t.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})
		// if the token is invalid, return 401
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// get the claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		// get the subject (user ID) from the claims
		sub, _ := claims["sub"].(string)
		if sub == "" {
			http.Error(w, "missing subject", http.StatusUnauthorized)
			return
		}

		// Enforce that this token belongs to the single allowed user.
		email, _ := claims["email"].(string)
		if email == "" || !strings.EqualFold(email, allowedEmail) {
			http.Error(w, "unauthorized user", http.StatusUnauthorized)
			return
		}

		// add the user ID to the request context
		ctx := context.WithValue(r.Context(), userIDContextKey, sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext returns the authenticated user's ID from the request context.
// The second return value indicates whether a user ID was found in the context.
// This should only be called after JWTAuth middleware has processed the request.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDContextKey).(string)
	return v, ok
}

// extractBearerToken parses the Authorization header and extracts the Bearer token.
// It expects the format "Bearer <token>" and returns an error if the format is invalid.
func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("empty header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid header format")
	}
	return strings.TrimSpace(parts[1]), nil
}

