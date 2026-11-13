package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shop/services/notification/internal/domain"
	"github.com/shop/services/notification/internal/interface/http/response"
)

type fakeParser struct {
	principal domain.Principal
	err       error
	calls     int
}

func (f *fakeParser) ParseAccessToken(ctx context.Context, raw string) (domain.Principal, error) {
	f.calls++
	if raw != "token" {
		return domain.Principal{}, errors.New("unexpected token")
	}
	return f.principal, f.err
}

func TestVerifyAccessAllowsAnalystAndAdmin(t *testing.T) {
	for _, role := range []string{"2", "4", "5", "6"} {
		t.Run(role, func(t *testing.T) {
			parser := &fakeParser{principal: domain.Principal{UserID: "user-1", Role: role}}
			mw := newMiddleware(t, parser)

			var got domain.Principal
			var found bool
			handler := mw.VerifyAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, found = PrincipalFromContext(r.Context())
				w.WriteHeader(http.StatusNoContent)
			}))

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
			req.Header.Set("Authorization", "Bearer token")
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
			}
			if !found || got.UserID != "user-1" || got.Role != role {
				t.Fatalf("principal = %+v found = %v", got, found)
			}
		})
	}
}

func TestVerifyAccessUnauthorized(t *testing.T) {
	parser := &fakeParser{err: errors.New("bad signature")}
	mw := newMiddleware(t, parser)
	handler := mw.VerifyAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next was called")
	}))

	tests := []struct {
		name   string
		header string
		calls  int
	}{
		{name: "missing", calls: 0},
		{name: "empty bearer", header: "Bearer   ", calls: 0},
		{name: "raw token", header: "token", calls: 0},
		{name: "invalid token", header: "Bearer token", calls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.calls = 0
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			handler.ServeHTTP(rec, req)

			if parser.calls != tt.calls {
				t.Fatalf("parser calls = %d", parser.calls)
			}
			assertError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
		})
	}
}

func TestVerifyAccessForbidsClient(t *testing.T) {
	parser := &fakeParser{principal: domain.Principal{UserID: "user-1", Role: "1"}}
	mw := newMiddleware(t, parser)
	handler := mw.VerifyAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next was called")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.Header.Set("Authorization", "Bearer token")
	handler.ServeHTTP(rec, req)

	assertError(t, rec, http.StatusForbidden, "FORBIDDEN")
}

func TestNewAccessVerificationMiddlewareRejectsNilParser(t *testing.T) {
	if _, err := NewAccessVerificationMiddleware(nil); err == nil {
		t.Fatal("expected error")
	}
}

func newMiddleware(t *testing.T, parser TokenParser) *AccessVerificationMiddleware {
	t.Helper()
	mw, err := NewAccessVerificationMiddleware(parser)
	if err != nil {
		t.Fatal(err)
	}
	return mw
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Success || body.Errors == nil || body.Errors.Code != code {
		t.Fatalf("body = %+v", body)
	}
}
