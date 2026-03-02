package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// affectedProjects runs `nx affected` and returns projects under apps/.
func affectedProjects(ctx context.Context, repoDir, baseSHA, headSHA string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "nx", "affected",
		"--base="+baseSHA,
		"--head="+headSHA,
		"--plain",
	)
	cmd.Dir = repoDir

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nx affected: %w", err)
	}

	var projects []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		// Filter to projects under apps/ by checking the nx project root convention.
		// nx --plain returns project names; we check if apps/<name> exists.
		projectPath := filepath.Join(repoDir, "apps", name)
		if dirExists(projectPath) {
			projects = append(projects, name)
		}
	}
	return projects, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
