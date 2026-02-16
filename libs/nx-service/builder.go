package nxservice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/oci-build-system/libs/shared"
	"go.uber.org/zap"
)

// NXService define a interface para execução de builds NX
type NXService interface {
	Build(ctx context.Context, repoPath string, config BuildConfig) (*BuildResult, error)
	DetectProjects(repoPath string) ([]string, error)
}

// BuildConfig contém configurações para o build
type BuildConfig struct {
	CachePath   string
	Language    shared.Language
	Environment map[string]string
}

// BuildResult contém o resultado de um build
type BuildResult struct {
	Success      bool
	Duration     time.Duration
	Output       string
	ErrorOutput  string
	ArtifactPath string
}

// nxBuilder implementa NXService
type nxBuilder struct {
	logger *zap.Logger
}

// NewNXService cria uma nova instância de NXService
func NewNXService(logger *zap.Logger) NXService {
	return &nxBuilder{
		logger: logger,
	}
}

// Build executa o build NX no repositório especificado
func (nb *nxBuilder) Build(ctx context.Context, repoPath string, config BuildConfig) (*BuildResult, error) {
	startTime := time.Now()

	nb.logger.Info("starting nx build",
		zap.String("repo_path", repoPath),
		zap.String("language", string(config.Language)),
	)

	// Verificar se o diretório existe
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	// Detectar linguagem se não especificada
	language := config.Language
	if language == shared.LanguageUnknown || language == "" {
		detectedLang, err := nb.detectLanguage(repoPath)
		if err != nil {
			nb.logger.Warn("failed to detect language, using unknown",
				zap.Error(err),
			)
			language = shared.LanguageUnknown
		} else {
			language = detectedLang
			nb.logger.Info("language detected",
				zap.String("language", string(language)),
			)
		}
	}

	// Preparar comando NX
	cmd := exec.CommandContext(ctx, "nx", "build")
	cmd.Dir = repoPath

	// Configurar variáveis de ambiente
	cmd.Env = os.Environ()

	// Adicionar variáveis de ambiente customizadas
	for key, value := range config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Configurar cache baseado na linguagem
	if config.CachePath != "" && language.IsSupported() {
		nb.configureCacheEnvironment(cmd, language, config.CachePath)
	}

	// Capturar stdout e stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Executar comando
	nb.logger.Debug("executing nx build command",
		zap.String("command", cmd.String()),
		zap.String("working_dir", cmd.Dir),
	)

	err := cmd.Run()
	duration := time.Since(startTime)

	result := &BuildResult{
		Success:      err == nil,
		Duration:     duration,
		Output:       stdout.String(),
		ErrorOutput:  stderr.String(),
		ArtifactPath: filepath.Join(repoPath, "dist"),
	}

	if err != nil {
		nb.logger.Error("nx build failed",
			zap.String("repo_path", repoPath),
			zap.Duration("duration", duration),
			zap.Error(err),
			zap.String("stderr", stderr.String()),
		)
		return result, fmt.Errorf("nx build failed: %w", err)
	}

	nb.logger.Info("nx build completed successfully",
		zap.String("repo_path", repoPath),
		zap.Duration("duration", duration),
	)

	return result, nil
}

// DetectProjects descobre projetos NX no repositório
func (nb *nxBuilder) DetectProjects(repoPath string) ([]string, error) {
	nb.logger.Debug("detecting nx projects",
		zap.String("repo_path", repoPath),
	)

	// Verificar se existe workspace.json ou nx.json
	workspaceFile := filepath.Join(repoPath, "workspace.json")
	nxFile := filepath.Join(repoPath, "nx.json")

	if _, err := os.Stat(workspaceFile); err == nil {
		return nb.parseWorkspaceJson(workspaceFile)
	}

	if _, err := os.Stat(nxFile); err == nil {
		// NX moderno usa project.json em cada projeto
		return nb.findProjectJsonFiles(repoPath)
	}

	return nil, fmt.Errorf("no nx workspace configuration found")
}

// detectLanguage detecta a linguagem do projeto baseado em arquivos de configuração
func (nb *nxBuilder) detectLanguage(repoPath string) (shared.Language, error) {
	// Verificar Java (Maven)
	if _, err := os.Stat(filepath.Join(repoPath, "pom.xml")); err == nil {
		return shared.LanguageJava, nil
	}

	// Verificar Java (Gradle)
	if _, err := os.Stat(filepath.Join(repoPath, "build.gradle")); err == nil {
		return shared.LanguageJava, nil
	}
	if _, err := os.Stat(filepath.Join(repoPath, "build.gradle.kts")); err == nil {
		return shared.LanguageJava, nil
	}

	// Verificar .NET
	csprojFiles, err := filepath.Glob(filepath.Join(repoPath, "*.csproj"))
	if err == nil && len(csprojFiles) > 0 {
		return shared.LanguageDotNet, nil
	}

	// Verificar Go
	if _, err := os.Stat(filepath.Join(repoPath, "go.mod")); err == nil {
		return shared.LanguageGo, nil
	}

	// Buscar em subdiretórios comuns
	commonDirs := []string{"apps", "libs", "packages", "src"}
	for _, dir := range commonDirs {
		dirPath := filepath.Join(repoPath, dir)
		if _, err := os.Stat(dirPath); err == nil {
			// Buscar recursivamente em subdiretórios
			lang, err := nb.detectLanguageInDirectory(dirPath)
			if err == nil && lang.IsSupported() {
				return lang, nil
			}
		}
	}

	return shared.LanguageUnknown, fmt.Errorf("unable to detect language")
}

// detectLanguageInDirectory detecta linguagem em um diretório específico
func (nb *nxBuilder) detectLanguageInDirectory(dirPath string) (shared.Language, error) {
	var detectedLang shared.Language = shared.LanguageUnknown

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continuar mesmo com erros
		}

		if info.IsDir() {
			return nil
		}

		// Verificar por arquivos de configuração
		name := info.Name()
		switch {
		case name == "pom.xml" || name == "build.gradle" || name == "build.gradle.kts":
			detectedLang = shared.LanguageJava
			return filepath.SkipAll // Encontrou, pode parar
		case strings.HasSuffix(name, ".csproj"):
			detectedLang = shared.LanguageDotNet
			return filepath.SkipAll
		case name == "go.mod":
			detectedLang = shared.LanguageGo
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return shared.LanguageUnknown, err
	}

	if detectedLang.IsSupported() {
		return detectedLang, nil
	}

	return shared.LanguageUnknown, fmt.Errorf("no language detected in directory")
}

// configureCacheEnvironment configura variáveis de ambiente para cache por linguagem
func (nb *nxBuilder) configureCacheEnvironment(cmd *exec.Cmd, language shared.Language, cachePath string) {
	switch language {
	case shared.LanguageJava:
		// Maven cache
		mavenCache := filepath.Join(cachePath, "maven")
		cmd.Env = append(cmd.Env, fmt.Sprintf("MAVEN_OPTS=-Dmaven.repo.local=%s", mavenCache))

		// Gradle cache
		gradleCache := filepath.Join(cachePath, "gradle")
		cmd.Env = append(cmd.Env, fmt.Sprintf("GRADLE_USER_HOME=%s", gradleCache))

		nb.logger.Debug("configured java cache",
			zap.String("maven_cache", mavenCache),
			zap.String("gradle_cache", gradleCache),
		)

	case shared.LanguageDotNet:
		// NuGet cache
		nugetCache := filepath.Join(cachePath, "nuget")
		cmd.Env = append(cmd.Env, fmt.Sprintf("NUGET_PACKAGES=%s", nugetCache))

		nb.logger.Debug("configured dotnet cache",
			zap.String("nuget_cache", nugetCache),
		)

	case shared.LanguageGo:
		// Go modules cache
		goCache := filepath.Join(cachePath, "go")
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOMODCACHE=%s", goCache))
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOCACHE=%s/build-cache", goCache))

		nb.logger.Debug("configured go cache",
			zap.String("go_cache", goCache),
		)
	}
}

// parseWorkspaceJson analisa workspace.json para encontrar projetos
func (nb *nxBuilder) parseWorkspaceJson(filePath string) ([]string, error) {
	// Implementação simplificada - em produção, usar encoding/json
	// Por enquanto, retornar lista vazia
	nb.logger.Warn("workspace.json parsing not fully implemented",
		zap.String("file", filePath),
	)
	return []string{}, nil
}

// findProjectJsonFiles encontra todos os arquivos project.json no workspace
func (nb *nxBuilder) findProjectJsonFiles(repoPath string) ([]string, error) {
	var projects []string

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continuar mesmo com erros
		}

		// Pular node_modules e outros diretórios comuns
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "dist" || name == ".nx" {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Name() == "project.json" {
			// Extrair nome do projeto do caminho
			projectDir := filepath.Dir(path)
			projectName := filepath.Base(projectDir)
			projects = append(projects, projectName)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to find project.json files: %w", err)
	}

	nb.logger.Debug("found nx projects",
		zap.Int("count", len(projects)),
		zap.Strings("projects", projects),
	)

	return projects, nil
}
