package detection

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		wantLang  Language
		wantTool  BuildTool
		wantError bool
	}{
		{
			name:     "go project",
			files:    []string{"go.mod"},
			wantLang: LanguageGo, wantTool: BuildToolGo,
		},
		{
			name:     "java maven",
			files:    []string{"pom.xml"},
			wantLang: LanguageJava, wantTool: BuildToolMaven,
		},
		{
			name:     "java gradle",
			files:    []string{"build.gradle"},
			wantLang: LanguageJava, wantTool: BuildToolGradle,
		},
		{
			name:     "java gradle kts",
			files:    []string{"build.gradle.kts"},
			wantLang: LanguageJava, wantTool: BuildToolGradle,
		},
		{
			name:     "dotnet",
			files:    []string{"MyApp.csproj"},
			wantLang: LanguageDotNet, wantTool: BuildToolDotNet,
		},
		{
			name:      "unknown",
			files:     []string{"README.md"},
			wantError: true,
		},
		{
			// Go wins over Java when both present
			name:     "go wins over java",
			files:    []string{"go.mod", "pom.xml"},
			wantLang: LanguageGo, wantTool: BuildToolGo,
		},
		{
			// Java wins over .NET when both present
			name:     "java wins over dotnet",
			files:    []string{"pom.xml", "App.csproj"},
			wantLang: LanguageJava, wantTool: BuildToolMaven,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte(""), 0600); err != nil {
					t.Fatal(err)
				}
			}

			result, err := Detect(dir)
			if tc.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Language != tc.wantLang {
				t.Errorf("language: got %q, want %q", result.Language, tc.wantLang)
			}
			if result.BuildTool != tc.wantTool {
				t.Errorf("build tool: got %q, want %q", result.BuildTool, tc.wantTool)
			}
		})
	}
}
