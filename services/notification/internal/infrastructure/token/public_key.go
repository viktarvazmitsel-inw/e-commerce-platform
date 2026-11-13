package token

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

const (
	defaultFetchAttempts = 5
	defaultFetchDelay    = time.Second
	fetchTimeout         = 2 * time.Second
)

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func FetchPublicKey(ctx context.Context, endpoint string, attempts int, delay time.Duration, client *http.Client) (*rsa.PublicKey, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("public key url is empty")
	}
	if attempts <= 0 {
		attempts = defaultFetchAttempts
	}
	if delay < 0 {
		delay = defaultFetchDelay
	}
	if client == nil {
		client = &http.Client{Timeout: fetchTimeout}
	}

	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		key, err := fetchPublicKeyOnce(ctx, client, endpoint)
		if err == nil {
			return key, nil
		}
		last = err
		if attempt == attempts-1 {
			break
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, fmt.Errorf("fetch public key: %w", last)
}

func fetchPublicKeyOnce(ctx context.Context, client *http.Client, endpoint string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build public key request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read public key response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("public key response status %d", resp.StatusCode)
	}

	return PublicKeyFromJWK(body)
}

func PublicKeyFromJWK(body []byte) (*rsa.PublicKey, error) {
	var key jwk
	if err := json.Unmarshal(body, &key); err != nil {
		return nil, fmt.Errorf("decode jwk: %w", err)
	}
	if key.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported jwk kty %q", key.Kty)
	}
	if key.Alg != "" && key.Alg != "RS256" {
		return nil, fmt.Errorf("unsupported jwk alg %q", key.Alg)
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil || len(nBytes) == 0 {
		return nil, fmt.Errorf("decode jwk modulus")
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil || len(eBytes) == 0 {
		return nil, fmt.Errorf("decode jwk exponent")
	}

	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 | int(b)
	}
	if exponent < 2 {
		return nil, fmt.Errorf("invalid jwk exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: exponent,
	}, nil
}
