package buildah

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/jorgerua/build-system/container-build-service/internal/config"
	"go.uber.org/zap"
)

// Builder executes buildah bud and buildah push as subprocesses.
type Builder struct {
	cfg    *config.Config
	driver string // "overlay" or "vfs"
	logger *zap.Logger
}

// New creates a Builder and detects the available storage driver.
func New(cfg *config.Config, logger *zap.Logger) *Builder {
	driver := detectStorageDriver(logger)
	cfg.Buildah.StorageDriver = driver
	return &Builder{cfg: cfg, driver: driver, logger: logger}
}

// detectStorageDriver probes for overlay capability at startup.
// Falls back to vfs if overlay is unavailable.
func detectStorageDriver(logger *zap.Logger) string {
	cmd := exec.Command("buildah", "info", "--storage-driver", "overlay")
	if err := cmd.Run(); err == nil {
		logger.Info("buildah: using overlay storage driver")
		return "overlay"
	}
	logger.Warn("buildah: overlay unavailable, falling back to vfs")
	return "vfs"
}

// Build writes the generated Dockerfile to a temp file, runs buildah bud,
// then removes the temp file regardless of outcome.
func (b *Builder) Build(ctx context.Context, jobID, project, imageRef, repoDir, dockerfileContent string) error {
	// Write Dockerfile to temp file.
	dfPath := fmt.Sprintf("/tmp/dockerfile-%s-%s", jobID, project)
	if err := os.WriteFile(dfPath, []byte(dockerfileContent), 0600); err != nil {
		return fmt.Errorf("write dockerfile: %w", err)
	}
	defer os.Remove(dfPath)

	args := []string{
		"bud",
		"--storage-driver", b.driver,
		"--root", b.cfg.Buildah.StorageRoot,
		"-f", dfPath,
		"-t", imageRef,
		repoDir,
	}

	stdout, stderr, err := b.run(ctx, args)
	b.logger.Info("buildah bud",
		zap.String("project", project),
		zap.String("image", imageRef),
		zap.String("stdout", stdout),
	)
	if err != nil {
		b.logger.Error("buildah bud failed",
			zap.String("project", project),
			zap.String("stderr", stderr),
			zap.Error(err),
		)
		return fmt.Errorf("buildah bud: %w", err)
	}
	return nil
}

// Push runs buildah push to send the built image to the registry.
func (b *Builder) Push(ctx context.Context, project, imageRef string) error {
	args := []string{
		"push",
		"--storage-driver", b.driver,
		"--root", b.cfg.Buildah.StorageRoot,
		imageRef,
		"--authfile", b.cfg.Registry.AuthFile,
	}

	stdout, stderr, err := b.run(ctx, args)
	b.logger.Info("buildah push",
		zap.String("project", project),
		zap.String("image", imageRef),
		zap.String("stdout", stdout),
	)
	if err != nil {
		b.logger.Error("buildah push failed",
			zap.String("project", project),
			zap.String("stderr", stderr),
			zap.Error(err),
		)
		return fmt.Errorf("buildah push: %w", err)
	}
	return nil
}

func (b *Builder) run(ctx context.Context, args []string) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "buildah", args...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// ImageRef builds the full image reference: registry/project:version.
func ImageRef(registry, project, version string) string {
	return fmt.Sprintf("%s/%s:%s", registry, project, version)
}
