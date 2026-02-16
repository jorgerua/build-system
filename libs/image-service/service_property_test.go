package imageservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"go.uber.org/zap"
)

// Feature: oci-build-system, Property 13: Localização de Dockerfile
// Para qualquer repositório, o sistema deve buscar Dockerfile no diretório raiz
// e em subdiretórios comuns (./docker, ./build, etc.).
// Valida: Requisitos 5.2
func TestProperty_DockerfileLocalization(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Locais comuns onde Dockerfiles podem estar
	commonLocations := []string{
		"Dockerfile",
		"dockerfile",
		"docker/Dockerfile",
		"build/Dockerfile",
		".docker/Dockerfile",
		"deployment/Dockerfile",
	}

	properties.Property("locates Dockerfile in common locations", prop.ForAll(
		func(locationIndex uint8, hasValidContent bool) bool {
			// Selecionar uma localização da lista
			location := commonLocations[int(locationIndex)%len(commonLocations)]

			// Criar diretório temporário para este teste
			tmpDir, err := os.MkdirTemp("", "dockerfile-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)

			// Criar o Dockerfile na localização escolhida
			dockerfilePath := filepath.Join(tmpDir, location)
			
			// Criar diretórios necessários
			if err := os.MkdirAll(filepath.Dir(dockerfilePath), 0755); err != nil {
				t.Logf("Failed to create directory for %s: %v", dockerfilePath, err)
				return false
			}

			// Criar conteúdo do Dockerfile
			var content string
			if hasValidContent {
				content = "FROM alpine:latest\nRUN echo 'test'\n"
			} else {
				// Conteúdo inválido (sem FROM)
				content = "RUN echo 'test'\n"
			}

			if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
				t.Logf("Failed to write Dockerfile: %v", err)
				return false
			}

			// Criar serviço
			logger := zap.NewNop()
			svc := NewImageService(logger).(*imageService)

			// Tentar localizar o Dockerfile
			foundPath, err := svc.locateDockerfile(tmpDir, "")
			
			// Verificar que o Dockerfile foi encontrado
			if err != nil {
				t.Logf("Failed to locate Dockerfile at %s: %v", location, err)
				return false
			}

			// Verificar que o path encontrado corresponde ao esperado
			// No Windows, o filesystem é case-insensitive, então comparamos paths normalizados
			if !pathsEqual(foundPath, dockerfilePath) {
				t.Logf("Found path %s does not match expected %s", foundPath, dockerfilePath)
				return false
			}

			// Verificar que o arquivo existe
			if _, err := os.Stat(foundPath); os.IsNotExist(err) {
				t.Logf("Located Dockerfile does not exist: %s", foundPath)
				return false
			}

			return true
		},
		gen.UInt8(),
		gen.Bool(),
	))

	properties.Property("returns error when Dockerfile not found in any location", prop.ForAll(
		func(seed uint8) bool {
			// Criar diretório temporário vazio
			tmpDir, err := os.MkdirTemp("", "no-dockerfile-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)

			// Criar alguns arquivos que NÃO são Dockerfiles
			nonDockerfiles := []string{
				"README.md",
				"main.go",
				"package.json",
			}

			for _, filename := range nonDockerfiles {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
					t.Logf("Failed to write file %s: %v", filename, err)
					return false
				}
			}

			// Criar serviço
			logger := zap.NewNop()
			svc := NewImageService(logger).(*imageService)

			// Tentar localizar Dockerfile (deve falhar)
			_, err = svc.locateDockerfile(tmpDir, "")
			
			// Verificar que retornou erro
			if err == nil {
				t.Logf("Expected error when Dockerfile not found, got nil")
				return false
			}

			// Verificar mensagem de erro
			expectedMsg := "Dockerfile not found in common locations"
			if err.Error() != expectedMsg {
				t.Logf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.Property("uses specified dockerfile path when provided", prop.ForAll(
		func(customPath string, isRelative bool) bool {
			// Gerar path customizado válido
			if customPath == "" || customPath == "." || customPath == ".." {
				customPath = "custom/Dockerfile.prod"
			}
			
			// Limpar path para evitar caracteres inválidos
			customPath = filepath.Clean(customPath)
			
			// Garantir que termina com um nome de arquivo
			if filepath.Ext(customPath) == "" {
				customPath = filepath.Join(customPath, "Dockerfile")
			}

			// Criar diretório temporário
			tmpDir, err := os.MkdirTemp("", "custom-dockerfile-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)

			// Determinar path completo
			var fullPath string
			var pathToProvide string
			
			if isRelative {
				// Path relativo ao context
				fullPath = filepath.Join(tmpDir, customPath)
				pathToProvide = customPath
			} else {
				// Path absoluto
				fullPath = filepath.Join(tmpDir, customPath)
				pathToProvide = fullPath
			}

			// Criar diretórios necessários
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				t.Logf("Failed to create directory for %s: %v", fullPath, err)
				return false
			}

			// Criar Dockerfile
			content := "FROM alpine:latest\n"
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Logf("Failed to write Dockerfile: %v", err)
				return false
			}

			// Criar serviço
			logger := zap.NewNop()
			svc := NewImageService(logger).(*imageService)

			// Localizar usando path especificado
			foundPath, err := svc.locateDockerfile(tmpDir, pathToProvide)
			
			// Verificar que encontrou
			if err != nil {
				t.Logf("Failed to locate specified Dockerfile at %s: %v", pathToProvide, err)
				return false
			}

			// Verificar que o path encontrado é o esperado
			if foundPath != fullPath {
				t.Logf("Found path %s does not match expected %s", foundPath, fullPath)
				return false
			}

			return true
		},
		gen.Identifier(),
		gen.Bool(),
	))

	properties.Property("returns error when specified dockerfile does not exist", prop.ForAll(
		func(nonExistentPath string) bool {
			// Gerar path que não existe
			if nonExistentPath == "" || nonExistentPath == "." {
				nonExistentPath = "nonexistent/Dockerfile"
			}
			
			nonExistentPath = filepath.Clean(nonExistentPath)

			// Criar diretório temporário
			tmpDir, err := os.MkdirTemp("", "missing-dockerfile-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)

			// Criar serviço
			logger := zap.NewNop()
			svc := NewImageService(logger).(*imageService)

			// Tentar localizar Dockerfile inexistente
			_, err = svc.locateDockerfile(tmpDir, nonExistentPath)
			
			// Verificar que retornou erro
			if err == nil {
				t.Logf("Expected error when specified Dockerfile does not exist, got nil")
				return false
			}

			// Verificar que a mensagem de erro menciona o arquivo especificado
			expectedPrefix := "specified Dockerfile not found:"
			if len(err.Error()) < len(expectedPrefix) || err.Error()[:len(expectedPrefix)] != expectedPrefix {
				t.Logf("Expected error message starting with '%s', got '%s'", expectedPrefix, err.Error())
				return false
			}

			return true
		},
		gen.Identifier(),
	))

	properties.Property("prioritizes root Dockerfile over subdirectory Dockerfiles", prop.ForAll(
		func(seed uint8) bool {
			// Criar diretório temporário
			tmpDir, err := os.MkdirTemp("", "priority-test-*")
			if err != nil {
				t.Logf("Failed to create temp dir: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)

			// Criar Dockerfile na raiz
			rootDockerfile := filepath.Join(tmpDir, "Dockerfile")
			if err := os.WriteFile(rootDockerfile, []byte("FROM alpine:latest\n"), 0644); err != nil {
				t.Logf("Failed to write root Dockerfile: %v", err)
				return false
			}

			// Criar Dockerfile em subdiretório
			dockerDir := filepath.Join(tmpDir, "docker")
			if err := os.MkdirAll(dockerDir, 0755); err != nil {
				t.Logf("Failed to create docker directory: %v", err)
				return false
			}
			subDockerfile := filepath.Join(dockerDir, "Dockerfile")
			if err := os.WriteFile(subDockerfile, []byte("FROM ubuntu:latest\n"), 0644); err != nil {
				t.Logf("Failed to write subdirectory Dockerfile: %v", err)
				return false
			}

			// Criar serviço
			logger := zap.NewNop()
			svc := NewImageService(logger).(*imageService)

			// Localizar Dockerfile
			foundPath, err := svc.locateDockerfile(tmpDir, "")
			
			// Verificar que encontrou
			if err != nil {
				t.Logf("Failed to locate Dockerfile: %v", err)
				return false
			}

			// Verificar que priorizou o da raiz
			if foundPath != rootDockerfile {
				t.Logf("Expected to find root Dockerfile %s, but found %s", rootDockerfile, foundPath)
				return false
			}

			return true
		},
		gen.UInt8(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}


// pathsEqual compara dois paths considerando case-insensitivity no Windows
func pathsEqual(path1, path2 string) bool {
	// Normalizar ambos os paths
	path1 = filepath.Clean(path1)
	path2 = filepath.Clean(path2)
	
	// Comparar case-insensitive usando strings.EqualFold
	return strings.EqualFold(filepath.ToSlash(path1), filepath.ToSlash(path2))
}
