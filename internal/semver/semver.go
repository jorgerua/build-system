package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// BumpType represents the type of SemVer bump.
type BumpType int

const (
	BumpPatch BumpType = iota
	BumpMinor
	BumpMajor
)

// conventionalPattern matches "type[(scope)][!]: description"
var conventionalPattern = regexp.MustCompile(`^(\w+)(\([^)]*\))?(!)?:`)

// ParseCommit extracts the bump type from a single commit message.
func ParseCommit(message string) BumpType {
	// Check for BREAKING CHANGE footer (anywhere in the message body).
	if strings.Contains(message, "BREAKING CHANGE:") {
		return BumpMajor
	}

	m := conventionalPattern.FindStringSubmatch(message)
	if m == nil {
		// No conventional prefix → default patch.
		return BumpPatch
	}

	commitType := m[1]
	bang := m[3] // "!" if present

	if bang == "!" {
		return BumpMajor
	}

	switch commitType {
	case "feat":
		return BumpMinor
	default:
		// fix, chore, ci, docs, style, refactor, perf, test, build, etc. → patch
		return BumpPatch
	}
}

// HighestBump returns the highest bump type across multiple commit messages.
func HighestBump(messages []string) BumpType {
	highest := BumpPatch
	for _, msg := range messages {
		b := ParseCommit(msg)
		if b > highest {
			highest = b
		}
	}
	return highest
}

// Increment applies a bump to the given version string (e.g. "1.2.3")
// and resets lower components per SemVer rules.
func Increment(current string, bump BumpType) (string, error) {
	parts := strings.SplitN(current, ".", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid semver %q", current)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", fmt.Errorf("parse major: %w", err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("parse minor: %w", err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("parse patch: %w", err)
	}

	switch bump {
	case BumpMajor:
		major++
		minor = 0
		patch = 0
	case BumpMinor:
		minor++
		patch = 0
	case BumpPatch:
		patch++
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}
