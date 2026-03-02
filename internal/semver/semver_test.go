package semver

import (
	"testing"
)

func TestParseCommit(t *testing.T) {
	tests := []struct {
		msg      string
		expected BumpType
	}{
		{"feat: add login", BumpMinor},
		{"feat(auth): add OAuth", BumpMinor},
		{"fix: correct typo", BumpPatch},
		{"fix(api): handle nil pointer", BumpPatch},
		{"chore: update deps", BumpPatch},
		{"docs: update README", BumpPatch},
		{"refactor: simplify handler", BumpPatch},
		{"feat!: redesign API", BumpMajor},
		{"fix!: breaking change", BumpMajor},
		{"feat(scope)!: breaking", BumpMajor},
		{"feat: add thing\n\nBREAKING CHANGE: old API removed", BumpMajor},
		{"no prefix here", BumpPatch},
		{"", BumpPatch},
	}

	for _, tc := range tests {
		t.Run(tc.msg, func(t *testing.T) {
			got := ParseCommit(tc.msg)
			if got != tc.expected {
				t.Errorf("ParseCommit(%q) = %v, want %v", tc.msg, got, tc.expected)
			}
		})
	}
}

func TestHighestBump(t *testing.T) {
	msgs := []string{"fix: typo", "feat: new thing", "chore: cleanup"}
	if got := HighestBump(msgs); got != BumpMinor {
		t.Errorf("expected BumpMinor, got %v", got)
	}

	breaking := []string{"feat!: breaking", "fix: small"}
	if got := HighestBump(breaking); got != BumpMajor {
		t.Errorf("expected BumpMajor, got %v", got)
	}

	patches := []string{"fix: a", "chore: b"}
	if got := HighestBump(patches); got != BumpPatch {
		t.Errorf("expected BumpPatch, got %v", got)
	}
}

func TestIncrement(t *testing.T) {
	tests := []struct {
		current string
		bump    BumpType
		want    string
	}{
		{"1.2.3", BumpPatch, "1.2.4"},
		{"1.2.3", BumpMinor, "1.3.0"},
		{"1.2.3", BumpMajor, "2.0.0"},
		{"0.1.0", BumpPatch, "0.1.1"},
		{"0.1.0", BumpMinor, "0.2.0"},
		{"0.1.0", BumpMajor, "1.0.0"},
		{"2.0.0", BumpMajor, "3.0.0"},
	}

	for _, tc := range tests {
		got, err := Increment(tc.current, tc.bump)
		if err != nil {
			t.Fatalf("Increment(%q, %v): %v", tc.current, tc.bump, err)
		}
		if got != tc.want {
			t.Errorf("Increment(%q, %v) = %q, want %q", tc.current, tc.bump, got, tc.want)
		}
	}

	_, err := Increment("invalid", BumpPatch)
	if err == nil {
		t.Error("expected error for invalid semver")
	}
}
