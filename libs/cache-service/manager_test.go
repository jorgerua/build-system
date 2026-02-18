package cacheservice

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jorgerua/build-system/libs/shared"
	"go.uber.org/zap"
)

// TestGetCachePath testa obtenção de paths de cache para diferentes linguagens
func TestGetCachePath(t *testing.T) {
	logger := zap.NewNop()
	basePath := "/var/cache/oci-build/deps"
	cm := NewCacheService(basePath, logger)

	tests := []struct {
		name     string
		language shared.Language
		expected string
	}{
		{
			"Java cache path",
			shared.LanguageJava,
			filepath.Join(basePath, "maven"),
		},
		{
			".NET cache path",
			shared.LanguageDotNet,
			filepath.Join(basePath, "nuget"),
		},
		{
			"Go cache path",
			shared.LanguageGo,
			filepath.Join(basePath, "go"),
		},
		{
			"Unknown language returns empty",
			shared.LanguageUnknown,
			"",
		},
		{
			"Invalid language returns empty",
			shared.Language("invalid"),
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cm.GetCachePath(tt.language)
			if got != tt.expected {
				t.Errorf("GetCachePath(%v) = %v, want %v", tt.language, got, tt.expected)
			}
		})
	}
}

// TestInitializeCache testa criação de estrutura de diretórios
func TestInitializeCache(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	tests := []struct {
		name     string
		language shared.Language
		wantErr  bool
	}{
		{
			"Initialize Java cache",
			shared.LanguageJava,
			false,
		},
		{
			"Initialize .NET cache",
			shared.LanguageDotNet,
			false,
		},
		{
			"Initialize Go cache",
			shared.LanguageGo,
			false,
		},
		{
			"Unsupported language returns error",
			shared.LanguageUnknown,
			true,
		},
		{
			"Invalid language returns error",
			shared.Language("invalid"),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.InitializeCache(tt.language)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitializeCache(%v) error = %v, wantErr %v", tt.language, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verificar que o diretório foi criado
				cachePath := cm.GetCachePath(tt.language)
				if _, err := os.Stat(cachePath); os.IsNotExist(err) {
					t.Errorf("Cache directory was not created: %s", cachePath)
				}

				// Para Java, verificar que o diretório Gradle também foi criado
				if tt.language == shared.LanguageJava {
					gradlePath := filepath.Join(tmpDir, "gradle")
					if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
						t.Errorf("Gradle cache directory was not created: %s", gradlePath)
					}
				}
			}
		})
	}
}

// TestInitializeCache_AlreadyExists testa que inicialização funciona mesmo se diretório já existe
func TestInitializeCache_AlreadyExists(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Criar diretório manualmente
	cachePath := filepath.Join(tmpDir, "maven")
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Inicializar cache deve funcionar sem erro
	err := cm.InitializeCache(shared.LanguageJava)
	if err != nil {
		t.Errorf("InitializeCache failed when directory already exists: %v", err)
	}
}

// TestGetCacheSize testa cálculo de tamanho de cache
func TestGetCacheSize(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Inicializar cache
	if err := cm.InitializeCache(shared.LanguageGo); err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	cachePath := cm.GetCachePath(shared.LanguageGo)

	// Criar alguns arquivos de teste
	testFiles := []struct {
		name string
		size int
	}{
		{"file1.txt", 100},
		{"file2.txt", 200},
		{"subdir/file3.txt", 300},
	}

	var expectedSize int64
	for _, tf := range testFiles {
		filePath := filepath.Join(cachePath, tf.name)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		content := make([]byte, tf.size)
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		expectedSize += int64(tf.size)
	}

	// Calcular tamanho
	size, err := cm.GetCacheSize(shared.LanguageGo)
	if err != nil {
		t.Errorf("GetCacheSize failed: %v", err)
	}

	if size != expectedSize {
		t.Errorf("GetCacheSize() = %d, want %d", size, expectedSize)
	}
}

// TestGetCacheSize_EmptyCache testa tamanho de cache vazio
func TestGetCacheSize_EmptyCache(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Inicializar cache vazio
	if err := cm.InitializeCache(shared.LanguageDotNet); err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	size, err := cm.GetCacheSize(shared.LanguageDotNet)
	if err != nil {
		t.Errorf("GetCacheSize failed: %v", err)
	}

	if size != 0 {
		t.Errorf("GetCacheSize() for empty cache = %d, want 0", size)
	}
}

// TestGetCacheSize_NonExistentCache testa tamanho de cache que não existe
func TestGetCacheSize_NonExistentCache(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Não inicializar cache, apenas calcular tamanho
	size, err := cm.GetCacheSize(shared.LanguageGo)
	if err != nil {
		t.Errorf("GetCacheSize failed for non-existent cache: %v", err)
	}

	if size != 0 {
		t.Errorf("GetCacheSize() for non-existent cache = %d, want 0", size)
	}
}

// TestGetCacheSize_Java_IncludesGradle testa que tamanho de cache Java inclui Gradle
func TestGetCacheSize_Java_IncludesGradle(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Inicializar cache Java
	if err := cm.InitializeCache(shared.LanguageJava); err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Criar arquivo no cache Maven
	mavenPath := filepath.Join(tmpDir, "maven", "file1.txt")
	mavenContent := make([]byte, 100)
	if err := os.WriteFile(mavenPath, mavenContent, 0644); err != nil {
		t.Fatalf("Failed to write maven file: %v", err)
	}

	// Criar arquivo no cache Gradle
	gradlePath := filepath.Join(tmpDir, "gradle", "file2.txt")
	gradleContent := make([]byte, 200)
	if err := os.WriteFile(gradlePath, gradleContent, 0644); err != nil {
		t.Fatalf("Failed to write gradle file: %v", err)
	}

	// Calcular tamanho
	size, err := cm.GetCacheSize(shared.LanguageJava)
	if err != nil {
		t.Errorf("GetCacheSize failed: %v", err)
	}

	expectedSize := int64(300) // 100 + 200
	if size != expectedSize {
		t.Errorf("GetCacheSize() = %d, want %d (should include both Maven and Gradle)", size, expectedSize)
	}
}

// TestGetCacheSize_UnsupportedLanguage testa erro com linguagem não suportada
func TestGetCacheSize_UnsupportedLanguage(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	_, err := cm.GetCacheSize(shared.LanguageUnknown)
	if err == nil {
		t.Error("Expected error for unsupported language, got nil")
	}
}

// TestCleanCache testa limpeza de arquivos antigos
func TestCleanCache(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Inicializar cache
	if err := cm.InitializeCache(shared.LanguageGo); err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	cachePath := cm.GetCachePath(shared.LanguageGo)

	// Criar arquivo antigo
	oldFile := filepath.Join(cachePath, "old.txt")
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatalf("Failed to write old file: %v", err)
	}
	// Modificar timestamp para 2 dias atrás
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to change file time: %v", err)
	}

	// Criar arquivo recente
	newFile := filepath.Join(cachePath, "new.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatalf("Failed to write new file: %v", err)
	}

	// Limpar arquivos mais antigos que 24 horas
	err := cm.CleanCache(shared.LanguageGo, 24*time.Hour)
	if err != nil {
		t.Errorf("CleanCache failed: %v", err)
	}

	// Verificar que arquivo antigo foi removido
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Old file was not removed")
	}

	// Verificar que arquivo recente ainda existe
	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		t.Error("New file was incorrectly removed")
	}
}

// TestCleanCache_EmptyCache testa limpeza de cache vazio
func TestCleanCache_EmptyCache(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Inicializar cache vazio
	if err := cm.InitializeCache(shared.LanguageDotNet); err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Limpar cache vazio não deve causar erro
	err := cm.CleanCache(shared.LanguageDotNet, 24*time.Hour)
	if err != nil {
		t.Errorf("CleanCache failed on empty cache: %v", err)
	}
}

// TestCleanCache_NonExistentCache testa limpeza de cache que não existe
func TestCleanCache_NonExistentCache(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Limpar cache que não existe não deve causar erro
	err := cm.CleanCache(shared.LanguageGo, 24*time.Hour)
	if err != nil {
		t.Errorf("CleanCache failed on non-existent cache: %v", err)
	}
}

// TestCleanCache_RemovesOldDirectories testa que diretórios antigos também são removidos
func TestCleanCache_RemovesOldDirectories(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// Inicializar cache
	if err := cm.InitializeCache(shared.LanguageGo); err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	cachePath := cm.GetCachePath(shared.LanguageGo)

	// Criar diretório antigo com arquivo dentro
	oldDir := filepath.Join(cachePath, "old-dir")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatalf("Failed to create old directory: %v", err)
	}
	oldFile := filepath.Join(oldDir, "file.txt")
	if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
		t.Fatalf("Failed to write file in old directory: %v", err)
	}

	// Modificar timestamp do diretório para 2 dias atrás
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to change directory time: %v", err)
	}

	// Limpar arquivos mais antigos que 24 horas
	err := cm.CleanCache(shared.LanguageGo, 24*time.Hour)
	if err != nil {
		t.Errorf("CleanCache failed: %v", err)
	}

	// Verificar que diretório antigo foi removido
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("Old directory was not removed")
	}
}

// TestCleanCache_UnsupportedLanguage testa erro com linguagem não suportada
func TestCleanCache_UnsupportedLanguage(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	err := cm.CleanCache(shared.LanguageUnknown, 24*time.Hour)
	if err == nil {
		t.Error("Expected error for unsupported language, got nil")
	}
}

// TestCacheService_CompleteWorkflow testa fluxo completo de operações de cache
func TestCacheService_CompleteWorkflow(t *testing.T) {
	logger := zap.NewNop()
	tmpDir := t.TempDir()
	cm := NewCacheService(tmpDir, logger)

	// 1. Inicializar cache
	if err := cm.InitializeCache(shared.LanguageJava); err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// 2. Verificar que diretórios foram criados
	mavenPath := cm.GetCachePath(shared.LanguageJava)
	if _, err := os.Stat(mavenPath); os.IsNotExist(err) {
		t.Error("Maven cache directory was not created")
	}
	gradlePath := filepath.Join(tmpDir, "gradle")
	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		t.Error("Gradle cache directory was not created")
	}

	// 3. Adicionar alguns arquivos
	file1 := filepath.Join(mavenPath, "dep1.jar")
	if err := os.WriteFile(file1, make([]byte, 1000), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	file2 := filepath.Join(gradlePath, "dep2.jar")
	if err := os.WriteFile(file2, make([]byte, 2000), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// 4. Calcular tamanho
	size, err := cm.GetCacheSize(shared.LanguageJava)
	if err != nil {
		t.Errorf("GetCacheSize failed: %v", err)
	}
	if size != 3000 {
		t.Errorf("GetCacheSize() = %d, want 3000", size)
	}

	// 5. Modificar um arquivo para ser antigo
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(file1, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to change file time: %v", err)
	}

	// 6. Limpar arquivos antigos
	if err := cm.CleanCache(shared.LanguageJava, 24*time.Hour); err != nil {
		t.Errorf("CleanCache failed: %v", err)
	}

	// 7. Verificar que arquivo antigo foi removido
	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Error("Old file was not removed")
	}

	// 8. Verificar que arquivo recente ainda existe
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Error("Recent file was incorrectly removed")
	}

	// 9. Calcular novo tamanho
	newSize, err := cm.GetCacheSize(shared.LanguageJava)
	if err != nil {
		t.Errorf("GetCacheSize failed: %v", err)
	}
	if newSize != 2000 {
		t.Errorf("GetCacheSize() after cleanup = %d, want 2000", newSize)
	}
}
