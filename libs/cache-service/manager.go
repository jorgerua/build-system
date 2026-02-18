package cacheservice

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jorgerua/build-system/libs/shared"
	"go.uber.org/zap"
)

// CacheService define a interface para gerenciamento de cache
type CacheService interface {
	GetCachePath(language shared.Language) string
	InitializeCache(language shared.Language) error
	CleanCache(language shared.Language, olderThan time.Duration) error
	GetCacheSize(language shared.Language) (int64, error)
}

// cacheManager implementa CacheService
type cacheManager struct {
	basePath string
	logger   *zap.Logger
}

// NewCacheService cria uma nova instância de CacheService
func NewCacheService(basePath string, logger *zap.Logger) CacheService {
	return &cacheManager{
		basePath: basePath,
		logger:   logger,
	}
}

// GetCachePath retorna o caminho do cache para uma linguagem específica
func (cm *cacheManager) GetCachePath(language shared.Language) string {
	switch language {
	case shared.LanguageJava:
		// Suporta tanto Maven quanto Gradle
		return filepath.Join(cm.basePath, "maven")
	case shared.LanguageDotNet:
		return filepath.Join(cm.basePath, "nuget")
	case shared.LanguageGo:
		return filepath.Join(cm.basePath, "go")
	default:
		return ""
	}
}

// InitializeCache cria a estrutura de diretórios para o cache de uma linguagem
func (cm *cacheManager) InitializeCache(language shared.Language) error {
	if !language.IsSupported() {
		return fmt.Errorf("unsupported language: %s", language)
	}

	cachePath := cm.GetCachePath(language)
	if cachePath == "" {
		return fmt.Errorf("no cache path defined for language: %s", language)
	}

	// Criar diretório com permissões apropriadas
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		cm.logger.Error("failed to create cache directory",
			zap.String("language", string(language)),
			zap.String("path", cachePath),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Para Java, criar também o diretório Gradle
	if language == shared.LanguageJava {
		gradlePath := filepath.Join(cm.basePath, "gradle")
		if err := os.MkdirAll(gradlePath, 0755); err != nil {
			cm.logger.Error("failed to create gradle cache directory",
				zap.String("path", gradlePath),
				zap.Error(err),
			)
			return fmt.Errorf("failed to create gradle cache directory: %w", err)
		}
	}

	cm.logger.Info("cache initialized",
		zap.String("language", string(language)),
		zap.String("path", cachePath),
	)

	return nil
}

// CleanCache remove arquivos de cache mais antigos que a duração especificada
func (cm *cacheManager) CleanCache(language shared.Language, olderThan time.Duration) error {
	if !language.IsSupported() {
		return fmt.Errorf("unsupported language: %s", language)
	}

	cachePath := cm.GetCachePath(language)
	if cachePath == "" {
		return fmt.Errorf("no cache path defined for language: %s", language)
	}

	// Verificar se o diretório existe
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		cm.logger.Warn("cache directory does not exist",
			zap.String("language", string(language)),
			zap.String("path", cachePath),
		)
		return nil
	}

	cutoffTime := time.Now().Add(-olderThan)
	var removedCount int
	var removedSize int64

	err := filepath.WalkDir(cachePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Pular o diretório raiz
		if path == cachePath {
			return nil
		}

		// Obter informações do arquivo
		info, err := d.Info()
		if err != nil {
			cm.logger.Warn("failed to get file info",
				zap.String("path", path),
				zap.Error(err),
			)
			return nil // Continuar com outros arquivos
		}

		// Remover se for mais antigo que o cutoff
		if info.ModTime().Before(cutoffTime) {
			size := info.Size()
			if err := os.RemoveAll(path); err != nil {
				cm.logger.Warn("failed to remove old cache file",
					zap.String("path", path),
					zap.Error(err),
				)
				return nil // Continuar com outros arquivos
			}

			removedCount++
			removedSize += size

			cm.logger.Debug("removed old cache file",
				zap.String("path", path),
				zap.Time("mod_time", info.ModTime()),
			)

			// Se removemos um diretório, pular seus filhos
			if d.IsDir() {
				return fs.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		cm.logger.Error("failed to clean cache",
			zap.String("language", string(language)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to clean cache: %w", err)
	}

	cm.logger.Info("cache cleaned",
		zap.String("language", string(language)),
		zap.Int("removed_count", removedCount),
		zap.Int64("removed_size_bytes", removedSize),
		zap.Duration("older_than", olderThan),
	)

	return nil
}

// GetCacheSize calcula o tamanho total do cache para uma linguagem
func (cm *cacheManager) GetCacheSize(language shared.Language) (int64, error) {
	if !language.IsSupported() {
		return 0, fmt.Errorf("unsupported language: %s", language)
	}

	cachePath := cm.GetCachePath(language)
	if cachePath == "" {
		return 0, fmt.Errorf("no cache path defined for language: %s", language)
	}

	// Verificar se o diretório existe
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return 0, nil
	}

	var totalSize int64

	err := filepath.WalkDir(cachePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Pular diretórios
		if d.IsDir() {
			return nil
		}

		// Obter informações do arquivo
		info, err := d.Info()
		if err != nil {
			cm.logger.Warn("failed to get file info",
				zap.String("path", path),
				zap.Error(err),
			)
			return nil // Continuar com outros arquivos
		}

		totalSize += info.Size()
		return nil
	})

	if err != nil {
		cm.logger.Error("failed to calculate cache size",
			zap.String("language", string(language)),
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to calculate cache size: %w", err)
	}

	// Para Java, incluir também o cache do Gradle
	if language == shared.LanguageJava {
		gradlePath := filepath.Join(cm.basePath, "gradle")
		if _, err := os.Stat(gradlePath); err == nil {
			gradleSize, err := cm.calculateDirectorySize(gradlePath)
			if err != nil {
				cm.logger.Warn("failed to calculate gradle cache size",
					zap.Error(err),
				)
			} else {
				totalSize += gradleSize
			}
		}
	}

	cm.logger.Debug("cache size calculated",
		zap.String("language", string(language)),
		zap.Int64("size_bytes", totalSize),
	)

	return totalSize, nil
}

// calculateDirectorySize é uma função auxiliar para calcular o tamanho de um diretório
func (cm *cacheManager) calculateDirectorySize(dirPath string) (int64, error) {
	var size int64

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // Continuar com outros arquivos
		}

		size += info.Size()
		return nil
	})

	return size, err
}
