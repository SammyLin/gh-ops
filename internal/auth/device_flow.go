package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	GitHubDeviceCodeURL = "https://github.com/login/device/code"
	GitHubTokenURL      = "https://github.com/login/oauth/access_token"
)

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenPollResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
}

func RequestDeviceCode(endpoint, clientID, scope string) (*DeviceCodeResponse, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.URL.RawQuery = url.Values{
		"client_id": {clientID},
		"scope":     {scope},
	}.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed with status %d", resp.StatusCode)
	}

	var result DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}
	return &result, nil
}

func PollForToken(endpoint, clientID, deviceCode string, interval int) (string, error) {
	if interval < 1 {
		interval = 5
	}

	for {
		time.Sleep(time.Duration(interval) * time.Second)

		req, err := http.NewRequest(http.MethodPost, endpoint, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.URL.RawQuery = url.Values{
			"client_id":   {clientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}.Encode()

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to poll for token: %w", err)
		}

		var result tokenPollResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			_ = resp.Body.Close()
			return "", fmt.Errorf("failed to decode token response: %w", err)
		}
		_ = resp.Body.Close()

		switch result.Error {
		case "":
			return result.AccessToken, nil
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5
			continue
		case "expired_token":
			return "", fmt.Errorf("device code expired, please try again")
		case "access_denied":
			return "", fmt.Errorf("authorization denied by user")
		default:
			return "", fmt.Errorf("unexpected error: %s", result.Error)
		}
	}
}
