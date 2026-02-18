package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
)

// Feature: jorgerua/build-system, Property 1: Validação de assinatura de webhook
// Para qualquer requisição de webhook recebida, se a assinatura HMAC-SHA256 não corresponder
// ao secret configurado, então o sistema deve retornar HTTP 401 e não processar o webhook.
// Valida: Requisitos 1.1, 10.4
func TestProperty_WebhookSignatureValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	properties := gopter.NewProperties(nil)

	properties.Property("invalid signature returns 401", prop.ForAll(
		func(payload []byte, wrongSecret string, correctSecret string) bool {
			// Garantir que os secrets sejam diferentes
			if wrongSecret == correctSecret {
				return true // Skip este caso
			}

			// Setup
			logger := zap.NewNop()
			mockNATS := new(MockNATSClient)
			handler := NewWebhookHandler(mockNATS, logger, correctSecret)

			// Calcular assinatura com secret errado
			mac := hmac.New(sha256.New, []byte(wrongSecret))
			mac.Write(payload)
			wrongSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			// Criar requisição HTTP
			router := gin.New()
			router.POST("/webhook", handler.Handle)

			req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Hub-Signature-256", wrongSignature)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verificar que retorna 401
			if w.Code != http.StatusUnauthorized {
				t.Logf("Expected 401, got %d", w.Code)
				return false
			}

			// Verificar que NATS não foi chamado
			mockNATS.AssertNotCalled(t, "Publish")

			return true
		},
		gen.SliceOfN(100, gen.UInt8()), // Payload aleatório de 100 bytes
		gen.AnyString(),                // Secret errado
		gen.AnyString(),                // Secret correto
	))

	properties.Property("valid signature is accepted", prop.ForAll(
		func(payload []byte, secret string) bool {
			// Skip secrets vazios
			if secret == "" {
				return true
			}

			// Setup
			logger := zap.NewNop()
			mockNATS := new(MockNATSClient)
			handler := NewWebhookHandler(mockNATS, logger, secret)

			// Calcular assinatura correta
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(payload)
			correctSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			// Criar requisição HTTP
			router := gin.New()
			router.POST("/webhook", handler.Handle)

			req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Hub-Signature-256", correctSignature)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verificar que NÃO retorna 401 (pode retornar 400 por JSON inválido, mas não 401)
			if w.Code == http.StatusUnauthorized {
				t.Logf("Valid signature returned 401")
				return false
			}

			return true
		},
		gen.SliceOfN(100, gen.UInt8()), // Payload aleatório de 100 bytes
		gen.AnyString(),                // Secret
	))

	properties.Property("missing signature returns 401", prop.ForAll(
		func(payload []byte, secret string) bool {
			// Setup
			logger := zap.NewNop()
			mockNATS := new(MockNATSClient)
			handler := NewWebhookHandler(mockNATS, logger, secret)

			// Criar requisição HTTP sem assinatura
			router := gin.New()
			router.POST("/webhook", handler.Handle)

			req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			// Não definir X-Hub-Signature-256

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verificar que retorna 401
			if w.Code != http.StatusUnauthorized {
				t.Logf("Missing signature returned %d instead of 401", w.Code)
				return false
			}

			// Verificar que NATS não foi chamado
			mockNATS.AssertNotCalled(t, "Publish")

			return true
		},
		gen.SliceOfN(100, gen.UInt8()), // Payload aleatório de 100 bytes
		gen.AnyString(),                // Secret
	))

	properties.Property("malformed signature returns 401", prop.ForAll(
		func(payload []byte, secret string, malformedSig string) bool {
			// Garantir que a assinatura não tenha o prefixo correto
			if len(malformedSig) > 7 && malformedSig[:7] == "sha256=" {
				return true // Skip este caso
			}

			// Setup
			logger := zap.NewNop()
			mockNATS := new(MockNATSClient)
			handler := NewWebhookHandler(mockNATS, logger, secret)

			// Criar requisição HTTP com assinatura malformada
			router := gin.New()
			router.POST("/webhook", handler.Handle)

			req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Hub-Signature-256", malformedSig)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verificar que retorna 401
			if w.Code != http.StatusUnauthorized {
				t.Logf("Malformed signature returned %d instead of 401", w.Code)
				return false
			}

			// Verificar que NATS não foi chamado
			mockNATS.AssertNotCalled(t, "Publish")

			return true
		},
		gen.SliceOfN(100, gen.UInt8()), // Payload aleatório de 100 bytes
		gen.AnyString(),                // Secret
		gen.AnyString(),                // Assinatura malformada
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: jorgerua/build-system, Property 2: Extração completa de informações de webhook
// Para qualquer webhook válido do GitHub, o sistema deve extrair corretamente todas as
// informações necessárias (repository URL, owner, name, commit hash, branch, author, message)
// e criar um BuildJob com esses dados.
// Valida: Requisitos 1.2
func TestProperty_WebhookInformationExtraction(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("extracts all required fields from valid payload", prop.ForAll(
		func(repoName string, repoOwner string, commitHash string, branch string, author string, message string) bool {
			// Skip valores vazios
			if repoName == "" || repoOwner == "" || commitHash == "" || branch == "" {
				return true
			}

			// Criar payload válido
			payload := &WebhookPayload{
				Ref:   "refs/heads/" + branch,
				After: commitHash,
				Repository: struct {
					Name     string `json:"name"`
					FullName string `json:"full_name"`
					CloneURL string `json:"clone_url"`
					Owner    struct {
						Login string `json:"login"`
					} `json:"owner"`
				}{
					Name:     repoName,
					FullName: repoOwner + "/" + repoName,
					CloneURL: "https://github.com/" + repoOwner + "/" + repoName + ".git",
					Owner: struct {
						Login string `json:"login"`
					}{
						Login: repoOwner,
					},
				},
				HeadCommit: struct {
					ID      string `json:"id"`
					Message string `json:"message"`
					Author  struct {
						Name  string `json:"name"`
						Email string `json:"email"`
					} `json:"author"`
				}{
					ID:      commitHash,
					Message: message,
					Author: struct {
						Name  string `json:"name"`
						Email string `json:"email"`
					}{
						Name:  author,
						Email: author + "@example.com",
					},
				},
			}

			// Setup handler
			logger := zap.NewNop()
			mockNATS := new(MockNATSClient)
			handler := NewWebhookHandler(mockNATS, logger, "test-secret")

			// Extrair BuildJob
			job, err := handler.extractBuildJob(payload)
			if err != nil {
				t.Logf("Failed to extract build job: %v", err)
				return false
			}

			// Verificar que todas as informações foram extraídas corretamente
			if job.Repository.Name != repoName {
				t.Logf("Repository name mismatch: expected %s, got %s", repoName, job.Repository.Name)
				return false
			}

			if job.Repository.Owner != repoOwner {
				t.Logf("Repository owner mismatch: expected %s, got %s", repoOwner, job.Repository.Owner)
				return false
			}

			if job.CommitHash != commitHash {
				t.Logf("Commit hash mismatch: expected %s, got %s", commitHash, job.CommitHash)
				return false
			}

			if job.Branch != branch {
				t.Logf("Branch mismatch: expected %s, got %s", branch, job.Branch)
				return false
			}

			if job.CommitAuthor != author {
				t.Logf("Commit author mismatch: expected %s, got %s", author, job.CommitAuthor)
				return false
			}

			if job.CommitMsg != message {
				t.Logf("Commit message mismatch: expected %s, got %s", message, job.CommitMsg)
				return false
			}

			// Verificar que o job ID foi gerado
			if job.ID == "" {
				t.Logf("Job ID is empty")
				return false
			}

			// Verificar que o status é pending
			if job.Status != "pending" {
				t.Logf("Job status should be pending, got %s", job.Status)
				return false
			}

			return true
		},
		gen.Identifier(), // repoName
		gen.Identifier(), // repoOwner
		gen.Identifier(), // commitHash
		gen.Identifier(), // branch
		gen.Identifier(), // author
		gen.AnyString(),  // message
	))

	properties.Property("handles branch ref format correctly", prop.ForAll(
		func(branch string) bool {
			// Skip branches vazias
			if branch == "" {
				return true
			}

			// Criar payload com ref no formato GitHub
			payload := &WebhookPayload{
				Ref:   "refs/heads/" + branch,
				After: "abc123",
				Repository: struct {
					Name     string `json:"name"`
					FullName string `json:"full_name"`
					CloneURL string `json:"clone_url"`
					Owner    struct {
						Login string `json:"login"`
					} `json:"owner"`
				}{
					Name:     "test-repo",
					FullName: "test/test-repo",
					CloneURL: "https://github.com/test/test-repo.git",
					Owner: struct {
						Login string `json:"login"`
					}{
						Login: "test",
					},
				},
				HeadCommit: struct {
					ID      string `json:"id"`
					Message string `json:"message"`
					Author  struct {
						Name  string `json:"name"`
						Email string `json:"email"`
					} `json:"author"`
				}{
					ID:      "abc123",
					Message: "test",
					Author: struct {
						Name  string `json:"name"`
						Email string `json:"email"`
					}{
						Name:  "test",
						Email: "test@example.com",
					},
				},
			}

			// Setup handler
			logger := zap.NewNop()
			mockNATS := new(MockNATSClient)
			handler := NewWebhookHandler(mockNATS, logger, "test-secret")

			// Extrair BuildJob
			job, err := handler.extractBuildJob(payload)
			if err != nil {
				t.Logf("Failed to extract build job: %v", err)
				return false
			}

			// Verificar que o branch foi extraído corretamente (sem o prefixo refs/heads/)
			if job.Branch != branch {
				t.Logf("Branch extraction failed: expected %s, got %s", branch, job.Branch)
				return false
			}

			return true
		},
		gen.Identifier(), // branch
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
