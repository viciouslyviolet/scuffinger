// Package auth implements the GitHub OAuth Device Flow and manages
// persisted credentials via the vault package.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"scuffinger/internal/vault"
)

const (
	// Vault keys
	vaultKeyToken = "github_token"
	vaultKeyUser  = "github_user"

	// GitHub endpoints
	deviceCodeURL  = "https://github.com/login/device/code"
	accessTokenURL = "https://github.com/login/oauth/access_token"
)

// DeviceCodeResponse is the initial response from GitHub's device flow.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// AccessTokenResponse is the response when polling for the access token.
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// RequestDeviceCode initiates the device flow with GitHub.
func RequestDeviceCode(clientID string, scopes []string) (*DeviceCodeResponse, error) {
	data := url.Values{
		"client_id": {clientID},
		"scope":     {strings.Join(scopes, " ")},
	}

	req, err := http.NewRequest(http.MethodPost, deviceCodeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request returned %d: %s", resp.StatusCode, string(body))
	}

	var dcr DeviceCodeResponse
	if err := json.Unmarshal(body, &dcr); err != nil {
		return nil, fmt.Errorf("decode device code response: %w", err)
	}
	return &dcr, nil
}

// PollForToken polls GitHub until the user enters the code or the flow expires.
func PollForToken(clientID, deviceCode string, interval, expiresIn int) (*AccessTokenResponse, error) {
	pollInterval := time.Duration(interval) * time.Second
	if pollInterval < 5*time.Second {
		pollInterval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		atr, err := requestAccessToken(clientID, deviceCode)
		if err != nil {
			return nil, err
		}

		switch atr.Error {
		case "":
			// Success
			return atr, nil
		case "authorization_pending":
			// User hasn't entered the code yet — keep polling
			continue
		case "slow_down":
			// GitHub is asking us to slow down
			pollInterval += 5 * time.Second
			continue
		case "expired_token":
			return nil, errors.New("device code expired — please try again")
		case "access_denied":
			return nil, errors.New("authorisation was denied by the user")
		default:
			return nil, fmt.Errorf("oauth error: %s — %s", atr.Error, atr.ErrorDesc)
		}
	}

	return nil, errors.New("device code expired — please try again")
}

func requestAccessToken(clientID, deviceCode string) (*AccessTokenResponse, error) {
	data := url.Values{
		"client_id":   {clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	req, err := http.NewRequest(http.MethodPost, accessTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("access token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var atr AccessTokenResponse
	if err := json.Unmarshal(body, &atr); err != nil {
		return nil, fmt.Errorf("decode access token response: %w", err)
	}
	return &atr, nil
}

// ── Credential management ────────────────────────────────────────────────────

// SaveToken stores the OAuth token securely in the system vault.
func SaveToken(store vault.Store, token string) error {
	return store.Set(vaultKeyToken, token)
}

// LoadToken retrieves a previously saved OAuth token from the vault.
// Returns vault.ErrNotFound if no token is stored.
func LoadToken(store vault.Store) (string, error) {
	return store.Get(vaultKeyToken)
}

// SaveUser stores the authenticated GitHub username.
func SaveUser(store vault.Store, username string) error {
	return store.Set(vaultKeyUser, username)
}

// LoadUser retrieves the stored GitHub username.
func LoadUser(store vault.Store) (string, error) {
	return store.Get(vaultKeyUser)
}

// ClearCredentials removes all stored GitHub credentials.
func ClearCredentials(store vault.Store) {
	_ = store.Delete(vaultKeyToken)
	_ = store.Delete(vaultKeyUser)
}

// HasCredentials returns true if a token is stored in the vault.
func HasCredentials(store vault.Store) bool {
	_, err := store.Get(vaultKeyToken)
	return err == nil
}
