package live

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTarget(t *testing.T) {
	t.Parallel()

	got := Target("work")
	want := "../skill-stacks/work/.agents/skills"
	if got != want {
		t.Fatalf("Target(work) = %q, want %q", got, want)
	}
}

func TestParseTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		want   string
		ok     bool
	}{
		{name: "managed", target: "../skill-stacks/work/.agents/skills", want: "work", ok: true},
		{name: "allowed characters", target: "../skill-stacks/a.b_c-d/.agents/skills", want: "a.b_c-d", ok: true},
		{name: "empty name", target: "../skill-stacks//.agents/skills"},
		{name: "dot dot name", target: "../skill-stacks/../.agents/skills"},
		{name: "trailing slash", target: "../skill-stacks/work/.agents/skills/"},
		{name: "dot prefix", target: "./../skill-stacks/work/.agents/skills"},
		{name: "absolute", target: "/abs/skill-stacks/work/.agents/skills"},
		{name: "wrong parent", target: "../../work/.agents/skills"},
		{name: "wrong root", target: "../other/work/.agents/skills"},
		{name: "wrong leaf", target: "../skill-stacks/work/.agents/other"},
		{name: "invalid name", target: "../skill-stacks/_bad/.agents/skills"},
		{name: "backslashes", target: `..\skill-stacks\work\.agents\skills`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseTarget(tt.target)
			if ok != tt.ok || got != tt.want {
				t.Errorf("parseTarget(%q) = %q, %v; want %q, %v", tt.target, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestInspect(t *testing.T) {
	t.Parallel()

	t.Run("absent", func(t *testing.T) {
		t.Parallel()
		state, err := Inspect(filepath.Join(t.TempDir(), ".agents", "skills"))
		if err != nil {
			t.Fatal(err)
		}
		if state.Kind != Absent {
			t.Fatalf("Kind = %v, want Absent", state.Kind)
		}
	})

	t.Run("managed symlink", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		link := filepath.Join(root, "skills")
		if err := os.Symlink(Target("work"), link); err != nil {
			t.Fatal(err)
		}
		state, err := Inspect(link)
		if err != nil {
			t.Fatal(err)
		}
		if state.Kind != Managed || state.Stack != "work" || state.RawTarget != Target("work") {
			t.Fatalf("state = %#v", state)
		}
	})

	t.Run("absolute symlink", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		link := filepath.Join(root, "skills")
		target := filepath.Join(root, "stack", ".agents", "skills")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		state, err := Inspect(link)
		if err != nil {
			t.Fatal(err)
		}
		if state.Kind != UnexpectedSymlink || state.RawTarget != target || state.Stack != "" {
			t.Fatalf("state = %#v", state)
		}
	})

	t.Run("unexpected relative symlink", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		link := filepath.Join(root, "skills")
		if err := os.Symlink("../elsewhere", link); err != nil {
			t.Fatal(err)
		}
		state, err := Inspect(link)
		if err != nil {
			t.Fatal(err)
		}
		if state.Kind != UnexpectedSymlink || state.RawTarget != "../elsewhere" {
			t.Fatalf("state = %#v", state)
		}
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		link := filepath.Join(t.TempDir(), "skills")
		if err := os.Mkdir(link, 0o755); err != nil {
			t.Fatal(err)
		}
		state, err := Inspect(link)
		if err != nil {
			t.Fatal(err)
		}
		if state.Kind != Directory {
			t.Fatalf("Kind = %v, want Directory", state.Kind)
		}
	})

	t.Run("file", func(t *testing.T) {
		t.Parallel()
		link := filepath.Join(t.TempDir(), "skills")
		if err := os.WriteFile(link, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		state, err := Inspect(link)
		if err != nil {
			t.Fatal(err)
		}
		if state.Kind != File {
			t.Fatalf("Kind = %v, want File", state.Kind)
		}
	})
}

func TestResolve(t *testing.T) {
	t.Parallel()

	link := "/repo/.agents/skills"
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "relative", target: Target("work"), want: "/repo/skill-stacks/work/.agents/skills"},
		{name: "absolute", target: "/other/skills", want: "/other/skills"},
		{name: "parent traversal", target: "../../etc", want: "/etc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Resolve(link, tt.target); got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestReplaceCreatesMissingLink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	link := filepath.Join(dir, "skills")
	if err := Replace(link, Target("work")); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if target != Target("work") {
		t.Fatalf("readlink = %q, want %q", target, Target("work"))
	}
	assertNoTempLinks(t, dir)
}

func TestReplaceUpdatesExistingLink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	link := filepath.Join(dir, "skills")
	if err := os.Symlink(Target("old"), link); err != nil {
		t.Fatal(err)
	}
	if err := Replace(link, Target("new")); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if target != Target("new") {
		t.Fatalf("readlink = %q, want %q", target, Target("new"))
	}
	assertNoTempLinks(t, dir)
}

func TestReplaceKeepsDirectoryAndCleansUp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	link := filepath.Join(dir, "skills")
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(link, "keep")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Replace(link, Target("work")); err == nil {
		t.Fatal("Replace() succeeded over a real directory")
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatal("live path is no longer a directory")
	}
	if content, err := os.ReadFile(sentinel); err != nil || string(content) != "keep" {
		t.Fatalf("sentinel = %q, %v; want %q", content, err, "keep")
	}
	assertNoTempLinks(t, dir)
}

func assertNoTempLinks(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".skp-tmp-") {
			t.Errorf("temporary link left behind: %s", entry.Name())
		}
	}
}
