package middleware

import (
	"context"
	"net/http"
	"strings"

	"authorization/internal/infrastructure/token"
	"authorization/internal/interface/http/response"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

type AccessDataExtractor interface {
	GetDataByAccessToken(ctx context.Context, accessToken string) (*token.CustomClaims, error)
}

type AccessVerificationMiddleware struct {
	extractor AccessDataExtractor
}

func NewAccessVerificationMiddleware(extractor AccessDataExtractor) *AccessVerificationMiddleware {
	return &AccessVerificationMiddleware{extractor: extractor}
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

		claims, err := m.extractor.GetDataByAccessToken(r.Context(), accessToken)

		if err != nil {
			response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "INVALID_TOKEN",
					Message: err.Error(),
				},
			})
			return
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
