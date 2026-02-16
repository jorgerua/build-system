package nxservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/oci-build-system/libs/shared"
	"go.uber.org/zap"
)

// Feature: oci-build-system, Property 6: Captura de saída de build
// Valida: Requisitos 3.2
//
// Para qualquer execução de build NX, tanto stdout quanto stderr devem ser
// capturados completamente e armazenados no BuildResult.
func TestProperty_BuildOutputCapture(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("captures stdout and stderr from build execution", prop.ForAll(
		func(stdoutContent string, stderrContent string) bool {
			// Criar diretório temporário para o teste
			tempDir := t.TempDir()

			// Criar arquivo go.mod para detecção de linguagem
			goModPath := filepath.Join(tempDir, "go.mod")
			if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
				t.Logf("Failed to create go.mod: %v", err)
				return false
			}

			// Criar um script que simula o comando nx
			// Este script irá produzir saída específica em stdout e stderr
			scriptPath := createMockNxScript(t, tempDir, stdoutContent, stderrContent)
			if scriptPath == "" {
				return false
			}

			// Substituir temporariamente o PATH para usar nosso script mock
			oldPath := os.Getenv("PATH")
			scriptDir := filepath.Dir(scriptPath)
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				Language: shared.LanguageGo,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			// Verificar que o resultado não é nil
			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Propriedade: stdout deve ser capturado completamente
			if !strings.Contains(result.Output, stdoutContent) {
				t.Logf("Expected stdout to contain '%s', got '%s'", stdoutContent, result.Output)
				return false
			}

			// Propriedade: stderr deve ser capturado completamente
			if !strings.Contains(result.ErrorOutput, stderrContent) {
				t.Logf("Expected stderr to contain '%s', got '%s'", stderrContent, result.ErrorOutput)
				return false
			}

			return true
		},
		// Gerar strings aleatórias para stdout
		gen.AlphaString().SuchThat(func(s string) bool {
			return len(s) > 0 && len(s) < 100
		}),
		// Gerar strings aleatórias para stderr
		gen.AlphaString().SuchThat(func(s string) bool {
			return len(s) > 0 && len(s) < 100
		}),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// createMockNxScript cria um script que simula o comando nx
// O script produz saída específica em stdout e stderr
func createMockNxScript(t *testing.T, baseDir, stdoutContent, stderrContent string) string {
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
		scriptContent = fmt.Sprintf(`@echo off
echo %s
echo %s 1>&2
exit /b 0
`, stdoutContent, stderrContent)
	} else {
		scriptPath = filepath.Join(scriptDir, "nx")
		scriptContent = fmt.Sprintf(`#!/bin/sh
echo "%s"
echo "%s" >&2
exit 0
`, stdoutContent, stderrContent)
	}

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Logf("Failed to create mock script: %v", err)
		return ""
	}

	return scriptPath
}

// isWindows verifica se o sistema operacional é Windows
func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

// Feature: oci-build-system, Property 6: Captura de saída de build
// Valida: Requisitos 3.2
//
// Teste adicional: verifica que saídas longas são capturadas completamente
func TestProperty_BuildOutputCapture_LongOutput(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("captures long stdout and stderr completely", prop.ForAll(
		func(lineCount uint) bool {
			// Limitar o número de linhas para evitar testes muito longos
			if lineCount == 0 || lineCount > 50 {
				return true // Skip valores fora do range
			}

			tempDir := t.TempDir()

			// Criar arquivo go.mod
			goModPath := filepath.Join(tempDir, "go.mod")
			if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
				t.Logf("Failed to create go.mod: %v", err)
				return false
			}

			// Gerar conteúdo com múltiplas linhas - usar formato simples
			var stdoutBuilder, stderrBuilder strings.Builder
			for i := uint(0); i < lineCount; i++ {
				stdoutBuilder.WriteString(fmt.Sprintf("OUT%d ", i))
				stderrBuilder.WriteString(fmt.Sprintf("ERR%d ", i))
			}
			stdoutContent := stdoutBuilder.String()
			stderrContent := stderrBuilder.String()

			// Criar script mock
			scriptDir := filepath.Join(tempDir, "bin")
			if err := os.MkdirAll(scriptDir, 0755); err != nil {
				t.Logf("Failed to create script directory: %v", err)
				return false
			}

			var scriptPath string
			var scriptContent string

			if isWindows() {
				scriptPath = filepath.Join(scriptDir, "nx.bat")
				scriptContent = fmt.Sprintf(`@echo off
echo %s
echo %s 1>&2
exit /b 0
`, stdoutContent, stderrContent)
			} else {
				scriptPath = filepath.Join(scriptDir, "nx")
				scriptContent = fmt.Sprintf(`#!/bin/sh
echo "%s"
echo "%s" >&2
exit 0
`, stdoutContent, stderrContent)
			}

			if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
				t.Logf("Failed to create mock script: %v", err)
				return false
			}

			// Substituir PATH temporariamente
			oldPath := os.Getenv("PATH")
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				Language: shared.LanguageGo,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Verificar que os tokens foram capturados
			for i := uint(0); i < lineCount; i++ {
				expectedStdout := fmt.Sprintf("OUT%d", i)
				expectedStderr := fmt.Sprintf("ERR%d", i)

				if !strings.Contains(result.Output, expectedStdout) {
					t.Logf("Missing stdout token %s", expectedStdout)
					return false
				}

				if !strings.Contains(result.ErrorOutput, expectedStderr) {
					t.Logf("Missing stderr token %s", expectedStderr)
					return false
				}
			}

			return true
		},
		gen.UIntRange(1, 50),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: oci-build-system, Property 6: Captura de saída de build
// Valida: Requisitos 3.2
//
// Teste adicional: verifica que caracteres especiais são capturados corretamente
func TestProperty_BuildOutputCapture_SpecialCharacters(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("captures special characters in output", prop.ForAll(
		func(specialChar rune) bool {
			// Filtrar caracteres que podem causar problemas em scripts
			if specialChar == 0 || specialChar == '\n' || specialChar == '\r' {
				return true // Skip
			}

			tempDir := t.TempDir()

			// Criar arquivo go.mod
			goModPath := filepath.Join(tempDir, "go.mod")
			if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
				t.Logf("Failed to create go.mod: %v", err)
				return false
			}

			// Criar conteúdo com caractere especial
			stdoutContent := fmt.Sprintf("output with special char: %c", specialChar)
			stderrContent := fmt.Sprintf("error with special char: %c", specialChar)

			// Criar script mock
			scriptPath := createMockNxScript(t, tempDir, stdoutContent, stderrContent)
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
				Language: shared.LanguageGo,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, _ := service.Build(ctx, tempDir, config)

			if result == nil {
				t.Log("Build returned nil result")
				return false
			}

			// Verificar que o caractere especial foi capturado
			if !strings.Contains(result.Output, string(specialChar)) {
				t.Logf("Special character %c not found in stdout", specialChar)
				return false
			}

			if !strings.Contains(result.ErrorOutput, string(specialChar)) {
				t.Logf("Special character %c not found in stderr", specialChar)
				return false
			}

			return true
		},
		gen.Rune(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: oci-build-system, Property 6: Captura de saída de build
// Valida: Requisitos 3.2
//
// Teste adicional: verifica que saída vazia é tratada corretamente
func TestProperty_BuildOutputCapture_EmptyOutput(t *testing.T) {
	tempDir := t.TempDir()

	// Criar arquivo go.mod
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Criar script que não produz saída
	scriptPath := createMockNxScript(t, tempDir, "", "")
	if scriptPath == "" {
		t.Fatal("Failed to create mock script")
	}

	// Substituir PATH temporariamente
	oldPath := os.Getenv("PATH")
	scriptDir := filepath.Dir(scriptPath)
	os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	logger := zap.NewNop()
	service := NewNXService(logger)

	config := BuildConfig{
		Language: shared.LanguageGo,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := service.Build(ctx, tempDir, config)

	// Verificar que não há erro
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verificar que o resultado não é nil
	if result == nil {
		t.Fatal("Build returned nil result")
	}

	// Propriedade: campos de saída devem existir (podem estar vazios)
	// Verificar que são strings válidas
	_ = result.Output
	_ = result.ErrorOutput

	// Verificar que Success é true quando não há erro
	if !result.Success {
		t.Error("Expected Success to be true for successful build with empty output")
	}
}

// Feature: oci-build-system, Property 6: Captura de saída de build
// Valida: Requisitos 3.2
//
// Teste adicional: verifica que saída é capturada mesmo quando o comando falha
func TestProperty_BuildOutputCapture_OnFailure(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("captures output even when build fails", prop.ForAll(
		func(exitCode uint8, errorMsg string) bool {
			// Usar apenas códigos de erro válidos (1-255)
			if exitCode == 0 {
				exitCode = 1
			}

			// Garantir que errorMsg não está vazio
			if len(errorMsg) == 0 {
				errorMsg = "build failed"
			}

			tempDir := t.TempDir()

			// Criar arquivo go.mod
			goModPath := filepath.Join(tempDir, "go.mod")
			if err := os.WriteFile(goModPath, []byte("module test\n"), 0644); err != nil {
				t.Logf("Failed to create go.mod: %v", err)
				return false
			}

			// Criar script que falha
			scriptDir := filepath.Join(tempDir, "bin")
			if err := os.MkdirAll(scriptDir, 0755); err != nil {
				t.Logf("Failed to create script directory: %v", err)
				return false
			}

			var scriptPath string
			var scriptContent string

			if isWindows() {
				scriptPath = filepath.Join(scriptDir, "nx.bat")
				scriptContent = fmt.Sprintf(`@echo off
echo Build output before failure
echo %s 1>&2
exit /b %d
`, errorMsg, exitCode)
			} else {
				scriptPath = filepath.Join(scriptDir, "nx")
				scriptContent = fmt.Sprintf(`#!/bin/sh
echo "Build output before failure"
echo "%s" >&2
exit %d
`, errorMsg, exitCode)
			}

			if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
				t.Logf("Failed to create mock script: %v", err)
				return false
			}

			// Substituir PATH temporariamente
			oldPath := os.Getenv("PATH")
			os.Setenv("PATH", scriptDir+string(os.PathListSeparator)+oldPath)
			defer os.Setenv("PATH", oldPath)

			logger := zap.NewNop()
			service := NewNXService(logger)

			config := BuildConfig{
				Language: shared.LanguageGo,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := service.Build(ctx, tempDir, config)

			// Deve retornar erro quando o build falha
			if err == nil {
				t.Log("Expected error when build fails")
				return false
			}

			// Mas o resultado ainda deve ser retornado
			if result == nil {
				t.Log("Build returned nil result even on failure")
				return false
			}

			// Propriedade: Success deve ser false
			if result.Success {
				t.Log("Expected Success to be false when build fails")
				return false
			}

			// Propriedade: stdout deve ser capturado
			if !strings.Contains(result.Output, "Build output before failure") {
				t.Logf("Expected stdout to be captured on failure, got '%s'", result.Output)
				return false
			}

			// Propriedade: stderr deve conter a mensagem de erro
			if !strings.Contains(result.ErrorOutput, errorMsg) {
				t.Logf("Expected stderr to contain error message '%s', got '%s'", errorMsg, result.ErrorOutput)
				return false
			}

			return true
		},
		gen.UInt8(),
		gen.AlphaString().SuchThat(func(s string) bool {
			return len(s) > 0 && len(s) < 100
		}),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: oci-build-system, Property 11: Detecção automática de linguagem
// **Valida: Requisitos 6.4**
//
// Para qualquer repositório contendo arquivos de configuração de linguagem
// (pom.xml, build.gradle, *.csproj, go.mod), o sistema deve detectar
// corretamente a linguagem correspondente.
func TestProperty_LanguageDetection(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("detects Java from pom.xml", prop.ForAll(
		func(seed uint64) bool {
			tempDir := t.TempDir()

			// Criar pom.xml no diretório raiz
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

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			lang, err := builder.detectLanguage(tempDir)

			// Propriedade: deve detectar Java sem erro
			if err != nil {
				t.Logf("Unexpected error detecting Java: %v", err)
				return false
			}

			// Propriedade: linguagem detectada deve ser Java
			if lang != shared.LanguageJava {
				t.Logf("Expected Java, got %s", lang)
				return false
			}

			return true
		},
		gen.UInt64(),
	))

	properties.Property("detects Java from build.gradle", prop.ForAll(
		func(hasKotlinExt bool) bool {
			tempDir := t.TempDir()

			// Criar build.gradle ou build.gradle.kts
			var gradleFile string
			if hasKotlinExt {
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

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			lang, err := builder.detectLanguage(tempDir)

			// Propriedade: deve detectar Java sem erro
			if err != nil {
				t.Logf("Unexpected error detecting Java from %s: %v", gradleFile, err)
				return false
			}

			// Propriedade: linguagem detectada deve ser Java
			if lang != shared.LanguageJava {
				t.Logf("Expected Java from %s, got %s", gradleFile, lang)
				return false
			}

			return true
		},
		gen.Bool(),
	))

	properties.Property("detects .NET from .csproj files", prop.ForAll(
		func(seed uint64) bool {
			tempDir := t.TempDir()

			// Gerar nome de projeto baseado no seed
			projectName := fmt.Sprintf("Project%d", seed%1000)

			// Criar arquivo .csproj
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

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			lang, err := builder.detectLanguage(tempDir)

			// Propriedade: deve detectar .NET sem erro
			if err != nil {
				t.Logf("Unexpected error detecting .NET: %v", err)
				return false
			}

			// Propriedade: linguagem detectada deve ser .NET
			if lang != shared.LanguageDotNet {
				t.Logf("Expected .NET, got %s", lang)
				return false
			}

			return true
		},
		gen.UInt64(),
	))

	properties.Property("detects Go from go.mod", prop.ForAll(
		func(seed uint64) bool {
			tempDir := t.TempDir()

			// Gerar nome de módulo baseado no seed
			moduleName := fmt.Sprintf("github.com/example/module%d", seed%1000)

			// Criar go.mod
			goModPath := filepath.Join(tempDir, "go.mod")
			goModContent := fmt.Sprintf("module %s\n\ngo 1.21\n", moduleName)
			if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
				t.Logf("Failed to create go.mod: %v", err)
				return false
			}

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			lang, err := builder.detectLanguage(tempDir)

			// Propriedade: deve detectar Go sem erro
			if err != nil {
				t.Logf("Unexpected error detecting Go: %v", err)
				return false
			}

			// Propriedade: linguagem detectada deve ser Go
			if lang != shared.LanguageGo {
				t.Logf("Expected Go, got %s", lang)
				return false
			}

			return true
		},
		gen.UInt64(),
	))

	properties.Property("detects language in subdirectories", prop.ForAll(
		func(langChoice uint8) bool {
			// Usar apenas subdiretórios comuns
			commonDirs := []string{"apps", "libs", "packages", "src"}
			subdirName := commonDirs[int(langChoice)%len(commonDirs)]

			tempDir := t.TempDir()

			// Criar subdiretório com nome baseado no langChoice
			projectName := fmt.Sprintf("project%d", langChoice)
			subdirPath := filepath.Join(tempDir, subdirName, projectName)
			if err := os.MkdirAll(subdirPath, 0755); err != nil {
				t.Logf("Failed to create subdirectory: %v", err)
				return false
			}

			// Escolher linguagem baseado em langChoice
			var expectedLang shared.Language
			var configFile string
			var content string

			switch langChoice % 3 {
			case 0: // Java
				expectedLang = shared.LanguageJava
				configFile = "pom.xml"
				content = `<?xml version="1.0"?><project></project>`
			case 1: // .NET
				expectedLang = shared.LanguageDotNet
				configFile = "test.csproj"
				content = `<Project Sdk="Microsoft.NET.Sdk"></Project>`
			case 2: // Go
				expectedLang = shared.LanguageGo
				configFile = "go.mod"
				content = "module test\n"
			}

			configPath := filepath.Join(subdirPath, configFile)
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Logf("Failed to create config file: %v", err)
				return false
			}

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			lang, err := builder.detectLanguage(tempDir)

			// Propriedade: deve detectar linguagem em subdiretório
			if err != nil {
				t.Logf("Failed to detect language in subdirectory: %v", err)
				return false
			}

			// Propriedade: linguagem detectada deve corresponder ao arquivo criado
			if lang != expectedLang {
				t.Logf("Expected %s in subdirectory, got %s", expectedLang, lang)
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.Property("returns unknown for directories without config files", prop.ForAll(
		func(numFiles uint8) bool {
			// Limitar número de arquivos
			if numFiles > 10 {
				numFiles = 10
			}

			tempDir := t.TempDir()

			// Criar arquivos aleatórios que não são arquivos de configuração
			for i := uint8(0); i < numFiles; i++ {
				filename := fmt.Sprintf("file%d.txt", i)
				filepath := filepath.Join(tempDir, filename)
				if err := os.WriteFile(filepath, []byte("random content"), 0644); err != nil {
					t.Logf("Failed to create file: %v", err)
					return false
				}
			}

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			lang, err := builder.detectLanguage(tempDir)

			// Propriedade: deve retornar erro quando nenhuma linguagem é detectada
			if err == nil {
				t.Log("Expected error when no language config found")
				return false
			}

			// Propriedade: linguagem deve ser Unknown
			if lang != shared.LanguageUnknown {
				t.Logf("Expected Unknown language, got %s", lang)
				return false
			}

			return true
		},
		gen.UInt8Range(0, 10),
	))

	properties.Property("prioritizes root directory over subdirectories", prop.ForAll(
		func(rootLang uint8, subdirLang uint8) bool {
			// Garantir que são linguagens diferentes
			if rootLang%3 == subdirLang%3 {
				return true // Skip se forem iguais
			}

			tempDir := t.TempDir()

			// Criar arquivo de configuração no diretório raiz
			var rootExpectedLang shared.Language
			var rootConfigFile string
			var rootContent string

			switch rootLang % 3 {
			case 0: // Java
				rootExpectedLang = shared.LanguageJava
				rootConfigFile = "pom.xml"
				rootContent = `<?xml version="1.0"?><project></project>`
			case 1: // .NET
				rootExpectedLang = shared.LanguageDotNet
				rootConfigFile = "root.csproj"
				rootContent = `<Project Sdk="Microsoft.NET.Sdk"></Project>`
			case 2: // Go
				rootExpectedLang = shared.LanguageGo
				rootConfigFile = "go.mod"
				rootContent = "module root\n"
			}

			rootConfigPath := filepath.Join(tempDir, rootConfigFile)
			if err := os.WriteFile(rootConfigPath, []byte(rootContent), 0644); err != nil {
				t.Logf("Failed to create root config: %v", err)
				return false
			}

			// Criar arquivo de configuração diferente em subdiretório
			subdirPath := filepath.Join(tempDir, "apps")
			if err := os.MkdirAll(subdirPath, 0755); err != nil {
				t.Logf("Failed to create subdirectory: %v", err)
				return false
			}

			var subdirConfigFile string
			var subdirContent string

			switch subdirLang % 3 {
			case 0: // Java
				subdirConfigFile = "pom.xml"
				subdirContent = `<?xml version="1.0"?><project></project>`
			case 1: // .NET
				subdirConfigFile = "subdir.csproj"
				subdirContent = `<Project Sdk="Microsoft.NET.Sdk"></Project>`
			case 2: // Go
				subdirConfigFile = "go.mod"
				subdirContent = "module subdir\n"
			}

			subdirConfigPath := filepath.Join(subdirPath, subdirConfigFile)
			if err := os.WriteFile(subdirConfigPath, []byte(subdirContent), 0644); err != nil {
				t.Logf("Failed to create subdir config: %v", err)
				return false
			}

			logger := zap.NewNop()
			builder := &nxBuilder{logger: logger}

			lang, err := builder.detectLanguage(tempDir)

			// Propriedade: deve detectar linguagem do diretório raiz
			if err != nil {
				t.Logf("Unexpected error: %v", err)
				return false
			}

			// Propriedade: deve priorizar arquivo do diretório raiz
			if lang != rootExpectedLang {
				t.Logf("Expected root language %s, got %s", rootExpectedLang, lang)
				return false
			}

			return true
		},
		gen.UInt8(),
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
