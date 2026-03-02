package detection

import (
	"fmt"
	"os"
	"path/filepath"
)

// Language identifies the programming language of a project.
type Language string

const (
	LanguageGo   Language = "go"
	LanguageJava Language = "java"
	LanguageDotNet Language = "dotnet"
)

// BuildTool identifies the build tool used by a project.
type BuildTool string

const (
	BuildToolGo     BuildTool = "go"
	BuildToolMaven  BuildTool = "maven"
	BuildToolGradle BuildTool = "gradle"
	BuildToolDotNet BuildTool = "dotnet"
)

// Result holds the detected language and build tool for a project.
type Result struct {
	Language  Language
	BuildTool BuildTool
}

// ErrUnknownLanguage is returned when no supported language marker is found.
type ErrUnknownLanguage struct {
	ProjectPath string
}

func (e *ErrUnknownLanguage) Error() string {
	return fmt.Sprintf("unknown language for project at %q: no supported marker file found", e.ProjectPath)
}

// Detect scans projectDir for language marker files and returns the result.
// Priority order: Go > Java > .NET.
func Detect(projectDir string) (Result, error) {
	// Go: go.mod
	if exists(projectDir, "go.mod") {
		return Result{Language: LanguageGo, BuildTool: BuildToolGo}, nil
	}

	// Java: pom.xml (Maven) or build.gradle / build.gradle.kts (Gradle)
	if exists(projectDir, "pom.xml") {
		return Result{Language: LanguageJava, BuildTool: BuildToolMaven}, nil
	}
	if exists(projectDir, "build.gradle") || exists(projectDir, "build.gradle.kts") {
		return Result{Language: LanguageJava, BuildTool: BuildToolGradle}, nil
	}

	// .NET: any *.csproj file
	matches, err := filepath.Glob(filepath.Join(projectDir, "*.csproj"))
	if err != nil {
		return Result{}, fmt.Errorf("glob csproj: %w", err)
	}
	if len(matches) > 0 {
		return Result{Language: LanguageDotNet, BuildTool: BuildToolDotNet}, nil
	}

	return Result{}, &ErrUnknownLanguage{ProjectPath: projectDir}
}

func exists(dir, filename string) bool {
	_, err := os.Stat(filepath.Join(dir, filename))
	return err == nil
}
