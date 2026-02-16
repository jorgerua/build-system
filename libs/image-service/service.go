package imageservice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ImageService define a interface para construção de imagens OCI
type ImageService interface {
	BuildImage(ctx context.Context, config ImageConfig) (*ImageResult, error)
	TagImage(imageID string, tags []string) error
}

// ImageConfig contém a configuração para build de imagem
type ImageConfig struct {
	ContextPath    string            `json:"context_path"`
	DockerfilePath string            `json:"dockerfile_path"`
	Tags           []string          `json:"tags"`
	BuildArgs      map[string]string `json:"build_args"`
}

// ImageResult contém o resultado do build de imagem
type ImageResult struct {
	ImageID  string        `json:"image_id"`
	Tags     []string      `json:"tags"`
	Size     int64         `json:"size"`
	Duration time.Duration `json:"duration"`
}

// imageService implementa ImageService
type imageService struct {
	logger *zap.Logger
}

// NewImageService cria uma nova instância de ImageService
func NewImageService(logger *zap.Logger) ImageService {
	return &imageService{
		logger: logger,
	}
}

// BuildImage constrói uma imagem OCI usando buildah
func (s *imageService) BuildImage(ctx context.Context, config ImageConfig) (*ImageResult, error) {
	startTime := time.Now()

	s.logger.Info("starting image build",
		zap.String("context_path", config.ContextPath),
		zap.String("dockerfile_path", config.DockerfilePath),
		zap.Strings("tags", config.Tags),
	)

	// Validar configuração
	if err := s.validateConfig(config); err != nil {
		s.logger.Error("invalid image config", zap.Error(err))
		return nil, fmt.Errorf("invalid image config: %w", err)
	}

	// Localizar Dockerfile
	dockerfilePath, err := s.locateDockerfile(config.ContextPath, config.DockerfilePath)
	if err != nil {
		s.logger.Error("failed to locate Dockerfile", zap.Error(err))
		return nil, fmt.Errorf("failed to locate Dockerfile: %w", err)
	}

	s.logger.Debug("dockerfile located", zap.String("path", dockerfilePath))

	// Validar Dockerfile
	if err := s.validateDockerfile(dockerfilePath); err != nil {
		s.logger.Error("invalid Dockerfile", zap.Error(err))
		return nil, fmt.Errorf("invalid Dockerfile: %w", err)
	}

	// Construir comando buildah
	args := []string{
		"bud",
		"-f", dockerfilePath,
	}

	// Adicionar build args
	for key, value := range config.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Adicionar tags
	for _, tag := range config.Tags {
		args = append(args, "-t", tag)
	}

	// Adicionar context path
	args = append(args, config.ContextPath)

	// Executar buildah
	cmd := exec.CommandContext(ctx, "buildah", args...)
	cmd.Dir = config.ContextPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("buildah build failed",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return nil, fmt.Errorf("buildah build failed: %w: %s", err, string(output))
	}

	s.logger.Debug("buildah build output", zap.String("output", string(output)))

	// Extrair image ID do output
	imageID := s.extractImageID(string(output))
	if imageID == "" {
		s.logger.Warn("could not extract image ID from buildah output")
	}

	duration := time.Since(startTime)

	result := &ImageResult{
		ImageID:  imageID,
		Tags:     config.Tags,
		Size:     0, // Size calculation would require additional buildah inspect call
		Duration: duration,
	}

	s.logger.Info("image build completed",
		zap.String("image_id", imageID),
		zap.Strings("tags", config.Tags),
		zap.Duration("duration", duration),
	)

	return result, nil
}

// TagImage aplica tags adicionais a uma imagem existente
func (s *imageService) TagImage(imageID string, tags []string) error {
	if imageID == "" {
		return fmt.Errorf("image ID is required")
	}

	if len(tags) == 0 {
		return fmt.Errorf("at least one tag is required")
	}

	s.logger.Info("tagging image",
		zap.String("image_id", imageID),
		zap.Strings("tags", tags),
	)

	for _, tag := range tags {
		cmd := exec.Command("buildah", "tag", imageID, tag)
		output, err := cmd.CombinedOutput()
		if err != nil {
			s.logger.Error("failed to tag image",
				zap.String("image_id", imageID),
				zap.String("tag", tag),
				zap.Error(err),
				zap.String("output", string(output)),
			)
			return fmt.Errorf("failed to tag image %s with %s: %w: %s", imageID, tag, err, string(output))
		}

		s.logger.Debug("image tagged",
			zap.String("image_id", imageID),
			zap.String("tag", tag),
		)
	}

	s.logger.Info("image tagging completed",
		zap.String("image_id", imageID),
		zap.Strings("tags", tags),
	)

	return nil
}

// validateConfig valida a configuração de build
func (s *imageService) validateConfig(config ImageConfig) error {
	if config.ContextPath == "" {
		return fmt.Errorf("context_path is required")
	}

	// Verificar se context path existe
	if _, err := os.Stat(config.ContextPath); os.IsNotExist(err) {
		return fmt.Errorf("context_path does not exist: %s", config.ContextPath)
	}

	if len(config.Tags) == 0 {
		return fmt.Errorf("at least one tag is required")
	}

	return nil
}

// locateDockerfile localiza o Dockerfile no repositório
func (s *imageService) locateDockerfile(contextPath, dockerfilePath string) (string, error) {
	// Se um path específico foi fornecido, usar ele
	if dockerfilePath != "" {
		fullPath := dockerfilePath
		if !filepath.IsAbs(dockerfilePath) {
			fullPath = filepath.Join(contextPath, dockerfilePath)
		}

		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
		return "", fmt.Errorf("specified Dockerfile not found: %s", dockerfilePath)
	}

	// Locais comuns para procurar Dockerfile
	commonLocations := []string{
		"Dockerfile",
		"dockerfile",
		"docker/Dockerfile",
		"build/Dockerfile",
		".docker/Dockerfile",
		"deployment/Dockerfile",
	}

	for _, location := range commonLocations {
		fullPath := filepath.Join(contextPath, location)
		if _, err := os.Stat(fullPath); err == nil {
			s.logger.Debug("found Dockerfile", zap.String("location", location))
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("Dockerfile not found in common locations")
}

// validateDockerfile valida o conteúdo do Dockerfile
func (s *imageService) validateDockerfile(dockerfilePath string) error {
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	if len(content) == 0 {
		return fmt.Errorf("Dockerfile is empty")
	}

	// Verificar se contém pelo menos uma instrução FROM
	lines := strings.Split(string(content), "\n")
	hasFrom := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "FROM ") {
			hasFrom = true
			break
		}
	}

	if !hasFrom {
		return fmt.Errorf("Dockerfile must contain at least one FROM instruction")
	}

	return nil
}

// extractImageID extrai o image ID do output do buildah
func (s *imageService) extractImageID(output string) string {
	// O buildah geralmente imprime o image ID na última linha
	// Formato típico: "Successfully tagged localhost/myimage:latest"
	// ou simplesmente o hash SHA256
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return ""
	}

	lastLine := strings.TrimSpace(lines[len(lines)-1])

	// Se a última linha parece um hash SHA256 (64 caracteres hex)
	if len(lastLine) == 64 {
		return lastLine
	}

	// Procurar por linhas que contenham hash
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		// Procurar por padrão de hash SHA256
		if len(line) >= 64 {
			// Extrair possível hash
			fields := strings.Fields(line)
			for _, field := range fields {
				if len(field) == 64 {
					return field
				}
			}
		}
	}

	return ""
}

// GenerateImageTags gera tags baseadas em commit hash e branch
func GenerateImageTags(repoName, commitHash, branch string) []string {
	tags := make([]string, 0, 3)

	// Tag com commit hash completo
	if commitHash != "" {
		tags = append(tags, fmt.Sprintf("%s:%s", repoName, commitHash))
	}

	// Tag com branch
	if branch != "" {
		// Limpar nome do branch (remover refs/heads/ se presente)
		cleanBranch := strings.TrimPrefix(branch, "refs/heads/")
		// Substituir caracteres inválidos
		cleanBranch = strings.ReplaceAll(cleanBranch, "/", "-")
		tags = append(tags, fmt.Sprintf("%s:%s", repoName, cleanBranch))
	}

	// Tag latest se for branch main/master
	if branch == "refs/heads/main" || branch == "refs/heads/master" || branch == "main" || branch == "master" {
		tags = append(tags, fmt.Sprintf("%s:latest", repoName))
	}

	return tags
}
