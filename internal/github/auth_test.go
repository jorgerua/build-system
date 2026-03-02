package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func computeTestSig(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestValidateWebhookSignature(t *testing.T) {
	secret := "my-secret"
	body := []byte(`{"ref":"refs/heads/main"}`)
	validSig := computeTestSig(secret, body)

	tests := []struct {
		name      string
		signature string
		wantErr   bool
	}{
		{"valid", validSig, false},
		{"wrong signature", "sha256=deadbeef00000000000000000000000000000000000000000000000000000000", true},
		{"missing sha256 prefix", "deadbeef", true},
		{"empty", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWebhookSignature(secret, tc.signature, body)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateWebhookSignature(%q) error = %v, wantErr %v", tc.signature, err, tc.wantErr)
			}
		})
	}
}
