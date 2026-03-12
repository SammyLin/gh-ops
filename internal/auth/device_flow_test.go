package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("client_id") != "test-client-id" {
			t.Fatalf("expected client_id=test-client-id, got %s", r.URL.Query().Get("client_id"))
		}
		if r.URL.Query().Get("scope") != "repo" {
			t.Fatalf("expected scope=repo, got %s", r.URL.Query().Get("scope"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode:      "device-123",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        5,
		})
	}))
	defer server.Close()

	resp, err := RequestDeviceCode(server.URL, "test-client-id", "repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserCode != "ABCD-1234" {
		t.Fatalf("expected user code ABCD-1234, got %s", resp.UserCode)
	}
	if resp.DeviceCode != "device-123" {
		t.Fatalf("expected device code device-123, got %s", resp.DeviceCode)
	}
}

func TestPollForToken_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount < 2 {
			json.NewEncoder(w).Encode(tokenPollResponse{
				Error: "authorization_pending",
			})
			return
		}
		json.NewEncoder(w).Encode(tokenPollResponse{
			AccessToken: "gho_test_token_123",
			TokenType:   "bearer",
		})
	}))
	defer server.Close()

	token, err := PollForToken(server.URL, "test-client-id", "device-123", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "gho_test_token_123" {
		t.Fatalf("expected gho_test_token_123, got %s", token)
	}
}
