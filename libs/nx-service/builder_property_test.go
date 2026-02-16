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
