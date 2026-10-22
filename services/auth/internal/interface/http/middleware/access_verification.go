package middleware

import (
	"authorization/internal/interface/http/response"
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

type AccessVerificationMiddleware struct {
	publicKey *rsa.PublicKey
}

func NewAccessVerificationMiddleware(publicKey *rsa.PublicKey) *AccessVerificationMiddleware {
	return &AccessVerificationMiddleware{publicKey: publicKey}
}

type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

func (m *AccessVerificationMiddleware) VerifyAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "UNAUTHORIZED",
					Message: "unauthorized",
				},
			})
			return
		}

		accessToken := header
		if scheme, token, ok := strings.Cut(header, " "); ok && strings.EqualFold(scheme, "Bearer") {
			accessToken = strings.TrimSpace(token)
		}
		if accessToken == "" {
			response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "UNAUTHORIZED",
					Message: "unauthorized",
				},
			})
			return
		}

		token, err := jwt.ParseWithClaims(accessToken, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("wrong token sign algoritm: %v", token.Header["alg"])
			}
			return m.publicKey, nil
		})

		if err != nil || !token.Valid {
			response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "INVALID_TOKEN",
					Message: "invalid or expired token",
				},
			})
			return
		}

		claims, ok := token.Claims.(*CustomClaims)
		if !ok {
			response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "INVALID_TOKEN",
					Message: "invalid or expired token",
				},
			})
			return
		}

		if claims.Type != "access" {
			response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "INVALID_TOKEN_TYPE",
					Message: "token is not an access token",
				},
			})
			return
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
