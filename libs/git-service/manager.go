package gitservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/oci-build-system/libs/shared"
	"go.uber.org/zap"
)

// GitService define a interface para operações Git
type GitService interface {
	SyncRepository(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error)
	RepositoryExists(repoURL string) bool
	GetLocalPath(repoURL string) string
}

// gitServiceImpl implementa GitService
type gitServiceImpl struct {
	codeCachePath string
	logger        *zap.Logger
	maxRetries    int
	retryDelay    time.Duration
}

// Config contém configurações para o GitService
type Config struct {
	CodeCachePath string
	MaxRetries    int
	RetryDelay    time.Duration
}

// NewGitService cria uma nova instância de GitService
func NewGitService(config Config, logger *zap.Logger) GitService {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}

	return &gitServiceImpl{
		codeCachePath: config.CodeCachePath,
		logger:        logger,
		maxRetries:    config.MaxRetries,
		retryDelay:    config.RetryDelay,
	}
}

// SyncRepository sincroniza um repositório (clone ou pull) e faz checkout do commit especificado
func (g *gitServiceImpl) SyncRepository(ctx context.Context, repo shared.RepositoryInfo, commitHash string) (string, error) {
	if !repo.IsValid() {
		return "", fmt.Errorf("invalid repository info")
	}

	if commitHash == "" {
		return "", fmt.Errorf("commit hash is required")
	}

	localPath := g.GetLocalPath(repo.URL)
	g.logger.Info("syncing repository",
		zap.String("repo", repo.FullName()),
		zap.String("url", repo.URL),
		zap.String("commit", commitHash),
		zap.String("local_path", localPath),
	)

	var err error
	var gitRepo *git.Repository

	if g.RepositoryExists(repo.URL) {
		g.logger.Debug("repository exists in cache, performing pull",
			zap.String("repo", repo.FullName()),
		)
		gitRepo, err = g.pullWithRetry(ctx, localPath)
	} else {
		g.logger.Debug("repository not in cache, performing clone",
			zap.String("repo", repo.FullName()),
		)
		gitRepo, err = g.cloneWithRetry(ctx, repo.URL, localPath)
	}

	if err != nil {
		return "", fmt.Errorf("failed to sync repository: %w", err)
	}

	// Fazer checkout do commit específico
	if err := g.checkoutCommit(gitRepo, commitHash); err != nil {
		return "", fmt.Errorf("failed to checkout commit %s: %w", commitHash, err)
	}

	g.logger.Info("repository synced successfully",
		zap.String("repo", repo.FullName()),
		zap.String("commit", commitHash),
		zap.String("local_path", localPath),
	)

	return localPath, nil
}

// RepositoryExists verifica se um repositório existe no cache local
func (g *gitServiceImpl) RepositoryExists(repoURL string) bool {
	localPath := g.GetLocalPath(repoURL)
	
	// Verificar se o diretório existe
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return false
	}

	// Verificar se é um repositório Git válido
	_, err := git.PlainOpen(localPath)
	return err == nil
}

// GetLocalPath calcula o path local para um repositório
func (g *gitServiceImpl) GetLocalPath(repoURL string) string {
	// Extrair owner e name da URL
	// Exemplo: https://github.com/owner/repo.git -> owner/repo
	parts := strings.Split(strings.TrimSuffix(repoURL, ".git"), "/")
	
	if len(parts) < 2 {
		// Fallback: usar hash da URL
		return filepath.Join(g.codeCachePath, fmt.Sprintf("repo-%x", repoURL))
	}

	owner := parts[len(parts)-2]
	name := parts[len(parts)-1]

	return filepath.Join(g.codeCachePath, owner, name)
}

// cloneWithRetry realiza clone com retry e backoff exponencial
func (g *gitServiceImpl) cloneWithRetry(ctx context.Context, repoURL, localPath string) (*git.Repository, error) {
	var lastErr error
	delay := g.retryDelay

	for attempt := 1; attempt <= g.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		g.logger.Debug("attempting to clone repository",
			zap.String("url", repoURL),
			zap.Int("attempt", attempt),
			zap.Int("max_retries", g.maxRetries),
		)

		// Criar diretório pai se não existir
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}

		repo, err := git.PlainCloneContext(ctx, localPath, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: nil, // Podemos adicionar progress tracking depois
		})

		if err == nil {
			g.logger.Info("repository cloned successfully",
				zap.String("url", repoURL),
				zap.String("local_path", localPath),
				zap.Int("attempt", attempt),
			)
			return repo, nil
		}

		lastErr = err
		g.logger.Warn("clone attempt failed",
			zap.String("url", repoURL),
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		// Se não for o último retry, aguardar com backoff exponencial
		if attempt < g.maxRetries {
			g.logger.Debug("waiting before retry",
				zap.Duration("delay", delay),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay *= 2 // Backoff exponencial
			}
		}
	}

	return nil, fmt.Errorf("failed to clone after %d attempts: %w", g.maxRetries, lastErr)
}

// pullWithRetry realiza pull com retry e backoff exponencial
func (g *gitServiceImpl) pullWithRetry(ctx context.Context, localPath string) (*git.Repository, error) {
	repo, err := git.PlainOpen(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	var lastErr error
	delay := g.retryDelay

	for attempt := 1; attempt <= g.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		g.logger.Debug("attempting to pull repository",
			zap.String("local_path", localPath),
			zap.Int("attempt", attempt),
			zap.Int("max_retries", g.maxRetries),
		)

		err := worktree.PullContext(ctx, &git.PullOptions{
			RemoteName: "origin",
		})

		// git.NoErrAlreadyUpToDate não é um erro real
		if err == nil || err == git.NoErrAlreadyUpToDate {
			g.logger.Info("repository pulled successfully",
				zap.String("local_path", localPath),
				zap.Int("attempt", attempt),
				zap.Bool("already_up_to_date", err == git.NoErrAlreadyUpToDate),
			)
			return repo, nil
		}

		lastErr = err
		g.logger.Warn("pull attempt failed",
			zap.String("local_path", localPath),
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		// Se não for o último retry, aguardar com backoff exponencial
		if attempt < g.maxRetries {
			g.logger.Debug("waiting before retry",
				zap.Duration("delay", delay),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay *= 2 // Backoff exponencial
			}
		}
	}

	// Se pull falhou após todos os retries, usar cache existente com aviso
	g.logger.Warn("failed to pull after all retries, using cached version",
		zap.String("local_path", localPath),
		zap.Error(lastErr),
	)

	return repo, nil
}

// checkoutCommit faz checkout de um commit específico
func (g *gitServiceImpl) checkoutCommit(repo *git.Repository, commitHash string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	g.logger.Debug("checking out commit",
		zap.String("commit", commitHash),
	)

	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(commitHash),
	})

	if err != nil {
		return fmt.Errorf("failed to checkout: %w", err)
	}

	g.logger.Debug("commit checked out successfully",
		zap.String("commit", commitHash),
	)

	return nil
}
