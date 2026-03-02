package templates

import (
	"strings"
	"testing"

	"github.com/jorgerua/build-system/container-build-service/internal/detection"
)

func TestRender(t *testing.T) {
	vars := TemplateVars{
		ProjectName:    "api",
		ProjectSubpath: "apps/api",
		ArtifactName:   "api",
	}

	tests := []struct {
		tool      detection.BuildTool
		mustContain []string
	}{
		{
			tool: detection.BuildToolGo,
			mustContain: []string{
				"FROM golang:",
				"COPY . .",
				"go build",
				"./apps/api/...",
				"distroless",
			},
		},
		{
			tool: detection.BuildToolMaven,
			mustContain: []string{
				"FROM maven:",
				"COPY apps/api/ .",
				"mvn package -DskipTests",
				"eclipse-temurin",
			},
		},
		{
			tool: detection.BuildToolGradle,
			mustContain: []string{
				"FROM gradle:",
				"COPY apps/api/ .",
				"gradle build -x test",
				"eclipse-temurin",
			},
		},
		{
			tool: detection.BuildToolDotNet,
			mustContain: []string{
				"FROM mcr.microsoft.com/dotnet/sdk",
				"COPY apps/api/ .",
				"dotnet restore",
				"dotnet publish",
				"aspnet",
			},
		},
	}

	for _, tc := range tests {
		t.Run(string(tc.tool), func(t *testing.T) {
			out, err := Render(tc.tool, vars)
			if err != nil {
				t.Fatalf("Render(%q): %v", tc.tool, err)
			}
			for _, s := range tc.mustContain {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nfull output:\n%s", s, out)
				}
			}
		})
	}

	_, err := Render("unknown", vars)
	if err == nil {
		t.Error("expected error for unknown build tool")
	}
}
