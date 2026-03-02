package templates

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/jorgerua/build-system/container-build-service/internal/detection"
)

//go:embed *.tmpl
var templateFS embed.FS

// TemplateVars holds the variables injected into Dockerfile templates.
type TemplateVars struct {
	ProjectName    string // e.g. "api"
	ProjectSubpath string // e.g. "apps/api"
	ArtifactName   string // e.g. "api" or "api-1.0.0.jar"
}

var templateNames = map[detection.BuildTool]string{
	detection.BuildToolGo:     "go.dockerfile.tmpl",
	detection.BuildToolMaven:  "java-maven.dockerfile.tmpl",
	detection.BuildToolGradle: "java-gradle.dockerfile.tmpl",
	detection.BuildToolDotNet: "dotnet.dockerfile.tmpl",
}

// Render generates a Dockerfile string for the given build tool and variables.
func Render(buildTool detection.BuildTool, vars TemplateVars) (string, error) {
	tmplName, ok := templateNames[buildTool]
	if !ok {
		return "", fmt.Errorf("no template for build tool %q", buildTool)
	}

	tmplContent, err := templateFS.ReadFile(tmplName)
	if err != nil {
		return "", fmt.Errorf("read template %q: %w", tmplName, err)
	}

	tmpl, err := template.New(tmplName).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", tmplName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("render template %q: %w", tmplName, err)
	}
	return buf.String(), nil
}
