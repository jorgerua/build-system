package gitservice

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/oci-build-system/libs/shared"
	"go.uber.org/zap"
)

// Feature: oci-build-system, Property 4: Sincronização de repositório
// Para qualquer repositório, se ele não existe localmente, então git clone deve ser executado;
// se existe, então git pull deve ser executado; e em ambos os casos o código local deve
// refletir o commit especificado.
// **Valida: Requisitos 2.1, 2.2, 2.3**
func TestProperty_RepositorySynchronization(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("repository sync creates or updates local copy with correct commit", prop.ForAll(
		func(repoSuffix uint8, ownerSuffix uint8, commitCount uint8, syncTwice bool) bool {
			// Gerar nomes válidos a partir dos sufixos
			repoName := "test-repo-" + string(rune('a'+repoSuffix%26))
			ownerName := "test-owner-" + string(rune('a'+ownerSuffix%26))
			
			// Limitar commitCount
			if commitCount == 0 {
				commitCount = 1
			}
			if commitCount > 10 {
				commitCount = 10
			}

			// Criar diretórios temporários para repositório remoto e cache local
			remoteDir, err := os.MkdirTemp("", "remote-repo-*")
			if err != nil {
				t.Logf("Failed to create remote dir: %v", err)
				return false
			}
			defer os.RemoveAll(remoteDir)

			cacheDir, err := os.MkdirTemp("", "cache-dir-*")
			if err != nil {
				t.Logf("Failed to create cache dir: %v", err)
				return false
			}
			defer os.RemoveAll(cacheDir)

			// 1. Criar repositório remoto simulado
			remoteRepo, err := git.PlainInit(remoteDir, false)
			if err != nil {
				t.Logf("Failed to init remote repo: %v", err)
				return false
			}

			// Criar commits no repositório remoto
			worktree, err := remoteRepo.Worktree()
			if err != nil {
				t.Logf("Failed to get worktree: %v", err)
				return false
			}

			commitHashes := make([]string, 0, commitCount)
			for i := uint8(0); i < commitCount; i++ {
				// Criar arquivo único para cada commit
				fileName := filepath.Join(remoteDir, "file"+string(rune('a'+i))+".txt")
				content := []byte("content " + string(rune('a'+i)))
				if err := os.WriteFile(fileName, content, 0644); err != nil {
					t.Logf("Failed to write file: %v", err)
					return false
				}

				// Adicionar ao staging
				if _, err := worktree.Add(filepath.Base(fileName)); err != nil {
					t.Logf("Failed to add file: %v", err)
					return false
				}

				// Criar commit
				hash, err := worktree.Commit("Commit "+string(rune('a'+i)), &git.CommitOptions{
					Author: &object.Signature{
						Name:  "Test Author",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				if err != nil {
					t.Logf("Failed to commit: %v", err)
					return false
				}

				commitHashes = append(commitHashes, hash.String())
			}

			// Selecionar um commit para sincronizar (o último)
			targetCommitHash := commitHashes[len(commitHashes)-1]

			// 2. Criar GitService
			logger := zap.NewNop()
			config := Config{
				CodeCachePath: cacheDir,
				MaxRetries:    1,
				RetryDelay:    time.Millisecond * 10,
			}
			svc := NewGitService(config, logger)

			// 3. Criar RepositoryInfo
			repoInfo := shared.RepositoryInfo{
				URL:    remoteDir, // Usar diretório local como "remoto"
				Name:   repoName,
				Owner:  ownerName,
				Branch: "main",
			}

			ctx := context.Background()

			// 4. Primeira sincronização (deve fazer clone)
			// Verificar que repositório não existe antes
			if svc.RepositoryExists(repoInfo.URL) {
				t.Logf("Repository should not exist before first sync")
				return false
			}

			localPath1, err := svc.SyncRepository(ctx, repoInfo, targetCommitHash)
			if err != nil {
				t.Logf("First sync failed: %v", err)
				return false
			}

			// Verificar que repositório agora existe
			if !svc.RepositoryExists(repoInfo.URL) {
				t.Logf("Repository should exist after first sync")
				return false
			}

			// Verificar que o path local está correto
			expectedPath := svc.GetLocalPath(repoInfo.URL)
			if localPath1 != expectedPath {
				t.Logf("Local path mismatch: got %s, want %s", localPath1, expectedPath)
				return false
			}

			// Verificar que o commit correto foi checked out
			localRepo1, err := git.PlainOpen(localPath1)
			if err != nil {
				t.Logf("Failed to open local repo: %v", err)
				return false
			}

			head1, err := localRepo1.Head()
			if err != nil {
				t.Logf("Failed to get HEAD: %v", err)
				return false
			}

			if head1.Hash().String() != targetCommitHash {
				t.Logf("Wrong commit checked out: got %s, want %s", head1.Hash().String(), targetCommitHash)
				return false
			}

			// Verificar que os arquivos existem no diretório local
			for i := uint8(0); i < commitCount; i++ {
				fileName := filepath.Join(localPath1, "file"+string(rune('a'+i))+".txt")
				if _, err := os.Stat(fileName); os.IsNotExist(err) {
					t.Logf("File %s does not exist in local repo", fileName)
					return false
				}
			}

			// 5. Se syncTwice, fazer segunda sincronização (deve fazer pull)
			if syncTwice {
				// Adicionar novo commit ao repositório remoto
				newFileName := filepath.Join(remoteDir, "new-file.txt")
				if err := os.WriteFile(newFileName, []byte("new content"), 0644); err != nil {
					t.Logf("Failed to write new file: %v", err)
					return false
				}

				if _, err := worktree.Add("new-file.txt"); err != nil {
					t.Logf("Failed to add new file: %v", err)
					return false
				}

				newHash, err := worktree.Commit("New commit", &git.CommitOptions{
					Author: &object.Signature{
						Name:  "Test Author",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				if err != nil {
					t.Logf("Failed to create new commit: %v", err)
					return false
				}

				newCommitHash := newHash.String()

				// Segunda sincronização
				localPath2, err := svc.SyncRepository(ctx, repoInfo, newCommitHash)
				if err != nil {
					t.Logf("Second sync failed: %v", err)
					return false
				}

				// Verificar que o path é o mesmo
				if localPath2 != localPath1 {
					t.Logf("Local path changed: got %s, want %s", localPath2, localPath1)
					return false
				}

				// Verificar que o novo commit foi checked out
				localRepo2, err := git.PlainOpen(localPath2)
				if err != nil {
					t.Logf("Failed to open local repo after second sync: %v", err)
					return false
				}

				head2, err := localRepo2.Head()
				if err != nil {
					t.Logf("Failed to get HEAD after second sync: %v", err)
					return false
				}

				if head2.Hash().String() != newCommitHash {
					t.Logf("Wrong commit after second sync: got %s, want %s", head2.Hash().String(), newCommitHash)
					return false
				}

				// Verificar que o novo arquivo existe
				newFileLocal := filepath.Join(localPath2, "new-file.txt")
				if _, err := os.Stat(newFileLocal); os.IsNotExist(err) {
					t.Logf("New file does not exist after second sync")
					return false
				}
			}

			return true
		},
		gen.UInt8(),
		gen.UInt8(),
		gen.UInt8Range(1, 10),
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
