package cmd

import (
	"encoding/json"
	"fmt"
)

// JSONEvent represents a JSON output event for machine-readable output.
type JSONEvent struct {
	Event           string `json:"event"`
	VerificationURI string `json:"verification_uri,omitempty"`
	UserCode        string `json:"user_code,omitempty"`
	User            string `json:"user,omitempty"`
	Result          string `json:"result,omitempty"`
	Error           string `json:"error,omitempty"`
	ApprovalURL     string `json:"approval_url,omitempty"`
}

func emitJSON(evt JSONEvent) {
	b, _ := json.Marshal(evt)
	fmt.Println(string(b))
}
