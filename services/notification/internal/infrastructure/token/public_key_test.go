package token

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchPublicKeyRetriesThenParsesJWK(t *testing.T) {
	privateKey := testKey(t)
	body := testJWK(t, &privateKey.PublicKey)

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		calls++
		if calls < 3 {
			http.Error(w, "starting", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer server.Close()

	key, err := FetchPublicKey(context.Background(), server.URL, 5, time.Millisecond, server.Client())
	if err != nil {
		t.Fatalf("FetchPublicKey() error = %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d", calls)
	}
	if key.N.Cmp(privateKey.PublicKey.N) != 0 || key.E != privateKey.PublicKey.E {
		t.Fatal("fetched key does not match")
	}
}

func TestFetchPublicKeyFailsAfterAttempts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := FetchPublicKey(context.Background(), server.URL, 2, time.Millisecond, server.Client())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchPublicKeyRejectsEmptyURL(t *testing.T) {
	_, err := FetchPublicKey(context.Background(), "  ", 1, 0, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublicKeyFromJWKRejectsWrongAlgorithm(t *testing.T) {
	privateKey := testKey(t)
	payload := map[string]string{
		"kty": "RSA",
		"alg": "RS512",
		"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PublicKeyFromJWK(body); err == nil {
		t.Fatal("expected error")
	}
}

func testKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func testJWK(t *testing.T, key *rsa.PublicKey) []byte {
	t.Helper()
	body, err := json.Marshal(jwk{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}
