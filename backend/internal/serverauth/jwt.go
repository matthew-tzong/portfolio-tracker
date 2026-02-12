package serverauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// JWTAuth is middleware that validates a Supabase JWT (ES256) from the Authorization header.
// It uses Supabase's JWKS discovery URL to fetch the public key and validate the token.
// It expects: Authorization: Bearer <access_token> and extracts the user ID (sub claim) from the token.
// It adds the request context with the user ID; if the token is valid it calls the next handler, else returns 401.
func JWTAuth(next http.Handler) http.Handler {
	supabaseURL := strings.TrimSuffix(os.Getenv("SUPABASE_URL"), "/")
	allowedUserEmail := os.Getenv("ALLOWED_USER_EMAIL")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := extractBearerToken(r.Header.Get("Authorization"))

		// if the token is missing or invalid, return 401
		if err != nil || tokenString == "" {
			http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
			return
		}

		// Ensure required configuration is present.
		if supabaseURL == "" || allowedUserEmail == "" {
			http.Error(w, "server auth not configured", http.StatusInternalServerError)
			return
		}

		jwksURL := supabaseURL + "/auth/v1/.well-known/jwks.json"

		// Parse and validate the token using an ES256 key from JWKS.
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodES256.Alg() {
				return nil, errors.New("unexpected signing method, expected ES256")
			}
			keyID, _ := t.Header["kid"].(string)
			publicKey, err := fetchES256KeyFromJWKS(jwksURL, keyID)
			if err != nil {
				return nil, err
			}
			return publicKey, nil
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
		if email == "" || !strings.EqualFold(email, allowedUserEmail) {
			http.Error(w, "unauthorized user", http.StatusUnauthorized)
			return
		}

		// add the user ID to the request context
		ctx := context.WithValue(r.Context(), userIDContextKey, sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Fetches JWKS and returns the ES256 public key matching the given keyID
func fetchES256KeyFromJWKS(jwksURL string, keyID string) (*ecdsa.PublicKey, error) {
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch JWKS")
	}

	var jwks struct {
		Keys []struct {
			KeyID   string `json:"kid"`
			KeyType string `json:"kty"`
			Curve   string `json:"crv"`
			XCoord  string `json:"x"`
			YCoord  string `json:"y"`
		} `json:"keys"`
	}

	err = json.NewDecoder(resp.Body).Decode(&jwks)
	if err != nil {
		return nil, err
	}

	for _, key := range jwks.Keys {
		if key.KeyType != "EC" || key.Curve != "P-256" {
			continue
		}
		if keyID != "" && key.KeyID != keyID {
			continue
		}

		xBytes, err := base64.RawURLEncoding.DecodeString(key.XCoord)
		if err != nil {
			continue
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(key.YCoord)
		if err != nil {
			continue
		}

		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)

		return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, nil
	}

	return nil, errors.New("no suitable ES256 key found in JWKS")
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
