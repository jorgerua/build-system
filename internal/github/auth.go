package github

import (
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"go.uber.org/fx"
)

// Client handles GitHub App authentication and webhook validation.
type Client struct {
	appID      int64
	privateKey *rsa.PrivateKey
	httpClient *http.Client
}

// NewClient creates a GitHub App client from config.
func NewClient(cfg *config.Config) (*Client, error) {
	keyBytes, err := os.ReadFile(cfg.GitHub.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read github private key: %w", err)
	}
	key, err := parseRSAPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse github private key: %w", err)
	}
	return &Client{
		appID:      cfg.GitHub.AppID,
		privateKey: key,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// GenerateJWT creates a short-lived GitHub App JWT (valid 10 minutes).
func (c *Client) GenerateJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Add(-60 * time.Second).Unix(), // allow clock skew
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": c.appID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

// installationTokenResponse is the GitHub API response for installation tokens.
type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GenerateInstallationToken creates a fresh installation access token.
func (c *Client) GenerateInstallationToken(ctx context.Context, installationID int64) (string, error) {
	jwtToken, err := c.GenerateJWT()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status from github: %d", resp.StatusCode)
	}

	var result installationTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	return result.Token, nil
}

// ValidateWebhookSignature verifies the X-Hub-Signature-256 header.
func ValidateWebhookSignature(secret, signature string, body []byte) error {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return fmt.Errorf("invalid signature format")
	}
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil {
		return fmt.Errorf("decode signature hex: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)
	if !hmac.Equal(expected, sigBytes) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse rsa key (pkcs1: %v, pkcs8: %v)", err, err2)
		}
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		return rsaKey, nil
	}
	return key, nil
}

// Module provides *Client via fx.
var Module = fx.Module("github",
	fx.Provide(NewClient),
)
