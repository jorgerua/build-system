# Test Utilities Package

This package provides reusable test utilities, mocks, and helpers for testing the OCI Build System.

## Overview

The `testutil` package contains:
- **Mocks**: Mock implementations of service interfaces for unit testing
- **Fixtures**: Helper functions for creating test data, temporary repositories, and webhook payloads
- **Assertions**: Custom assertion helpers for common test scenarios

## Mocks

### MockNATSClient

Mock implementation of the NATS client for testing message publishing and subscriptions.

**Usage:**

```go
import "github.com/oci-build-system/tests/testutil"

func TestMyHandler(t *testing.T) {
    client := testutil.NewMockNATSClient()
    
    // Use the mock client
    err := client.Publish("test.subject", []byte("data"))
    
    // Verify messages were published
    messages := client.GetPublishedMessagesBySubject("test.subject")
    assert.Equal(t, 1, len(messages))
}
```

**Features:**
- Track all published messages
- Simulate connection failures
- Custom publish/subscribe functions
- Query messages by subject

### MockGitService

Mock implementation of Git operations for testing repository synchronization.

**Usage:**

```go
service := testutil.NewMockGitService()

// Use default behavior
path, err := service.SyncRepository(ctx, repo, "abc123")

// Or customize behavior
service.SyncRepositoryFunc = func(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error) {
    return "/custom/path", nil
}
```

### MockNXService

Mock implementation of NX build service for testing build operations.

**Usage:**

```go
service := testutil.NewMockNXService()

result, err := service.Build(ctx, "/repo/path", config)
```

### MockImageService

Mock implementation of image build service for testing OCI image operations.

**Usage:**

```go
service := testutil.NewMockImageService()

result, err := service.BuildImage(ctx, config)
err = service.TagImage("sha256:abc123", []string{"v1.0.0", "latest"})
```

### MockCacheService

Mock implementation of cache management service for testing cache operations.

**Usage:**

```go
service := testutil.NewMockCacheService()

path := service.GetCachePath(shared.LanguageJava)
size, err := service.GetCacheSize(shared.LanguageGo)
err = service.CleanCache(shared.LanguageJava, 24*time.Hour)
```

## Fixtures

### CreateTempRepo

Creates a temporary repository with language-specific project files for testing.

**Usage:**

```go
import "github.com/oci-build-system/tests/testutil"

func TestBuildJavaProject(t *testing.T) {
    // Create a temporary Java repository
    repoPath := testutil.CreateTempRepo(t, "java")
    
    // Repository contains:
    // - pom.xml (Maven configuration)
    // - Dockerfile (Java-specific multi-stage build)
    // - src/main/java/com/example/Main.java (minimal Java source)
    
    // Use the repository for testing
    // ...
}
```

**Supported Languages:**
- `"java"` - Creates Maven project with pom.xml, Dockerfile, and Java source
- `"dotnet"` - Creates .NET project with .csproj, Dockerfile, and C# source
- `"go"` - Creates Go project with go.mod, go.sum, Dockerfile, and Go source
- Any other value - Creates generic repository with just a Dockerfile

**Features:**
- Automatically cleaned up after test (uses `t.TempDir()`)
- Contains realistic project structure for each language
- Includes language-specific Dockerfiles with multi-stage builds
- Minimal but valid source code that compiles

### LoadWebhookPayload

Creates a GitHub webhook payload for testing webhook handlers.

**Usage:**

```go
func TestWebhookHandler(t *testing.T) {
    // Create a webhook payload
    payload := testutil.LoadWebhookPayload(t, "my-repo")
    
    // Payload contains:
    // - ref: "refs/heads/main"
    // - after: commit hash
    // - repository: name, full_name, clone_url, owner
    // - head_commit: id, message, author
    
    // Use for testing
    handler.HandleWebhook(payload)
}
```

### LoadWebhookPayloadWithBranch

Creates a webhook payload with custom branch and commit hash.

**Usage:**

```go
func TestWebhookWithCustomBranch(t *testing.T) {
    payload := testutil.LoadWebhookPayloadWithBranch(
        t, 
        "my-repo", 
        "develop", 
        "abc123def456",
    )
    
    // Payload has custom branch and commit
    assert.Equal(t, "refs/heads/develop", payload["ref"])
    assert.Equal(t, "abc123def456", payload["after"])
}
```

### GenerateHMACSignature

Generates HMAC-SHA256 signature for webhook validation.

**Usage:**

```go
func TestWebhookSignature(t *testing.T) {
    payload := []byte(`{"ref":"refs/heads/main"}`)
    secret := "my-webhook-secret"
    
    // Generate signature
    signature := testutil.GenerateHMACSignature(payload, secret)
    
    // signature format: "sha256=<64-char-hex-string>"
    // Use in X-Hub-Signature-256 header
}
```

### GenerateHMACSignatureForPayload

Generates signature for a map payload (automatically marshals to JSON).

**Usage:**

```go
func TestWebhookWithSignature(t *testing.T) {
    payload := testutil.LoadWebhookPayload(t, "test-repo")
    secret := "webhook-secret"
    
    // Generate signature for the payload
    signature, err := testutil.GenerateHMACSignatureForPayload(payload, secret)
    require.NoError(t, err)
    
    // Use signature in request
    req.Header.Set("X-Hub-Signature-256", signature)
}
```

## Customizing Mock Behavior

All mocks support custom functions to override default behavior:

```go
mock := testutil.NewMockNATSClient()

// Simulate connection failure
mock.PublishFunc = func(subject string, data []byte) error {
    return fmt.Errorf("connection lost")
}

// Now Publish will return an error
err := mock.Publish("test", []byte("data"))
// err != nil
```

## Testing Best Practices

1. **Use mocks for external dependencies**: Mock NATS, Git, and other external services
2. **Test both success and failure cases**: Use custom functions to simulate errors
3. **Verify interactions**: Check that mocks were called with expected parameters
4. **Clean up between tests**: Use `ClearPublishedMessages()` or create new mocks

## Examples

### Testing Webhook Handler with Fixtures

```go
func TestWebhookHandler_ValidSignature(t *testing.T) {
    // Setup
    mockNATS := testutil.NewMockNATSClient()
    handler := NewWebhookHandler(mockNATS, logger)
    
    // Create webhook payload
    payload := testutil.LoadWebhookPayload(t, "test-repo")
    secret := "webhook-secret"
    
    // Generate valid signature
    signature, err := testutil.GenerateHMACSignatureForPayload(payload, secret)
    require.NoError(t, err)
    
    // Execute
    handler.HandleWebhook(payload, signature)
    
    // Verify
    messages := mockNATS.GetPublishedMessagesBySubject("build.jobs")
    assert.Equal(t, 1, len(messages))
}
```

### Testing Build with Temporary Repository

```go
func TestBuildService_JavaProject(t *testing.T) {
    // Setup
    repoPath := testutil.CreateTempRepo(t, "java")
    buildService := NewBuildService(logger)
    
    // Execute
    result, err := buildService.Build(ctx, repoPath)
    
    // Verify
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.FileExists(t, filepath.Join(repoPath, "target", "test-project-1.0.0.jar"))
}
```

### Testing Language Detection

```go
func TestDetectLanguage_MultipleLanguages(t *testing.T) {
    tests := []struct {
        name     string
        language string
        expected string
    }{
        {"Java project", "java", "java"},
        {".NET project", "dotnet", "dotnet"},
        {"Go project", "go", "go"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create temporary repository
            repoPath := testutil.CreateTempRepo(t, tt.language)
            
            // Detect language
            detected, err := DetectLanguage(repoPath)
            
            // Verify
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, detected)
        })
    }
}
```

### Testing a Handler with Mock NATS

```go
func TestWebhookHandler(t *testing.T) {
    // Setup
    mockNATS := testutil.NewMockNATSClient()
    handler := NewWebhookHandler(mockNATS, logger)
    
    // Execute
    handler.HandleWebhook(payload)
    
    // Verify
    messages := mockNATS.GetPublishedMessagesBySubject("build.jobs")
    assert.Equal(t, 1, len(messages))
}
```

### Testing with Multiple Mocks

```go
func TestOrchestrator(t *testing.T) {
    // Setup all mocks
    mockGit := testutil.NewMockGitService()
    mockNX := testutil.NewMockNXService()
    mockImage := testutil.NewMockImageService()
    mockCache := testutil.NewMockCacheService()
    
    orchestrator := NewOrchestrator(mockGit, mockNX, mockImage, mockCache, logger)
    
    // Execute
    result, err := orchestrator.ProcessBuild(ctx, job)
    
    // Verify
    assert.NoError(t, err)
    assert.True(t, result.Success)
}
```

### Simulating Failures

```go
func TestHandleGitFailure(t *testing.T) {
    mockGit := testutil.NewMockGitService()
    
    // Simulate git failure
    mockGit.SyncRepositoryFunc = func(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error) {
        return "", fmt.Errorf("git clone failed")
    }
    
    orchestrator := NewOrchestrator(mockGit, nil, nil, nil, logger)
    
    result, err := orchestrator.ProcessBuild(ctx, job)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "git clone failed")
}
```

## Running Tests

```bash
# Run all testutil tests
cd tests/testutil
go test -v

# Run with coverage
go test -v -cover

# Run specific test
go test -v -run TestMockNATSClient_Publish
```

## Contributing

When adding new mocks:

1. Follow the existing pattern (struct with function fields)
2. Provide a `New*` constructor function
3. Implement all interface methods
4. Add tests for the mock itself
5. Document usage in this README

## Dependencies

- `github.com/nats-io/nats.go` - NATS client library
- `github.com/oci-build-system/libs/shared` - Shared types and interfaces
