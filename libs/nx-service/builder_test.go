package nxservice

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oci-build-system/libs/shared"
	"go.uber.org/zap"
)

// mockExecCommand é usado para mockar exec.Command em testes
var execCommand = exec.Command

// TestNewNXService testa a criação de uma nova instância do serviço
func TestNewNXService(t *testing.T) {
	logger := zap.NewNop()
	service := NewNXService(logger)

	if service == nil {
		t.Fatal("NewNXService returned nil")
	}

	// Verificar que retorna a interface correta
	_, ok := service.(NXService)
	if !ok {
		t.Error("NewNXService did not return NXService interface")
	}
}

// TestBuild_SuccessfulBuild testa execução de build bem-sucedido
func TestBuild_SuccessfulBuild(t *testing.T) {
	// Criar diretório temporário para simular repositório
	tempDir := t.TempDir()

	// Criar arquivo go.mod para detecção de linguagem
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	logger := zap.NewNop()
	service := NewNXService(logger)

	config := BuildConfig{
		CachePath:   filepath.Join(tempDir, "cache"),
		Language:    shared.LanguageGo,
		Environment: map[string]string{"TEST_VAR": "test_value"},
	}

	// Criar contexto com timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Nota: Este teste falhará se nx não estiver instalado
	// Em um ambiente de CI/CD real, você mockaria a execução do comando
	result, err := service.Build(ctx, tempDir, config)

	// Verificar que a função retorna um resultado mesmo com erro
	if result == nil {
		t.Fatal("Build returned nil result")
	}

	// Verificar que a duração foi registrada
	if result.Duration == 0 {
		t.Error("Build duration was not recorded")
	}

	// Verificar que o artifact path foi definido
	expectedArtifactPath := filepath.Join(tempDir, "dist")
	if result.ArtifactPath != expectedArtifactPath {
		t.Errorf("Expected artifact path %s, got %s", expectedArtifactPath, result.ArtifactPath)
	}

	// Se nx não estiver instalado, esperamos um erro
	if err != nil {
		// Verificar que o erro foi capturado corretamente
		if !strings.Contains(err.Error(), "nx build failed") {
			t.Errorf("Expected 'nx build failed' error, got: %v", err)
		}
		// Verificar que Success é false
		if result.Success {
			t.Error("Expected Success to be false when build fails")
		}
	}
}

// TestBuild_CaptureOutput testa captura de stdout e stderr
func TestBuild_CaptureOutput(t *testing.T) {
	tempDir := t.TempDir()

	// Criar arquivo go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	logger := zap.NewNop()
	service := NewNXService(logger)

	config := BuildConfig{
		Language: shared.LanguageGo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, _ := service.Build(ctx, tempDir, config)

	// Verificar que Output e ErrorOutput são strings (podem estar vazias)
	if result.Output == "" && result.ErrorOutput == "" {
		// Pelo menos um deve ter conteúdo se o comando foi executado
		t.Log("Both Output and ErrorOutput are empty - command may not have produced output")
	}

	// Verificar que os campos existem e são do tipo correto
	_ = result.Output
	_ = result.ErrorOutput
}

// TestBuild_NonExistentDirectory testa build com diretório inexistente
func TestBuild_NonExistentDirectory(t *testing.T) {
	logger := zap.NewNop()
	service := NewNXService(logger)

	config := BuildConfig{
		Language: shared.LanguageGo,
	}

	ctx := context.Background()
	nonExistentPath := "/path/that/does/not/exist/12345"

	result, err := service.Build(ctx, nonExistentPath, config)

	// Deve retornar erro
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}

	// Deve retornar nil result
	if result != nil {
		t.Error("Expected nil result for non-existent directory")
	}

	// Verificar mensagem de erro
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' in error message, got: %v", err)
	}
}

// TestBuild_ContextTimeout testa timeout de build
func TestBuild_ContextTimeout(t *testing.T) {
	tempDir := t.TempDir()

	// Criar arquivo go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	logger := zap.NewNop()
	service := NewNXService(logger)

	config := BuildConfig{
		Language: shared.LanguageGo,
	}

	// Criar contexto com timeout muito curto
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Aguardar para garantir que o contexto expire
	time.Sleep(10 * time.Millisecond)

	result, err := service.Build(ctx, tempDir, config)

	// Deve retornar erro ou resultado com falha
	if err == nil && result != nil && result.Success {
		t.Error("Expected timeout error or failed result")
	}
}

// TestDetectLanguage_Java testa detecção de linguagem Java
func TestDetectLanguage_Java(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected shared.Language
	}{
		{"Maven", "pom.xml", shared.LanguageJava},
		{"Gradle", "build.gradle", shared.LanguageJava},
		{"Gradle Kotlin", "build.gradle.kts", shared.LanguageJava},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Criar arquivo de configuração
			filePath := filepath.Join(tempDir, tt.filename)
			if err := os.WriteFile(filePath, []byte("test content\n"), 0644); err != nil {
				t.Fatalf("Failed to create %s: %v", tt.filename, err)
			}

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			lang, err := builder.detectLanguage(tempDir)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if lang != tt.expected {
				t.Errorf("Expected language %s, got %s", tt.expected, lang)
			}
		})
	}
}

// TestDetectLanguage_DotNet testa detecção de linguagem .NET
func TestDetectLanguage_DotNet(t *testing.T) {
	tempDir := t.TempDir()

	// Criar arquivo .csproj
	csprojPath := filepath.Join(tempDir, "test.csproj")
	if err := os.WriteFile(csprojPath, []byte("<Project></Project>\n"), 0644); err != nil {
		t.Fatalf("Failed to create .csproj: %v", err)
	}

	logger := zap.NewNop()
	builder := &nxBuilder{logger: logger}

	lang, err := builder.detectLanguage(tempDir)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if lang != shared.LanguageDotNet {
		t.Errorf("Expected language %s, got %s", shared.LanguageDotNet, lang)
	}
}

// TestDetectLanguage_Go testa detecção de linguagem Go
func TestDetectLanguage_Go(t *testing.T) {
	tempDir := t.TempDir()

	// Criar arquivo go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	logger := zap.NewNop()
	builder := &nxBuilder{logger: logger}

	lang, err := builder.detectLanguage(tempDir)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if lang != shared.LanguageGo {
		t.Errorf("Expected language %s, got %s", shared.LanguageGo, lang)
	}
}

// TestDetectLanguage_InSubdirectory testa detecção em subdiretórios
func TestDetectLanguage_InSubdirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Criar subdiretório apps
	appsDir := filepath.Join(tempDir, "apps")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		t.Fatalf("Failed to create apps directory: %v", err)
	}

	// Criar arquivo go.mod no subdiretório
	goModPath := filepath.Join(appsDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	logger := zap.NewNop()
	builder := &nxBuilder{logger: logger}

	lang, err := builder.detectLanguage(tempDir)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if lang != shared.LanguageGo {
		t.Errorf("Expected language %s, got %s", shared.LanguageGo, lang)
	}
}

// TestDetectLanguage_Unknown testa quando nenhuma linguagem é detectada
func TestDetectLanguage_Unknown(t *testing.T) {
	tempDir := t.TempDir()

	// Não criar nenhum arquivo de configuração

	logger := zap.NewNop()
	builder := &nxBuilder{logger: logger}

	lang, err := builder.detectLanguage(tempDir)

	if err == nil {
		t.Error("Expected error when no language detected")
	}

	if lang != shared.LanguageUnknown {
		t.Errorf("Expected language %s, got %s", shared.LanguageUnknown, lang)
	}
}

// TestConfigureCacheEnvironment_Java testa configuração de cache para Java
func TestConfigureCacheEnvironment_Java(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache")

	logger := zap.NewNop()
	builder := &nxBuilder{logger: logger}

	cmd := exec.Command("echo", "test")
	cmd.Env = os.Environ()

	builder.configureCacheEnvironment(cmd, shared.LanguageJava, cachePath)

	// Verificar que as variáveis de ambiente foram adicionadas
	foundMaven := false
	foundGradle := false

	for _, env := range cmd.Env {
		if strings.Contains(env, "MAVEN_OPTS") && strings.Contains(env, filepath.Join(cachePath, "maven")) {
			foundMaven = true
		}
		if strings.Contains(env, "GRADLE_USER_HOME") && strings.Contains(env, filepath.Join(cachePath, "gradle")) {
			foundGradle = true
		}
	}

	if !foundMaven {
		t.Error("MAVEN_OPTS not configured correctly")
	}
	if !foundGradle {
		t.Error("GRADLE_USER_HOME not configured correctly")
	}
}

// TestConfigureCacheEnvironment_DotNet testa configuração de cache para .NET
func TestConfigureCacheEnvironment_DotNet(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache")

	logger := zap.NewNop()
	builder := &nxBuilder{logger: logger}

	cmd := exec.Command("echo", "test")
	cmd.Env = os.Environ()

	builder.configureCacheEnvironment(cmd, shared.LanguageDotNet, cachePath)

	// Verificar que NUGET_PACKAGES foi adicionado
	found := false
	for _, env := range cmd.Env {
		if strings.Contains(env, "NUGET_PACKAGES") && strings.Contains(env, filepath.Join(cachePath, "nuget")) {
			found = true
			break
		}
	}

	if !found {
		t.Error("NUGET_PACKAGES not configured correctly")
	}
}

// TestConfigureCacheEnvironment_Go testa configuração de cache para Go
func TestConfigureCacheEnvironment_Go(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache")

	logger := zap.NewNop()
	builder := &nxBuilder{logger: logger}

	cmd := exec.Command("echo", "test")
	cmd.Env = os.Environ()

	builder.configureCacheEnvironment(cmd, shared.LanguageGo, cachePath)

	// Verificar que as variáveis Go foram adicionadas
	foundModCache := false
	foundGoCache := false

	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "GOMODCACHE=") {
			// Verificar que contém o caminho do cache (normalizar separadores)
			envValue := strings.TrimPrefix(env, "GOMODCACHE=")
			expectedPath := filepath.Join(cachePath, "go")
			// Normalizar ambos os caminhos para comparação
			if filepath.Clean(envValue) == filepath.Clean(expectedPath) {
				foundModCache = true
			}
		}
		if strings.HasPrefix(env, "GOCACHE=") {
			// Verificar que contém o caminho do cache de build
			envValue := strings.TrimPrefix(env, "GOCACHE=")
			expectedPath := filepath.Join(cachePath, "go", "build-cache")
			// Normalizar ambos os caminhos para comparação
			if filepath.Clean(envValue) == filepath.Clean(expectedPath) {
				foundGoCache = true
			}
		}
	}

	if !foundModCache {
		t.Error("GOMODCACHE not configured correctly")
	}
	if !foundGoCache {
		t.Error("GOCACHE not configured correctly")
	}
}

// TestDetectProjects testa descoberta de projetos NX
func TestDetectProjects(t *testing.T) {
	t.Run("NoWorkspaceConfig", func(t *testing.T) {
		tempDir := t.TempDir()

		logger := zap.NewNop()
		service := NewNXService(logger)

		projects, err := service.DetectProjects(tempDir)

		if err == nil {
			t.Error("Expected error when no workspace config found")
		}

		if projects != nil {
			t.Error("Expected nil projects when no workspace config found")
		}
	})

	t.Run("WithProjectJson", func(t *testing.T) {
		tempDir := t.TempDir()

		// Criar nx.json
		nxJsonPath := filepath.Join(tempDir, "nx.json")
		if err := os.WriteFile(nxJsonPath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create nx.json: %v", err)
		}

		// Criar estrutura de projeto
		projectDir := filepath.Join(tempDir, "apps", "my-app")
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("Failed to create project directory: %v", err)
		}

		projectJsonPath := filepath.Join(projectDir, "project.json")
		if err := os.WriteFile(projectJsonPath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create project.json: %v", err)
		}

		logger := zap.NewNop()
		service := NewNXService(logger)

		projects, err := service.DetectProjects(tempDir)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(projects) == 0 {
			t.Error("Expected to find at least one project")
		}

		// Verificar que encontrou o projeto
		found := false
		for _, p := range projects {
			if p == "my-app" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected to find 'my-app' project, got: %v", projects)
		}
	})

	t.Run("SkipNodeModules", func(t *testing.T) {
		tempDir := t.TempDir()

		// Criar nx.json
		nxJsonPath := filepath.Join(tempDir, "nx.json")
		if err := os.WriteFile(nxJsonPath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create nx.json: %v", err)
		}

		// Criar project.json em node_modules (deve ser ignorado)
		nodeModulesDir := filepath.Join(tempDir, "node_modules", "some-package")
		if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
			t.Fatalf("Failed to create node_modules directory: %v", err)
		}

		projectJsonPath := filepath.Join(nodeModulesDir, "project.json")
		if err := os.WriteFile(projectJsonPath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create project.json in node_modules: %v", err)
		}

		logger := zap.NewNop()
		service := NewNXService(logger)

		projects, err := service.DetectProjects(tempDir)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verificar que não encontrou projeto em node_modules
		for _, p := range projects {
			if p == "some-package" {
				t.Error("Should not find projects in node_modules")
			}
		}
	})
}

// TestBuild_WithCustomEnvironment testa build com variáveis de ambiente customizadas
func TestBuild_WithCustomEnvironment(t *testing.T) {
	tempDir := t.TempDir()

	// Criar arquivo go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	logger := zap.NewNop()
	service := NewNXService(logger)

	customEnv := map[string]string{
		"CUSTOM_VAR_1": "value1",
		"CUSTOM_VAR_2": "value2",
	}

	config := BuildConfig{
		Language:    shared.LanguageGo,
		Environment: customEnv,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, _ := service.Build(ctx, tempDir, config)

	// Verificar que o resultado foi criado
	if result == nil {
		t.Fatal("Build returned nil result")
	}

	// As variáveis de ambiente são passadas para o comando,
	// mas não podemos verificá-las diretamente no resultado
	// Este teste garante que o código não falha com env customizado
}

// TestBuild_LanguageAutoDetection testa detecção automática de linguagem
func TestBuild_LanguageAutoDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Criar arquivo go.mod para detecção automática
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	logger := zap.NewNop()
	service := NewNXService(logger)

	// Não especificar linguagem (ou usar Unknown)
	config := BuildConfig{
		Language: shared.LanguageUnknown,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, _ := service.Build(ctx, tempDir, config)

	// Verificar que o resultado foi criado
	if result == nil {
		t.Fatal("Build returned nil result")
	}

	// A linguagem deve ter sido detectada automaticamente
	// (verificado internamente pelo serviço)
}
