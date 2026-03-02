package orchestrator

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	githubpkg "github.com/jorgerua/build-system/container-build-service/internal/github"
)

// cloneRepo generates a fresh installation token and clones the repository
// to /tmp/repo-<jobID>, checking out the given SHA.
// Returns the local repo path.
func cloneRepo(ctx context.Context, gh *githubpkg.Client, repoURL string, installationID int64, sha, jobID string) (string, error) {
	token, err := gh.GenerateInstallationToken(ctx, installationID)
	if err != nil {
		return "", fmt.Errorf("generate installation token: %w", err)
	}

	// Inject token into clone URL: https://x-access-token:<token>@github.com/...
	authedURL := injectToken(repoURL, token)

	repoDir := fmt.Sprintf("/tmp/repo-%s", jobID)

	if out, err := runGit(ctx, "clone", "--no-tags", authedURL, repoDir); err != nil {
		return "", fmt.Errorf("git clone: %w\n%s", err, out)
	}

	if out, err := runGitDir(ctx, repoDir, "checkout", sha); err != nil {
		return "", fmt.Errorf("git checkout %s: %w\n%s", sha, err, out)
	}

	return repoDir, nil
}

// initialCommitSHA returns the first commit SHA of the repository using the local clone.
func initialCommitSHA(ctx context.Context, repoDir string) (string, error) {
	out, err := runGitDir(ctx, repoDir, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-list initial: %w", err)
	}
	sha := strings.TrimSpace(out)
	if sha == "" {
		return "", fmt.Errorf("no initial commit found")
	}
	return sha, nil
}

func injectToken(repoURL, token string) string {
	// Convert https://github.com/... → https://x-access-token:<token>@github.com/...
	const httpsPrefix = "https://"
	if strings.HasPrefix(repoURL, httpsPrefix) {
		return httpsPrefix + "x-access-token:" + token + "@" + repoURL[len(httpsPrefix):]
	}
	return repoURL
}

func runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runGitDir(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
