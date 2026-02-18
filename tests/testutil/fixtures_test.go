package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTempRepo_Java(t *testing.T) {
	// Execute
	repoPath := CreateTempRepo(t, "java")

	// Verify
	assert.DirExists(t, repoPath)

	// Check pom.xml exists
	pomPath := filepath.Join(repoPath, "pom.xml")
	assert.FileExists(t, pomPath)

	// Check Dockerfile exists
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	assert.FileExists(t, dockerfilePath)

	// Check Java source structure
	mainClassPath := filepath.Join(repoPath, "src", "main", "java", "com", "example", "Main.java")
	assert.FileExists(t, mainClassPath)

	// Verify pom.xml content
	pomContent, err := os.ReadFile(pomPath)
	require.NoError(t, err)
	assert.Contains(t, string(pomContent), "<artifactId>test-project</artifactId>")
	assert.Contains(t, string(pomContent), "<groupId>com.example</groupId>")
}

func TestCreateTempRepo_DotNet(t *testing.T) {
	// Execute
	repoPath := CreateTempRepo(t, "dotnet")

	// Verify
	assert.DirExists(t, repoPath)

	// Check .csproj exists
	csprojPath := filepath.Join(repoPath, "test-project.csproj")
	assert.FileExists(t, csprojPath)

	// Check Dockerfile exists
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	assert.FileExists(t, dockerfilePath)

	// Check C# source
	programPath := filepath.Join(repoPath, "Program.cs")
	assert.FileExists(t, programPath)

	// Verify .csproj content
	csprojContent, err := os.ReadFile(csprojPath)
	require.NoError(t, err)
	assert.Contains(t, string(csprojContent), "<TargetFramework>net8.0</TargetFramework>")
}

func TestCreateTempRepo_Go(t *testing.T) {
	// Execute
	repoPath := CreateTempRepo(t, "go")

	// Verify
	assert.DirExists(t, repoPath)

	// Check go.mod exists
	goModPath := filepath.Join(repoPath, "go.mod")
	assert.FileExists(t, goModPath)

	// Check go.sum exists
	goSumPath := filepath.Join(repoPath, "go.sum")
	assert.FileExists(t, goSumPath)

	// Check Dockerfile exists
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	assert.FileExists(t, dockerfilePath)

	// Check Go source
	mainPath := filepath.Join(repoPath, "main.go")
	assert.FileExists(t, mainPath)

	// Verify go.mod content
	goModContent, err := os.ReadFile(goModPath)
	require.NoError(t, err)
	assert.Contains(t, string(goModContent), "module github.com/example/test-project")
	assert.Contains(t, string(goModContent), "go 1.21")
}

func TestCreateTempRepo_Generic(t *testing.T) {
	// Execute
	repoPath := CreateTempRepo(t, "unknown")

	// Verify
	assert.DirExists(t, repoPath)

	// Check only Dockerfile exists for generic repo
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	assert.FileExists(t, dockerfilePath)

	// Verify no language-specific files
	assert.NoFileExists(t, filepath.Join(repoPath, "pom.xml"))
	assert.NoFileExists(t, filepath.Join(repoPath, "go.mod"))
	assert.NoFileExists(t, filepath.Join(repoPath, "test-project.csproj"))
}

func TestLoadWebhookPayload(t *testing.T) {
	// Execute
	payload := LoadWebhookPayload(t, "test-repo")

	// Verify
	assert.NotNil(t, payload)
	assert.Equal(t, "refs/heads/main", payload["ref"])
	assert.Equal(t, "abc123def456789012345678901234567890abcd", payload["after"])

	// Verify repository info
	repo, ok := payload["repository"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-repo", repo["name"])
	assert.Equal(t, "test-owner/test-repo", repo["full_name"])
	assert.Equal(t, "https://github.com/test-owner/test-repo.git", repo["clone_url"])

	// Verify owner
	owner, ok := repo["owner"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-owner", owner["login"])

	// Verify head commit
	headCommit, ok := payload["head_commit"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "abc123def456789012345678901234567890abcd", headCommit["id"])
	assert.Equal(t, "Test commit message", headCommit["message"])

	// Verify author
	author, ok := headCommit["author"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Test Author", author["name"])
	assert.Equal(t, "test@example.com", author["email"])
}

func TestLoadWebhookPayloadWithBranch(t *testing.T) {
	// Execute
	payload := LoadWebhookPayloadWithBranch(t, "my-repo", "develop", "def456abc123")

	// Verify
	assert.NotNil(t, payload)
	assert.Equal(t, "refs/heads/develop", payload["ref"])
	assert.Equal(t, "def456abc123", payload["after"])

	// Verify repository
	repo, ok := payload["repository"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "my-repo", repo["name"])

	// Verify head commit
	headCommit, ok := payload["head_commit"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "def456abc123", headCommit["id"])
	assert.Contains(t, headCommit["message"], "develop")
}

func TestGenerateHMACSignature(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		secret   string
		expected string
	}{
		{
			name:     "simple payload",
			payload:  []byte("test payload"),
			secret:   "secret123",
			expected: "sha256:8c5e5b5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e",
		},
		{
			name:     "empty payload",
			payload:  []byte(""),
			secret:   "secret123",
			expected: "sha256:",
		},
		{
			name:     "json payload",
			payload:  []byte(`{"key":"value"}`),
			secret:   "my-secret",
			expected: "sha256:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			signature := GenerateHMACSignature(tt.payload, tt.secret)

			// Verify
			assert.NotEmpty(t, signature)
			assert.Contains(t, signature, "sha256=")

			// Verify signature format (sha256= followed by 64 hex characters)
			assert.Len(t, signature, 71) // "sha256=" (7) + 64 hex chars
		})
	}
}

func TestGenerateHMACSignature_Consistency(t *testing.T) {
	payload := []byte("consistent test")
	secret := "test-secret"

	// Generate signature twice
	sig1 := GenerateHMACSignature(payload, secret)
	sig2 := GenerateHMACSignature(payload, secret)

	// Verify they are identical
	assert.Equal(t, sig1, sig2, "Signatures should be consistent for same input")
}

func TestGenerateHMACSignature_DifferentSecrets(t *testing.T) {
	payload := []byte("test payload")

	// Generate signatures with different secrets
	sig1 := GenerateHMACSignature(payload, "secret1")
	sig2 := GenerateHMACSignature(payload, "secret2")

	// Verify they are different
	assert.NotEqual(t, sig1, sig2, "Signatures should differ for different secrets")
}

func TestGenerateHMACSignatureForPayload(t *testing.T) {
	// Setup
	payload := map[string]interface{}{
		"ref":   "refs/heads/main",
		"after": "abc123",
		"repository": map[string]interface{}{
			"name": "test-repo",
		},
	}
	secret := "webhook-secret"

	// Execute
	signature, err := GenerateHMACSignatureForPayload(payload, secret)

	// Verify
	require.NoError(t, err)
	assert.NotEmpty(t, signature)
	assert.Contains(t, signature, "sha256=")

	// Verify we can manually recreate the same signature
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)
	expectedSignature := GenerateHMACSignature(payloadBytes, secret)
	assert.Equal(t, expectedSignature, signature)
}

func TestGenerateHMACSignatureForPayload_InvalidPayload(t *testing.T) {
	// Setup - create a payload that can't be marshaled
	payload := map[string]interface{}{
		"invalid": make(chan int), // channels can't be marshaled to JSON
	}
	secret := "test-secret"

	// Execute
	signature, err := GenerateHMACSignatureForPayload(payload, secret)

	// Verify
	assert.Error(t, err)
	assert.Empty(t, signature)
}

func TestFixtureConstants(t *testing.T) {
	// Verify all constants are non-empty
	assert.NotEmpty(t, javaPomXML, "javaPomXML should not be empty")
	assert.NotEmpty(t, dotnetCsproj, "dotnetCsproj should not be empty")
	assert.NotEmpty(t, goMod, "goMod should not be empty")
	assert.NotEmpty(t, goSum, "goSum should not be empty")
	assert.NotEmpty(t, dockerfile, "dockerfile should not be empty")
	assert.NotEmpty(t, javaDockerfile, "javaDockerfile should not be empty")
	assert.NotEmpty(t, dotnetDockerfile, "dotnetDockerfile should not be empty")
	assert.NotEmpty(t, goDockerfile, "goDockerfile should not be empty")

	// Verify constants contain expected content
	assert.Contains(t, javaPomXML, "<project")
	assert.Contains(t, javaPomXML, "<artifactId>test-project</artifactId>")

	assert.Contains(t, dotnetCsproj, "<Project Sdk=")
	assert.Contains(t, dotnetCsproj, "<TargetFramework>net8.0</TargetFramework>")

	assert.Contains(t, goMod, "module github.com/example/test-project")
	assert.Contains(t, goMod, "go 1.21")

	assert.Contains(t, dockerfile, "FROM alpine:latest")
	assert.Contains(t, javaDockerfile, "FROM maven:")
	assert.Contains(t, dotnetDockerfile, "FROM mcr.microsoft.com/dotnet/sdk:")
	assert.Contains(t, goDockerfile, "FROM golang:")
}

func TestSourceCodeTemplates(t *testing.T) {
	// Verify source code templates are non-empty
	assert.NotEmpty(t, javaMainClass, "javaMainClass should not be empty")
	assert.NotEmpty(t, dotnetProgram, "dotnetProgram should not be empty")
	assert.NotEmpty(t, goMain, "goMain should not be empty")

	// Verify templates contain expected content
	assert.Contains(t, javaMainClass, "public class Main")
	assert.Contains(t, javaMainClass, "public static void main")

	assert.Contains(t, dotnetProgram, "class Program")
	assert.Contains(t, dotnetProgram, "static void Main")

	assert.Contains(t, goMain, "package main")
	assert.Contains(t, goMain, "func main()")
}
