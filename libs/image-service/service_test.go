package imageservice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestNewImageService testa a criação de instância do serviço
func TestNewImageService(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("creates service successfully", func(t *testing.T) {
		svc := NewImageService(logger)
		if svc == nil {
			t.Error("NewImageService() returned nil")
		}
	})
}

// TestValidateConfig testa a validação de configuração
func TestValidateConfig(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewImageService(logger).(*imageService)

	tempDir := t.TempDir()

	tests := []struct {
		name    string
		config  ImageConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ImageConfig{
				ContextPath: tempDir,
				Tags:        []string{"myimage:latest"},
			},
			wantErr: false,
		},
		{
			name: "missing context path",
			config: ImageConfig{
				ContextPath: "",
				Tags:        []string{"myimage:latest"},
			},
			wantErr: true,
			errMsg:  "context_path is required",
		},
		{
			name: "non-existent context path",
			config: ImageConfig{
				ContextPath: "/nonexistent/path",
				Tags:        []string{"myimage:latest"},
			},
			wantErr: true,
			errMsg:  "context_path does not exist: /nonexistent/path",
		},
		{
			name: "missing tags",
			config: ImageConfig{
				ContextPath: tempDir,
				Tags:        []string{},
			},
			wantErr: true,
			errMsg:  "at least one tag is required",
		},
		{
			name: "nil tags",
			config: ImageConfig{
				ContextPath: tempDir,
				Tags:        nil,
			},
			wantErr: true,
			errMsg:  "at least one tag is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("validateConfig() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestLocateDockerfile testa a localização de Dockerfile
func TestLocateDockerfile(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewImageService(logger).(*imageService)

	tempDir := t.TempDir()

	t.Run("finds Dockerfile in root", func(t *testing.T) {
		dockerfilePath := filepath.Join(tempDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		result, err := svc.locateDockerfile(tempDir, "")
		if err != nil {
			t.Errorf("locateDockerfile() error = %v, want nil", err)
		}
		if result != dockerfilePath {
			t.Errorf("locateDockerfile() = %v, want %v", result, dockerfilePath)
		}
	})

	t.Run("finds lowercase dockerfile", func(t *testing.T) {
		tempDir2 := t.TempDir()
		dockerfilePath := filepath.Join(tempDir2, "dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
			t.Fatalf("Failed to create dockerfile: %v", err)
		}

		result, err := svc.locateDockerfile(tempDir2, "")
		if err != nil {
			t.Errorf("locateDockerfile() error = %v, want nil", err)
		}
		// On Windows, filesystem is case-insensitive, so we just check the file exists
		if _, statErr := os.Stat(result); statErr != nil {
			t.Errorf("locateDockerfile() returned path that doesn't exist: %v", result)
		}
	})

	t.Run("finds Dockerfile in docker subdirectory", func(t *testing.T) {
		tempDir3 := t.TempDir()
		dockerDir := filepath.Join(tempDir3, "docker")
		if err := os.MkdirAll(dockerDir, 0755); err != nil {
			t.Fatalf("Failed to create docker directory: %v", err)
		}
		dockerfilePath := filepath.Join(dockerDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		result, err := svc.locateDockerfile(tempDir3, "")
		if err != nil {
			t.Errorf("locateDockerfile() error = %v, want nil", err)
		}
		if result != dockerfilePath {
			t.Errorf("locateDockerfile() = %v, want %v", result, dockerfilePath)
		}
	})

	t.Run("finds Dockerfile in build subdirectory", func(t *testing.T) {
		tempDir4 := t.TempDir()
		buildDir := filepath.Join(tempDir4, "build")
		if err := os.MkdirAll(buildDir, 0755); err != nil {
			t.Fatalf("Failed to create build directory: %v", err)
		}
		dockerfilePath := filepath.Join(buildDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		result, err := svc.locateDockerfile(tempDir4, "")
		if err != nil {
			t.Errorf("locateDockerfile() error = %v, want nil", err)
		}
		if result != dockerfilePath {
			t.Errorf("locateDockerfile() = %v, want %v", result, dockerfilePath)
		}
	})

	t.Run("uses specified dockerfile path - relative", func(t *testing.T) {
		tempDir5 := t.TempDir()
		customDir := filepath.Join(tempDir5, "custom")
		if err := os.MkdirAll(customDir, 0755); err != nil {
			t.Fatalf("Failed to create custom directory: %v", err)
		}
		dockerfilePath := filepath.Join(customDir, "Dockerfile.prod")
		if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		result, err := svc.locateDockerfile(tempDir5, "custom/Dockerfile.prod")
		if err != nil {
			t.Errorf("locateDockerfile() error = %v, want nil", err)
		}
		if result != dockerfilePath {
			t.Errorf("locateDockerfile() = %v, want %v", result, dockerfilePath)
		}
	})

	t.Run("uses specified dockerfile path - absolute", func(t *testing.T) {
		tempDir6 := t.TempDir()
		dockerfilePath := filepath.Join(tempDir6, "Dockerfile.abs")
		if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		result, err := svc.locateDockerfile(tempDir6, dockerfilePath)
		if err != nil {
			t.Errorf("locateDockerfile() error = %v, want nil", err)
		}
		if result != dockerfilePath {
			t.Errorf("locateDockerfile() = %v, want %v", result, dockerfilePath)
		}
	})

	t.Run("error when Dockerfile not found", func(t *testing.T) {
		tempDir7 := t.TempDir()

		_, err := svc.locateDockerfile(tempDir7, "")
		if err == nil {
			t.Error("locateDockerfile() error = nil, want error")
		}
		if err.Error() != "Dockerfile not found in common locations" {
			t.Errorf("locateDockerfile() error = %v, want 'Dockerfile not found in common locations'", err)
		}
	})

	t.Run("error when specified Dockerfile not found", func(t *testing.T) {
		tempDir8 := t.TempDir()

		_, err := svc.locateDockerfile(tempDir8, "nonexistent/Dockerfile")
		if err == nil {
			t.Error("locateDockerfile() error = nil, want error")
		}
		if err.Error() != "specified Dockerfile not found: nonexistent/Dockerfile" {
			t.Errorf("locateDockerfile() error = %v, want 'specified Dockerfile not found'", err)
		}
	})
}

// TestValidateDockerfile testa a validação de Dockerfile
func TestValidateDockerfile(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewImageService(logger).(*imageService)

	tempDir := t.TempDir()

	t.Run("valid Dockerfile with FROM", func(t *testing.T) {
		dockerfilePath := filepath.Join(tempDir, "Dockerfile.valid")
		content := `FROM alpine:latest
RUN apk add --no-cache curl
CMD ["/bin/sh"]
`
		if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		err := svc.validateDockerfile(dockerfilePath)
		if err != nil {
			t.Errorf("validateDockerfile() error = %v, want nil", err)
		}
	})

	t.Run("valid Dockerfile with lowercase from", func(t *testing.T) {
		dockerfilePath := filepath.Join(tempDir, "Dockerfile.lowercase")
		content := `from alpine:latest
run apk add --no-cache curl
`
		if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		err := svc.validateDockerfile(dockerfilePath)
		if err != nil {
			t.Errorf("validateDockerfile() error = %v, want nil", err)
		}
	})

	t.Run("valid multi-stage Dockerfile", func(t *testing.T) {
		dockerfilePath := filepath.Join(tempDir, "Dockerfile.multistage")
		content := `FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o app

FROM alpine:latest
COPY --from=builder /app/app /app
CMD ["/app"]
`
		if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		err := svc.validateDockerfile(dockerfilePath)
		if err != nil {
			t.Errorf("validateDockerfile() error = %v, want nil", err)
		}
	})

	t.Run("error when Dockerfile is empty", func(t *testing.T) {
		dockerfilePath := filepath.Join(tempDir, "Dockerfile.empty")
		if err := os.WriteFile(dockerfilePath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		err := svc.validateDockerfile(dockerfilePath)
		if err == nil {
			t.Error("validateDockerfile() error = nil, want error")
		}
		if err.Error() != "Dockerfile is empty" {
			t.Errorf("validateDockerfile() error = %v, want 'Dockerfile is empty'", err)
		}
	})

	t.Run("error when Dockerfile has no FROM instruction", func(t *testing.T) {
		dockerfilePath := filepath.Join(tempDir, "Dockerfile.nofrom")
		content := `RUN apk add --no-cache curl
CMD ["/bin/sh"]
`
		if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		err := svc.validateDockerfile(dockerfilePath)
		if err == nil {
			t.Error("validateDockerfile() error = nil, want error")
		}
		if err.Error() != "Dockerfile must contain at least one FROM instruction" {
			t.Errorf("validateDockerfile() error = %v, want 'Dockerfile must contain at least one FROM instruction'", err)
		}
	})

	t.Run("error when Dockerfile does not exist", func(t *testing.T) {
		dockerfilePath := filepath.Join(tempDir, "Dockerfile.nonexistent")

		err := svc.validateDockerfile(dockerfilePath)
		if err == nil {
			t.Error("validateDockerfile() error = nil, want error")
		}
	})

	t.Run("valid Dockerfile with comments and whitespace", func(t *testing.T) {
		dockerfilePath := filepath.Join(tempDir, "Dockerfile.comments")
		content := `# This is a comment
# Another comment

FROM alpine:latest

# Install packages
RUN apk add --no-cache curl

CMD ["/bin/sh"]
`
		if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		err := svc.validateDockerfile(dockerfilePath)
		if err != nil {
			t.Errorf("validateDockerfile() error = %v, want nil", err)
		}
	})
}

// TestTagImage testa a aplicação de tags
func TestTagImage(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewImageService(logger).(*imageService)

	t.Run("error when image ID is empty", func(t *testing.T) {
		err := svc.TagImage("", []string{"myimage:latest"})
		if err == nil {
			t.Error("TagImage() error = nil, want error")
		}
		if err.Error() != "image ID is required" {
			t.Errorf("TagImage() error = %v, want 'image ID is required'", err)
		}
	})

	t.Run("error when tags are empty", func(t *testing.T) {
		err := svc.TagImage("abc123", []string{})
		if err == nil {
			t.Error("TagImage() error = nil, want error")
		}
		if err.Error() != "at least one tag is required" {
			t.Errorf("TagImage() error = %v, want 'at least one tag is required'", err)
		}
	})

	t.Run("error when tags are nil", func(t *testing.T) {
		err := svc.TagImage("abc123", nil)
		if err == nil {
			t.Error("TagImage() error = nil, want error")
		}
		if err.Error() != "at least one tag is required" {
			t.Errorf("TagImage() error = %v, want 'at least one tag is required'", err)
		}
	})
}

// TestExtractImageID testa a extração de image ID do output
func TestExtractImageID(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewImageService(logger).(*imageService)

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "64 character hash on last line",
			output:   "Building image...\n1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			expected: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		},
		{
			name:     "hash in middle of output",
			output:   "Step 1/3\nStep 2/3\n1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef\nSuccessfully built",
			expected: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		},
		{
			name:     "hash with other text",
			output:   "Successfully built 1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			expected: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		},
		{
			name:     "no hash in output",
			output:   "Building image...\nSuccessfully built",
			expected: "",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
		{
			name:     "short hash",
			output:   "Successfully built abc123",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractImageID(tt.output)
			if result != tt.expected {
				t.Errorf("extractImageID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGenerateImageTags testa a geração de tags de imagem
func TestGenerateImageTags(t *testing.T) {
	tests := []struct {
		name       string
		repoName   string
		commitHash string
		branch     string
		expected   []string
	}{
		{
			name:       "main branch with commit",
			repoName:   "myapp",
			commitHash: "abc123def456",
			branch:     "main",
			expected:   []string{"myapp:abc123def456", "myapp:main", "myapp:latest"},
		},
		{
			name:       "master branch with commit",
			repoName:   "myapp",
			commitHash: "abc123def456",
			branch:     "master",
			expected:   []string{"myapp:abc123def456", "myapp:master", "myapp:latest"},
		},
		{
			name:       "refs/heads/main with commit",
			repoName:   "myapp",
			commitHash: "abc123def456",
			branch:     "refs/heads/main",
			expected:   []string{"myapp:abc123def456", "myapp:main", "myapp:latest"},
		},
		{
			name:       "feature branch with commit",
			repoName:   "myapp",
			commitHash: "abc123def456",
			branch:     "feature/new-feature",
			expected:   []string{"myapp:abc123def456", "myapp:feature-new-feature"},
		},
		{
			name:       "develop branch with commit",
			repoName:   "myapp",
			commitHash: "abc123def456",
			branch:     "develop",
			expected:   []string{"myapp:abc123def456", "myapp:develop"},
		},
		{
			name:       "only commit hash",
			repoName:   "myapp",
			commitHash: "abc123def456",
			branch:     "",
			expected:   []string{"myapp:abc123def456"},
		},
		{
			name:       "only branch",
			repoName:   "myapp",
			commitHash: "",
			branch:     "develop",
			expected:   []string{"myapp:develop"},
		},
		{
			name:       "no commit or branch",
			repoName:   "myapp",
			commitHash: "",
			branch:     "",
			expected:   []string{},
		},
		{
			name:       "branch with multiple slashes",
			repoName:   "myapp",
			commitHash: "abc123",
			branch:     "feature/team/new-feature",
			expected:   []string{"myapp:abc123", "myapp:feature-team-new-feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateImageTags(tt.repoName, tt.commitHash, tt.branch)
			if len(result) != len(tt.expected) {
				t.Errorf("GenerateImageTags() returned %d tags, want %d", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Want: %v", tt.expected)
				return
			}
			for i, tag := range result {
				if tag != tt.expected[i] {
					t.Errorf("GenerateImageTags()[%d] = %v, want %v", i, tag, tt.expected[i])
				}
			}
		})
	}
}

// TestBuildImage_Validation testa validação de entrada no BuildImage
func TestBuildImage_Validation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewImageService(logger)
	ctx := context.Background()

	t.Run("error when context path is empty", func(t *testing.T) {
		config := ImageConfig{
			ContextPath: "",
			Tags:        []string{"myimage:latest"},
		}
		_, err := svc.BuildImage(ctx, config)
		if err == nil {
			t.Error("BuildImage() error = nil, want error")
		}
	})

	t.Run("error when context path does not exist", func(t *testing.T) {
		config := ImageConfig{
			ContextPath: "/nonexistent/path",
			Tags:        []string{"myimage:latest"},
		}
		_, err := svc.BuildImage(ctx, config)
		if err == nil {
			t.Error("BuildImage() error = nil, want error")
		}
	})

	t.Run("error when tags are empty", func(t *testing.T) {
		tempDir := t.TempDir()
		config := ImageConfig{
			ContextPath: tempDir,
			Tags:        []string{},
		}
		_, err := svc.BuildImage(ctx, config)
		if err == nil {
			t.Error("BuildImage() error = nil, want error")
		}
	})

	t.Run("error when Dockerfile not found", func(t *testing.T) {
		tempDir := t.TempDir()
		config := ImageConfig{
			ContextPath: tempDir,
			Tags:        []string{"myimage:latest"},
		}
		_, err := svc.BuildImage(ctx, config)
		if err == nil {
			t.Error("BuildImage() error = nil, want error")
		}
	})
}

// TestBuildImage_ContextCancellation testa cancelamento via context
func TestBuildImage_ContextCancellation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewImageService(logger)

	tempDir := t.TempDir()
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte("FROM alpine\n"), 0644); err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	t.Run("context cancelled before build", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		config := ImageConfig{
			ContextPath: tempDir,
			Tags:        []string{"myimage:latest"},
		}

		// This will fail because buildah won't be available in test environment
		// but we're testing that context cancellation is respected
		_, err := svc.BuildImage(ctx, config)
		if err == nil {
			t.Error("BuildImage() error = nil, want error for cancelled context")
		}
	})

	t.Run("context with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(time.Millisecond) // Ensure timeout occurs

		config := ImageConfig{
			ContextPath: tempDir,
			Tags:        []string{"myimage:latest"},
		}

		_, err := svc.BuildImage(ctx, config)
		if err == nil {
			t.Error("BuildImage() error = nil, want error for timeout")
		}
	})
}

// TestImageConfig_WithBuildArgs testa configuração com build args
func TestImageConfig_WithBuildArgs(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	svc := NewImageService(logger)

	tempDir := t.TempDir()
	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	content := `FROM alpine
ARG VERSION=1.0
RUN echo "Version: $VERSION"
`
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create Dockerfile: %v", err)
	}

	t.Run("config with build args", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		config := ImageConfig{
			ContextPath: tempDir,
			Tags:        []string{"myimage:latest"},
			BuildArgs: map[string]string{
				"VERSION": "2.0",
				"ENV":     "production",
			},
		}

		// This will fail in test environment without buildah, but validates config structure
		_, err := svc.BuildImage(ctx, config)
		// We expect an error because buildah is not available, but config should be valid
		if err != nil && err.Error() == "invalid image config: context_path is required" {
			t.Error("BuildImage() failed config validation, config should be valid")
		}
	})
}

// TestImageResult_Structure testa a estrutura do resultado
func TestImageResult_Structure(t *testing.T) {
	t.Run("ImageResult has required fields", func(t *testing.T) {
		result := &ImageResult{
			ImageID:  "abc123",
			Tags:     []string{"myimage:latest", "myimage:v1.0"},
			Size:     1024,
			Duration: time.Second * 30,
		}

		if result.ImageID != "abc123" {
			t.Errorf("ImageID = %v, want abc123", result.ImageID)
		}
		if len(result.Tags) != 2 {
			t.Errorf("len(Tags) = %d, want 2", len(result.Tags))
		}
		if result.Size != 1024 {
			t.Errorf("Size = %d, want 1024", result.Size)
		}
		if result.Duration != time.Second*30 {
			t.Errorf("Duration = %v, want 30s", result.Duration)
		}
	})
}
