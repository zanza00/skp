package stack

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "simple", input: "work", want: true},
		{name: "digits", input: "go1", want: true},
		{name: "dots underscores dashes", input: "a.b_c-d", want: true},
		{name: "empty", input: "", want: false},
		{name: "dot", input: ".", want: false},
		{name: "dot dot", input: "..", want: false},
		{name: "slash", input: "a/b", want: false},
		{name: "traversal", input: "../work", want: false},
		{name: "leading dash", input: "-work", want: false},
		{name: "leading underscore", input: "_work", want: false},
		{name: "space", input: "my stack", want: false},
		{name: "newline", input: "work\n", want: false},
		{name: "unicode", input: "caffè", want: false},
		{name: "absolute", input: "/work", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidName(tt.input); got != tt.want {
				t.Errorf("ValidName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	realSkills := filepath.Join(root, "real", ".agents", "skills")
	if err := os.MkdirAll(realSkills, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	fileSkills := filepath.Join(root, "file", ".agents", "skills")
	if err := os.MkdirAll(filepath.Dir(fileSkills), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileSkills, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	symlinkSkills := filepath.Join(root, "symlink", ".agents", "skills")
	if err := os.MkdirAll(filepath.Dir(symlinkSkills), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, symlinkSkills); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		stack   string
		wantErr error
	}{
		{name: "valid", stack: "real"},
		{name: "missing stack directory", stack: "absent", wantErr: ErrMissingSkills},
		{name: "empty stack directory", stack: "broken", wantErr: ErrMissingSkills},
		{name: "skills is a file", stack: "file", wantErr: ErrSkillsNotDir},
		{name: "skills is a symlink", stack: "symlink", wantErr: ErrSkillsNotDir},
		{name: "invalid name", stack: "../real", wantErr: ErrInvalidName},
		{name: "empty name", stack: "", wantErr: ErrInvalidName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Validate(root, tt.stack)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() = %v, want errors.Is(%v)", err, tt.wantErr)
			}
		})
	}
}

func TestDiscover(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, name := range []string{"work", "personal", "_bad"} {
		if err := os.MkdirAll(filepath.Join(root, name, ".agents", "skills"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	fileSkills := filepath.Join(root, "file", ".agents", "skills")
	if err := os.MkdirAll(filepath.Dir(fileSkills), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileSkills, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "not-a-stack"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	want := []Entry{
		{Name: "_bad", Reason: "nome non ammesso"},
		{Name: "broken", Reason: ".agents/skills mancante"},
		{Name: "file", Reason: ".agents/skills non è una directory"},
		{Name: "personal", Selectable: true},
		{Name: "work", Selectable: true},
	}
	if len(entries) != len(want) {
		t.Fatalf("Discover() returned %d entries, want %d: %#v", len(entries), len(want), entries)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entry %d = %#v, want %#v", i, entries[i], want[i])
		}
	}
}

func TestDiscoverMissingDirectory(t *testing.T) {
	t.Parallel()

	entries, err := Discover(filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Discover() = %#v, want no entries", entries)
	}
}

func TestReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "invalid name", err: ErrInvalidName, want: "nome non ammesso"},
		{name: "missing", err: ErrMissingSkills, want: ".agents/skills mancante"},
		{name: "not a directory", err: ErrSkillsNotDir, want: ".agents/skills non è una directory"},
		{name: "unknown", err: errors.New("boom"), want: "non selezionabile"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Reason(tt.err); got != tt.want {
				t.Errorf("Reason(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}
