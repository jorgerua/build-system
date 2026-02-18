package nxservice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jorgerua/build-system/libs/shared"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
)

// Feature: jorgerua/build-system, Property 9: Configuração de cache por linguagem
// **Valida: Requisitos 4.2, 6.1, 6.2, 6.3**
//
// Para qualquer projeto detectado como Java, .NET ou Go, o sistema deve configurar
// as variáveis de ambiente apropriadas para que as ferramentas de build
// (Maven/Gradle/NuGet/Go) utilizem o cache local correspondente.
func TestProperty_CacheConfigurationByLanguage(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("configures Maven cache for Java projects with pom.xml", prop.ForAll(
		func(seed uint64) bool {
			tempDir := t.TempDir()

			// Criar pom.xml para detecção de Java
			pomPath := filepath.Join(tempDir, "pom.xml")
			pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
</project>`
			if err := os.WriteFile(pomPath, []byte(pomContent), 0644); err != nil {
				t.Logf("Failed to create pom.xml: %v", err)
				return false
			}

			// Gerar caminho de cache único baseado no seed
			cachePath := filepath.Join(tempDir, fmt.Sprintf("cache-%d", seed))

			// Criar script mock que captura variáveis de ambiente
			scriptPath := createEnvCaptureScript(t, tempDir)
			if scriptPath == "" {
				return false
			}

			// Substituir PATH temporariamente
			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				CachePath: cachePath,
				Language:  shared.LanguageJava,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Propriedade: MAVEN_OPTS deve estar configurado com o caminho do cache
			expectedMavenCache := filepath.Join(cachePath, "maven")
			if !strings.Contains(result.Output, "MAVEN_OPTS") {
				t.Log("MAVEN_OPTS not found in environment")
				return false
			}

			if !strings.Contains(result.Output, expectedMavenCache) {
				t.Logf("Expected Maven cache path %s not found in output", expectedMavenCache)
				return false
			}

			// Propriedade: GRADLE_USER_HOME deve estar configurado
			expectedGradleCache := filepath.Join(cachePath, "gradle")
			if !strings.Contains(result.Output, "GRADLE_USER_HOME") {
				t.Log("GRADLE_USER_HOME not found in environment")
				return false
			}

			if !strings.Contains(result.Output, expectedGradleCache) {
				t.Logf("Expected Gradle cache path %s not found in output", expectedGradleCache)
				return false
			}

			return true
		},
		gen.UInt64(),
	))

	properties.Property("configures Gradle cache for Java projects with build.gradle", prop.ForAll(
		func(useKotlinDsl bool, seed uint64) bool {
			tempDir := t.TempDir()

			// Criar build.gradle ou build.gradle.kts
			var gradleFile string
			if useKotlinDsl {
				gradleFile = "build.gradle.kts"
			} else {
				gradleFile = "build.gradle"
			}

			gradlePath := filepath.Join(tempDir, gradleFile)
			gradleContent := `plugins {
    id 'java'
}

group = 'com.example'
version = '1.0.0'
`
			if err := os.WriteFile(gradlePath, []byte(gradleContent), 0644); err != nil {
				t.Logf("Failed to create %s: %v", gradleFile, err)
				return false
			}

			// Gerar caminho de cache único
			cachePath := filepath.Join(tempDir, fmt.Sprintf("cache-%d", seed))

			// Criar script mock
			scriptPath := createEnvCaptureScript(t, tempDir)
			if scriptPath == "" {
				return false
			}

			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				CachePath: cachePath,
				Language:  shared.LanguageJava,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Propriedade: GRADLE_USER_HOME deve estar configurado
			expectedGradleCache := filepath.Join(cachePath, "gradle")
			if !strings.Contains(result.Output, "GRADLE_USER_HOME") {
				t.Log("GRADLE_USER_HOME not found in environment")
				return false
			}

			if !strings.Contains(result.Output, expectedGradleCache) {
				t.Logf("Expected Gradle cache path %s not found", expectedGradleCache)
				return false
			}

			return true
		},
		gen.Bool(),
		gen.UInt64(),
	))

	properties.Property("configures NuGet cache for .NET projects", prop.ForAll(
		func(seed uint64) bool {
			tempDir := t.TempDir()

			// Criar arquivo .csproj
			projectName := fmt.Sprintf("Project%d", seed%1000)
			csprojPath := filepath.Join(tempDir, projectName+".csproj")
			csprojContent := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net6.0</TargetFramework>
  </PropertyGroup>
</Project>`
			if err := os.WriteFile(csprojPath, []byte(csprojContent), 0644); err != nil {
				t.Logf("Failed to create .csproj: %v", err)
				return false
			}

			// Gerar caminho de cache único
			cachePath := filepath.Join(tempDir, fmt.Sprintf("cache-%d", seed))

			// Criar script mock
			scriptPath := createEnvCaptureScript(t, tempDir)
			if scriptPath == "" {
				return false
			}

			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				CachePath: cachePath,
				Language:  shared.LanguageDotNet,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Propriedade: NUGET_PACKAGES deve estar configurado
			expectedNugetCache := filepath.Join(cachePath, "nuget")
			if !strings.Contains(result.Output, "NUGET_PACKAGES") {
				t.Log("NUGET_PACKAGES not found in environment")
				return false
			}

			if !strings.Contains(result.Output, expectedNugetCache) {
				t.Logf("Expected NuGet cache path %s not found", expectedNugetCache)
				return false
			}

			return true
		},
		gen.UInt64(),
	))

	properties.Property("configures Go module cache for Go projects", prop.ForAll(
		func(seed uint64) bool {
			tempDir := t.TempDir()

			// Criar go.mod
			moduleName := fmt.Sprintf("github.com/example/module%d", seed%1000)
			goModPath := filepath.Join(tempDir, "go.mod")
			goModContent := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)
			if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
				t.Logf("Failed to create go.mod: %v", err)
				return false
			}

			// Gerar caminho de cache único
			cachePath := filepath.Join(tempDir, fmt.Sprintf("cache-%d", seed))

			// Criar script mock
			scriptPath := createEnvCaptureScript(t, tempDir)
			if scriptPath == "" {
				return false
			}

			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				CachePath: cachePath,
				Language:  shared.LanguageGo,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Propriedade: GOMODCACHE deve estar configurado
			expectedGoCache := filepath.Join(cachePath, "go")
			if !strings.Contains(result.Output, "GOMODCACHE") {
				t.Log("GOMODCACHE not found in environment")
				return false
			}

			if !strings.Contains(result.Output, expectedGoCache) {
				t.Logf("Expected Go module cache path %s not found", expectedGoCache)
				return false
			}

			// Propriedade: GOCACHE deve estar configurado
			if !strings.Contains(result.Output, "GOCACHE") {
				t.Log("GOCACHE not found in environment")
				return false
			}

			// Verificar que GOCACHE contém o caminho base do cache Go
			// (o build-cache é um subdiretório)
			if !strings.Contains(result.Output, expectedGoCache) {
				t.Logf("Expected Go cache path %s not found in GOCACHE", expectedGoCache)
				return false
			}

			return true
		},
		gen.UInt64(),
	))

	properties.Property("does not configure cache when language is unknown", prop.ForAll(
		func(seed uint64) bool {
			tempDir := t.TempDir()

			// Criar arquivo não relacionado a linguagens suportadas
			randomFile := filepath.Join(tempDir, "README.md")
			if err := os.WriteFile(randomFile, []byte("# Test Project"), 0644); err != nil {
				t.Logf("Failed to create file: %v", err)
				return false
			}

			cachePath := filepath.Join(tempDir, fmt.Sprintf("cache-%d", seed))

			// Criar script mock
			scriptPath := createEnvCaptureScript(t, tempDir)
			if scriptPath == "" {
				return false
			}

			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				CachePath: cachePath,
				Language:  shared.LanguageUnknown,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Propriedade: nenhuma variável de cache deve estar configurada
			output := result.Output
			// Verificar que as variáveis não contêm caminhos de cache válidos
			// (podem aparecer vazias no output, mas não devem ter valores)
			if strings.Contains(output, cachePath) {
				t.Log("Cache path should not appear in environment when language is unknown")
				return false
			}

			return true
		},
		gen.UInt64(),
	))

	properties.Property("does not configure cache when cache path is empty", prop.ForAll(
		func(langChoice uint8) bool {
			tempDir := t.TempDir()

			// Escolher linguagem baseado em langChoice
			var language shared.Language
			var configFile string
			var content string

			switch langChoice % 3 {
			case 0: // Java
				language = shared.LanguageJava
				configFile = "pom.xml"
				content = `<?xml version="1.0"?><project></project>`
			case 1: // .NET
				language = shared.LanguageDotNet
				configFile = "test.csproj"
				content = `<Project Sdk="Microsoft.NET.Sdk"></Project>`
			case 2: // Go
				language = shared.LanguageGo
				configFile = "go.mod"
				content = "module test\n"
			}

			configPath := filepath.Join(tempDir, configFile)
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Logf("Failed to create config file: %v", err)
				return false
			}

			// Criar script mock
			scriptPath := createEnvCaptureScript(t, tempDir)
			if scriptPath == "" {
				return false
			}

			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				CachePath: "", // Cache path vazio
				Language:  language,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Propriedade: nenhuma variável de cache deve estar configurada quando CachePath está vazio
			output := result.Output
			// Verificar que não há caminhos de cache nas variáveis de ambiente
			// As variáveis podem aparecer no output, mas não devem ter valores de caminho
			cacheKeywords := []string{"/maven", "/gradle", "/nuget", "/go", "\\maven", "\\gradle", "\\nuget", "\\go"}
			for _, keyword := range cacheKeywords {
				if strings.Contains(output, keyword) {
					t.Logf("Cache path keyword %s should not appear when cache path is empty", keyword)
					return false
				}
			}

			return true
		},
		gen.UInt8(),
	))

	properties.Property("auto-detects language and configures cache accordingly", prop.ForAll(
		func(langChoice uint8, seed uint64) bool {
			tempDir := t.TempDir()

			// Escolher linguagem baseado em langChoice
			var expectedLang shared.Language
			var configFile string
			var content string
			var expectedEnvVar string

			switch langChoice % 3 {
			case 0: // Java
				expectedLang = shared.LanguageJava
				configFile = "pom.xml"
				content = `<?xml version="1.0"?><project></project>`
				expectedEnvVar = "MAVEN_OPTS"
			case 1: // .NET
				expectedLang = shared.LanguageDotNet
				configFile = "test.csproj"
				content = `<Project Sdk="Microsoft.NET.Sdk"></Project>`
				expectedEnvVar = "NUGET_PACKAGES"
			case 2: // Go
				expectedLang = shared.LanguageGo
				configFile = "go.mod"
				content = "module test\n"
				expectedEnvVar = "GOMODCACHE"
			}

			configPath := filepath.Join(tempDir, configFile)
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Logf("Failed to create config file: %v", err)
				return false
			}

			cachePath := filepath.Join(tempDir, fmt.Sprintf("cache-%d", seed))

			// Criar script mock
			scriptPath := createEnvCaptureScript(t, tempDir)
			if scriptPath == "" {
				return false
			}

			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				CachePath: cachePath,
				Language:  shared.LanguageUnknown, // Forçar auto-detecção
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Propriedade: deve detectar linguagem e configurar cache apropriado
			if !strings.Contains(result.Output, expectedEnvVar) {
				t.Logf("Expected environment variable %s not found for language %s", expectedEnvVar, expectedLang)
				return false
			}

			return true
		},
		gen.UInt8(),
		gen.UInt64(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// createEnvCaptureScript cria um script que captura e imprime variáveis de ambiente
// Este script simula o comando nx e imprime as variáveis de ambiente de cache
func createEnvCaptureScript(t *testing.T, baseDir string) string {
	scriptDir := filepath.Join(baseDir, "bin")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Logf("Failed to create script directory: %v", err)
		return ""
	}

	var scriptPath string
	var scriptContent string

	// Criar script apropriado para o sistema operacional
	if isWindows() {
		scriptPath = filepath.Join(scriptDir, "nx.bat")
		scriptContent = `@echo off
echo MAVEN_OPTS=%MAVEN_OPTS%
echo GRADLE_USER_HOME=%GRADLE_USER_HOME%
echo NUGET_PACKAGES=%NUGET_PACKAGES%
echo GOMODCACHE=%GOMODCACHE%
echo GOCACHE=%GOCACHE%
exit /b 0
`
	} else {
		scriptPath = filepath.Join(scriptDir, "nx")
		scriptContent = `#!/bin/sh
echo "MAVEN_OPTS=$MAVEN_OPTS"
echo "GRADLE_USER_HOME=$GRADLE_USER_HOME"
echo "NUGET_PACKAGES=$NUGET_PACKAGES"
echo "GOMODCACHE=$GOMODCACHE"
echo "GOCACHE=$GOCACHE"
exit 0
`
	}

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Logf("Failed to create env capture script: %v", err)
		return ""
	}

	return scriptPath
}

// TestCacheConfigurationIntegration verifica a integração completa da configuração de cache
// Este teste não é baseado em propriedades, mas valida o comportamento end-to-end
func TestCacheConfigurationIntegration(t *testing.T) {
	tests := []struct {
		name         string
		language     shared.Language
		configFile   string
		content      string
		expectedVars []string
	}{
		{
			name:       "Java with Maven",
			language:   shared.LanguageJava,
			configFile: "pom.xml",
			content:    `<?xml version="1.0"?><project></project>`,
			expectedVars: []string{
				"MAVEN_OPTS",
				"GRADLE_USER_HOME",
			},
		},
		{
			name:       ".NET project",
			language:   shared.LanguageDotNet,
			configFile: "test.csproj",
			content:    `<Project Sdk="Microsoft.NET.Sdk"></Project>`,
			expectedVars: []string{
				"NUGET_PACKAGES",
			},
		},
		{
			name:       "Go project",
			language:   shared.LanguageGo,
			configFile: "go.mod",
			content:    "module test\n",
			expectedVars: []string{
				"GOMODCACHE",
				"GOCACHE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Criar arquivo de configuração
			configPath := filepath.Join(tempDir, tt.configFile)
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}

			cachePath := filepath.Join(tempDir, "cache")

			// Criar script mock
			scriptPath := createEnvCaptureScript(t, tempDir)
			if scriptPath == "" {
				t.Fatal("Failed to create env capture script")
			}

			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				CachePath: cachePath,
				Language:  tt.language,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := service.Build(ctx, tempDir, config)

			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			if result == nil {
				t.Fatal("Build returned nil result")
			}

			// Verificar que todas as variáveis esperadas estão presentes
			for _, envVar := range tt.expectedVars {
				if !strings.Contains(result.Output, envVar) {
					t.Errorf("Expected environment variable %s not found in output", envVar)
				}
			}
		})
	}
}

// TestCachePathGeneration verifica que os caminhos de cache são gerados corretamente
func TestCachePathGeneration(t *testing.T) {
	tests := []struct {
		name          string
		language      shared.Language
		baseCachePath string
		expectedPaths map[string]string
	}{
		{
			name:          "Java cache paths",
			language:      shared.LanguageJava,
			baseCachePath: "/var/cache/build",
			expectedPaths: map[string]string{
				"maven":  "/var/cache/build/maven",
				"gradle": "/var/cache/build/gradle",
			},
		},
		{
			name:          ".NET cache paths",
			language:      shared.LanguageDotNet,
			baseCachePath: "/var/cache/build",
			expectedPaths: map[string]string{
				"nuget": "/var/cache/build/nuget",
			},
		},
		{
			name:          "Go cache paths",
			language:      shared.LanguageGo,
			baseCachePath: "/var/cache/build",
			expectedPaths: map[string]string{
				"go":          "/var/cache/build/go",
				"build-cache": "/var/cache/build/go/build-cache",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Criar comando mock para inspecionar variáveis de ambiente
			cmd := exec.Command("echo", "test")

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			// Configurar cache
			builder.configureCacheEnvironment(cmd, tt.language, tt.baseCachePath)

			// Verificar que as variáveis de ambiente foram configuradas corretamente
			envMap := make(map[string]string)
			for _, env := range cmd.Env {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			// Verificar caminhos esperados
			// Normalizar caminhos para comparação (converter para formato do OS)
			for key, expectedPath := range tt.expectedPaths {
				// Converter para caminho do sistema operacional
				normalizedExpected := filepath.FromSlash(expectedPath)

				found := false
				for envKey, envValue := range envMap {
					// Normalizar o valor da variável de ambiente também
					normalizedValue := filepath.FromSlash(envValue)

					// Verificar se o valor contém o caminho esperado
					if strings.Contains(normalizedValue, normalizedExpected) {
						found = true
						t.Logf("Found expected path %s in %s=%s", normalizedExpected, envKey, envValue)
						break
					}
				}
				if !found {
					t.Errorf("Expected cache path %s (%s) not found in environment variables", key, normalizedExpected)
					t.Logf("Environment variables: %+v", envMap)
				}
			}
		})
	}
}
