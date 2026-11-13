package token

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseAccessToken(t *testing.T) {
	privateKey := testKey(t)
	parser := testParser(t, &privateKey.PublicKey)

	signed := signToken(t, privateKey, CustomClaims{
		UserID: "user-1",
		Role:   "4",
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})

	principal, err := parser.ParseAccessToken(context.Background(), signed)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if principal.UserID != "user-1" || principal.Role != "4" {
		t.Fatalf("principal = %+v", principal)
	}
}

func TestParseAccessTokenRejectsRefreshExpiredAndWrongKey(t *testing.T) {
	privateKey := testKey(t)
	parser := testParser(t, &privateKey.PublicKey)

	refresh := signToken(t, privateKey, CustomClaims{
		UserID: "user-1",
		Role:   "4",
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	if _, err := parser.ParseAccessToken(context.Background(), refresh); err == nil {
		t.Fatal("refresh token was accepted")
	}

	expired := signToken(t, privateKey, CustomClaims{
		UserID: "user-1",
		Role:   "4",
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	if _, err := parser.ParseAccessToken(context.Background(), expired); err == nil {
		t.Fatal("expired token was accepted")
	}

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	foreign := signToken(t, otherKey, CustomClaims{
		UserID: "user-1",
		Role:   "4",
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	if _, err := parser.ParseAccessToken(context.Background(), foreign); err == nil {
		t.Fatal("token signed by another key was accepted")
	}
}

func TestParseAccessTokenRejectsHMAC(t *testing.T) {
	privateKey := testKey(t)
	parser := testParser(t, &privateKey.PublicKey)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		UserID: "user-1",
		Role:   "4",
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseAccessToken(context.Background(), signed); err == nil {
		t.Fatal("hmac token was accepted")
	}
}

func testParser(t *testing.T, key *rsa.PublicKey) *Parser {
	t.Helper()
	parser, err := NewParser(key)
	if err != nil {
		t.Fatal(err)
	}
	return parser
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims CustomClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}
