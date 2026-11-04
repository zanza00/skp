package cli

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := Run(t.TempDir(), []string{"--version"}, &stdout, &stderr)

	if code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", code, ExitOK, stderr.String())
	}
	if stderr.String() != "" {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}

	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("stdout = %q, want three lines", stdout.String())
	}
	if lines[0] != "skills-stack" {
		t.Errorf("line 0 = %q, want %q", lines[0], "skills-stack")
	}
	if !strings.HasPrefix(lines[1], "commit: ") || strings.TrimPrefix(lines[1], "commit: ") == "" {
		t.Errorf("line 1 = %q, want a non-empty commit", lines[1])
	}
	if !strings.HasPrefix(lines[2], "data commit: ") || strings.TrimPrefix(lines[2], "data commit: ") == "" {
		t.Errorf("line 2 = %q, want a non-empty commit date", lines[2])
	}
}

func TestVersionFromSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		settings   []debug.BuildSetting
		wantCommit string
		wantDate   string
	}{
		{
			name: "vcs metadata",
			settings: []debug.BuildSetting{
				{Key: "GOOS", Value: "linux"},
				{Key: "vcs.revision", Value: "abc123"},
				{Key: "vcs.time", Value: "2026-11-04T13:06:48Z"},
			},
			wantCommit: "abc123",
			wantDate:   "2026-11-04T13:06:48Z",
		},
		{
			name:     "no metadata",
			settings: nil,
		},
		{
			name: "commit without date",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abc123"},
			},
			wantCommit: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			commit, date := versionFromSettings(tt.settings)
			if commit != tt.wantCommit {
				t.Errorf("commit = %q, want %q", commit, tt.wantCommit)
			}
			if date != tt.wantDate {
				t.Errorf("date = %q, want %q", date, tt.wantDate)
			}
		})
	}
}
