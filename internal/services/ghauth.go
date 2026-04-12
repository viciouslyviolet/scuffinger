package services

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ghAppTransport is an http.RoundTripper that authenticates as a GitHub App
// installation. It creates a JWT signed with the App's private key, exchanges
// it for a short-lived installation token, and caches the token until expiry.
//
// This replaces the third-party ghinstallation library with a minimal,
// self-contained implementation using only golang-jwt.
type ghAppTransport struct {
	appID          int64
	installationID int64
	key            *rsa.PrivateKey
	base           http.RoundTripper

	mu    sync.Mutex
	token string
	exp   time.Time
}

func newGHAppTransport(appID, installationID int64, key *rsa.PrivateKey) *ghAppTransport {
	return &ghAppTransport{
		appID:          appID,
		installationID: installationID,
		key:            key,
		base:           http.DefaultTransport,
	}
}

func (t *ghAppTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.installationToken(req.Context())
	if err != nil {
		return nil, err
	}
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "token "+tok)
	req2.Header.Set("Accept", "application/vnd.github+json")
	return t.base.RoundTrip(req2)
}

// installationToken returns a cached installation token or fetches a new one.
func (t *ghAppTransport) installationToken(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Return cached token if still valid (with 1-minute buffer).
	if t.token != "" && time.Now().Add(time.Minute).Before(t.exp) {
		return t.token, nil
	}

	jwtToken, err := t.createJWT()
	if err != nil {
		return "", fmt.Errorf("create JWT: %w", err)
	}

	tok, exp, err := t.exchangeJWT(ctx, jwtToken)
	if err != nil {
		return "", err
	}

	t.token = tok
	t.exp = exp
	return tok, nil
}

// createJWT builds a short-lived JWT signed with the App's RSA private key.
func (t *ghAppTransport) createJWT() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)), // clock drift
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),   // max 10 min
		Issuer:    fmt.Sprintf("%d", t.appID),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(t.key)
}

// exchangeJWT calls the GitHub API to exchange a JWT for an installation access token.
func (t *ghAppTransport) exchangeJWT(ctx context.Context, jwtToken string) (string, time.Time, error) {
	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", t.installationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("installation token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", time.Time{}, fmt.Errorf("installation token: HTTP %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", time.Time{}, fmt.Errorf("decode installation token: %w", err)
	}

	return result.Token, result.ExpiresAt, nil
}
