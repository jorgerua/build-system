package cacheservice

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/oci-build-system/libs/shared"
	"go.uber.org/zap"
)

// Feature: oci-build-system, Property 10: Persistência de dependências em cache
// Para qualquer build que baixe dependências, essas dependências devem ser armazenadas
// no diretório de cache apropriado para a linguagem e estar disponíveis para builds subsequentes.
// Valida: Requisitos 4.3, 4.5
func TestProperty_CachePersistence(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("dependencies stored in cache are available for subsequent builds", prop.ForAll(
		func(language shared.Language, fileCount uint8, fileSize uint16) bool {
			// Limitar fileCount para evitar criar muitos arquivos
			if fileCount == 0 {
				fileCount = 1
			}
			if fileCount > 20 {
				fileCount = 20
			}

			// Limitar fileSize para evitar arquivos muito grandes
			if fileSize == 0 {
				fileSize = 100
			}
			if fileSize > 10000 {
				fileSize = 10000
			}

			// Criar diretório temporário para este teste
			tmpDir, err := os.MkdirTemp("", "cache-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)

			logger := zap.NewNop()
			cm := NewCacheService(tmpDir, logger)

			// 1. Inicializar cache para a linguagem
			if err := cm.InitializeCache(language); err != nil {
				// Linguagens não suportadas devem retornar erro
				return !language.IsSupported()
			}

			// Se chegamos aqui, a linguagem é suportada
			if !language.IsSupported() {
				t.Logf("Expected error for unsupported language %s", language)
				return false
			}

			cachePath := cm.GetCachePath(language)
			if cachePath == "" {
				t.Logf("Cache path is empty for supported language %s", language)
				return false
			}

			// 2. Simular download de dependências criando arquivos no cache
			createdFiles := make([]string, 0, fileCount)
			var expectedTotalSize int64

			for i := uint8(0); i < fileCount; i++ {
				// Criar arquivo com nome único
				fileName := filepath.Join(cachePath, "dep", "group", "artifact", "version", "file"+string(rune('a'+i))+".jar")
				
				// Criar diretórios necessários
				if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
					t.Logf("Failed to create directory for %s: %v", fileName, err)
					return false
				}

				// Criar arquivo com conteúdo de tamanho específico
				content := make([]byte, fileSize)
				for j := range content {
					content[j] = byte(i + uint8(j%256))
				}

				if err := os.WriteFile(fileName, content, 0644); err != nil {
					t.Logf("Failed to write file %s: %v", fileName, err)
					return false
				}

				createdFiles = append(createdFiles, fileName)
				expectedTotalSize += int64(fileSize)
			}

			// 3. Verificar que os arquivos foram armazenados no cache
			for _, filePath := range createdFiles {
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Logf("File %s was not stored in cache", filePath)
					return false
				}
			}

			// 4. Verificar que o tamanho do cache reflete as dependências armazenadas
			cacheSize, err := cm.GetCacheSize(language)
			if err != nil {
				t.Logf("Failed to get cache size: %v", err)
				return false
			}

			// Para Java, o tamanho pode incluir o diretório Gradle vazio
			// então verificamos que o tamanho é pelo menos o esperado
			if cacheSize < expectedTotalSize {
				t.Logf("Cache size %d is less than expected %d", cacheSize, expectedTotalSize)
				return false
			}

			// 5. Simular um "build subsequente" verificando que os arquivos ainda estão disponíveis
			// Criar uma nova instância do cache service (simula reinicialização)
			cm2 := NewCacheService(tmpDir, logger)

			// Verificar que todos os arquivos ainda estão acessíveis
			for _, filePath := range createdFiles {
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Logf("File %s is not available for subsequent build", filePath)
					return false
				}

				// Verificar que o conteúdo está intacto
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Logf("Failed to read file %s: %v", filePath, err)
					return false
				}

				if len(content) != int(fileSize) {
					t.Logf("File %s has incorrect size: got %d, want %d", filePath, len(content), fileSize)
					return false
				}
			}

			// 6. Verificar que o tamanho do cache ainda é correto
			cacheSize2, err := cm2.GetCacheSize(language)
			if err != nil {
				t.Logf("Failed to get cache size on second instance: %v", err)
				return false
			}

			if cacheSize2 != cacheSize {
				t.Logf("Cache size changed between instances: %d vs %d", cacheSize, cacheSize2)
				return false
			}

			return true
		},
		// Gerar linguagens suportadas e não suportadas
		gen.OneConstOf(
			shared.LanguageJava,
			shared.LanguageDotNet,
			shared.LanguageGo,
			shared.LanguageUnknown,
			shared.Language("invalid"),
		),
		gen.UInt8(),    // fileCount
		gen.UInt16(),   // fileSize
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: oci-build-system, Property 10: Persistência de dependências em cache (variante com limpeza)
// Verifica que dependências persistem mesmo após limpeza de cache antigo
func TestProperty_CachePersistence_WithCleanup(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("recent dependencies persist after cache cleanup", prop.ForAll(
		func(language shared.Language, fileCount uint8) bool {
			// Limitar fileCount
			if fileCount == 0 {
				fileCount = 1
			}
			if fileCount > 10 {
				fileCount = 10
			}

			// Criar diretório temporário
			tmpDir, err := os.MkdirTemp("", "cache-cleanup-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)

			logger := zap.NewNop()
			cm := NewCacheService(tmpDir, logger)

			// Inicializar cache
			if err := cm.InitializeCache(language); err != nil {
				return !language.IsSupported()
			}

			if !language.IsSupported() {
				return false
			}

			cachePath := cm.GetCachePath(language)

			// Criar arquivos recentes (dependências do build atual)
			recentFiles := make([]string, 0, fileCount)
			for i := uint8(0); i < fileCount; i++ {
				fileName := filepath.Join(cachePath, "recent", "dep"+string(rune('a'+i))+".jar")
				if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
					t.Logf("Failed to create directory: %v", err)
					return false
				}
				if err := os.WriteFile(fileName, []byte("recent"), 0644); err != nil {
					t.Logf("Failed to write recent file: %v", err)
					return false
				}
				recentFiles = append(recentFiles, fileName)
			}

			// Criar alguns arquivos antigos
			oldFiles := make([]string, 0, 3)
			for i := 0; i < 3; i++ {
				fileName := filepath.Join(cachePath, "old", "dep"+string(rune('x'+i))+".jar")
				if err := os.MkdirAll(filepath.Dir(fileName), 0755); err != nil {
					t.Logf("Failed to create directory: %v", err)
					return false
				}
				if err := os.WriteFile(fileName, []byte("old"), 0644); err != nil {
					t.Logf("Failed to write old file: %v", err)
					return false
				}
				
				// Modificar timestamp para 48 horas atrás
				oldTime := time.Now().Add(-48 * time.Hour)
				if err := os.Chtimes(fileName, oldTime, oldTime); err != nil {
					t.Logf("Failed to change file time: %v", err)
					return false
				}
				
				oldFiles = append(oldFiles, fileName)
			}

			// Executar limpeza de cache (remover arquivos mais antigos que 24 horas)
			if err := cm.CleanCache(language, 24*time.Hour); err != nil {
				t.Logf("Failed to clean cache: %v", err)
				return false
			}

			// Verificar que arquivos recentes ainda existem
			for _, filePath := range recentFiles {
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Logf("Recent file %s was incorrectly removed", filePath)
					return false
				}
			}

			// Verificar que arquivos antigos foram removidos
			for _, filePath := range oldFiles {
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Logf("Old file %s was not removed", filePath)
					return false
				}
			}

			// Verificar que cache ainda funciona após limpeza
			cacheSize, err := cm.GetCacheSize(language)
			if err != nil {
				t.Logf("Failed to get cache size after cleanup: %v", err)
				return false
			}

			// Tamanho deve ser maior que zero (arquivos recentes)
			if cacheSize == 0 {
				t.Logf("Cache size is zero after cleanup, but recent files should exist")
				return false
			}

			return true
		},
		gen.OneConstOf(
			shared.LanguageJava,
			shared.LanguageDotNet,
			shared.LanguageGo,
		),
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
