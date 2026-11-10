package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/shop/services/notification/internal/domain"
	"github.com/shop/services/notification/internal/interface/http/response"
)

var errNilParser = errors.New("token parser is nil")

type TokenParser interface {
	ParseAccessToken(ctx context.Context, raw string) (domain.Principal, error)
}

type principalContextKey struct{}

type AccessVerificationMiddleware struct {
	parser TokenParser
}

func NewAccessVerificationMiddleware(parser TokenParser) (*AccessVerificationMiddleware, error) {
	if parser == nil {
		return nil, errNilParser
	}
	return &AccessVerificationMiddleware{parser: parser}, nil
}

func (m *AccessVerificationMiddleware) VerifyAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeUnauthorized(w)
			return
		}

		principal, err := m.parser.ParseAccessToken(r.Context(), raw)
		if err != nil {
			writeUnauthorized(w)
			return
		}
		if !domain.CanReadNotifications(principal.Role) {
			response.ResponseJSON(w, http.StatusForbidden, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "FORBIDDEN",
					Message: "forbidden",
				},
			})
			return
		}

		ctx := context.WithValue(r.Context(), principalContextKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func PrincipalFromContext(ctx context.Context) (domain.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(domain.Principal)
	return principal, ok
}

func bearerToken(header string) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.Contains(token, " ") {
		return "", false
	}
	return token, true
}

func writeUnauthorized(w http.ResponseWriter) {
	response.ResponseJSON(w, http.StatusUnauthorized, &response.Response{
		Success: false,
		Errors: &response.APIError{
			Code:    "UNAUTHORIZED",
			Message: "unauthorized",
		},
	})
}
