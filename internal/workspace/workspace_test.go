package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	_ = os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	os.Exit(m.Run())
}

func TestOpenFromSubdirectory(t *testing.T) {
	t.Parallel()

	root := newGitRepo(t)
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := Open(deep)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if ws.Root != root {
		t.Fatalf("Root = %q, want %q", ws.Root, root)
	}
}

func TestOpenOutsideRepository(t *testing.T) {
	t.Parallel()

	if _, err := Open(t.TempDir()); err == nil {
		t.Fatal("Open() succeeded outside a Git work tree")
	}
}

func TestIgnoresLiveLink(t *testing.T) {
	t.Parallel()

	t.Run("ignored", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)
		writeFile(t, filepath.Join(root, ".gitignore"), "/.agents/skills\n")

		ws := mustOpen(t, root)
		ignored, err := ws.IgnoresLiveLink()
		if err != nil {
			t.Fatal(err)
		}
		if !ignored {
			t.Fatal("IgnoresLiveLink() = false, want true")
		}
	})

	t.Run("not ignored", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)

		ws := mustOpen(t, root)
		ignored, err := ws.IgnoresLiveLink()
		if err != nil {
			t.Fatal(err)
		}
		if ignored {
			t.Fatal("IgnoresLiveLink() = true, want false")
		}
	})

	t.Run("other pattern does not match", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)
		writeFile(t, filepath.Join(root, ".gitignore"), "/.skills\n")

		ws := mustOpen(t, root)
		ignored, err := ws.IgnoresLiveLink()
		if err != nil {
			t.Fatal(err)
		}
		if ignored {
			t.Fatal("IgnoresLiveLink() = true, want false")
		}
	})

	t.Run("tracked live link is not ignored", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)
		writeFile(t, filepath.Join(root, ".gitignore"), "/.agents/skills\n")
		if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../skill-stacks/work/.agents/skills", filepath.Join(root, ".agents", "skills")); err != nil {
			t.Fatal(err)
		}
		runGit(t, root, "add", "-f", ".agents/skills")

		ws := mustOpen(t, root)
		ignored, err := ws.IgnoresLiveLink()
		if err != nil {
			t.Fatal(err)
		}
		if ignored {
			t.Fatal("IgnoresLiveLink() = true for a tracked path, want false")
		}
	})
}

func TestIgnoresLiveLinkIncludingTracked(t *testing.T) {
	t.Parallel()

	t.Run("tracked path with ignore rule", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)
		writeFile(t, filepath.Join(root, ".gitignore"), "/.agents/skills\n")
		writeFile(t, filepath.Join(root, ".agents", "skills", "SKILL.md"), "# work\n")
		runGit(t, root, "add", "-f", ".agents/skills")

		ws := mustOpen(t, root)
		ignored, err := ws.IgnoresLiveLink()
		if err != nil {
			t.Fatal(err)
		}
		if ignored {
			t.Fatal("IgnoresLiveLink() = true for a tracked path, want false")
		}

		ignored, err = ws.IgnoresLiveLinkIncludingTracked()
		if err != nil {
			t.Fatal(err)
		}
		if !ignored {
			t.Fatal("IgnoresLiveLinkIncludingTracked() = false, want true")
		}
	})

	t.Run("untracked path with ignore rule", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)
		writeFile(t, filepath.Join(root, ".gitignore"), "/.agents/skills\n")

		ws := mustOpen(t, root)
		ignored, err := ws.IgnoresLiveLinkIncludingTracked()
		if err != nil {
			t.Fatal(err)
		}
		if !ignored {
			t.Fatal("IgnoresLiveLinkIncludingTracked() = false, want true")
		}
	})

	t.Run("no ignore rule", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)

		ws := mustOpen(t, root)
		ignored, err := ws.IgnoresLiveLinkIncludingTracked()
		if err != nil {
			t.Fatal(err)
		}
		if ignored {
			t.Fatal("IgnoresLiveLinkIncludingTracked() = true, want false")
		}
	})
}

func TestAgentsDirState(t *testing.T) {
	t.Parallel()

	t.Run("absent", func(t *testing.T) {
		t.Parallel()
		ws := mustOpen(t, newGitRepo(t))
		state, err := ws.AgentsDirState()
		if err != nil {
			t.Fatal(err)
		}
		if state != DirAbsent {
			t.Fatalf("state = %v, want DirAbsent", state)
		}
	})

	t.Run("real directory", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)
		if err := os.Mkdir(filepath.Join(root, ".agents"), 0o755); err != nil {
			t.Fatal(err)
		}
		ws := mustOpen(t, root)
		state, err := ws.AgentsDirState()
		if err != nil {
			t.Fatal(err)
		}
		if state != DirReal {
			t.Fatalf("state = %v, want DirReal", state)
		}
	})

	t.Run("regular file", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)
		writeFile(t, filepath.Join(root, ".agents"), "x")
		ws := mustOpen(t, root)
		state, err := ws.AgentsDirState()
		if err != nil {
			t.Fatal(err)
		}
		if state != DirOther {
			t.Fatalf("state = %v, want DirOther", state)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		t.Parallel()
		root := newGitRepo(t)
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(root, ".agents")); err != nil {
			t.Fatal(err)
		}
		ws := mustOpen(t, root)
		state, err := ws.AgentsDirState()
		if err != nil {
			t.Fatal(err)
		}
		if state != DirOther {
			t.Fatalf("state = %v, want DirOther", state)
		}
	})
}

func TestPaths(t *testing.T) {
	t.Parallel()

	ws := &Workspace{Root: "/repo"}
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "agents", got: ws.AgentsDir(), want: "/repo/.agents"},
		{name: "live", got: ws.LivePath(), want: "/repo/.agents/skills"},
		{name: "stacks", got: ws.StacksDir(), want: "/repo/skill-stacks"},
		{name: "stack", got: ws.StackDir("work"), want: "/repo/skill-stacks/work"},
		{name: "stack skills", got: ws.StackSkillsDir("work"), want: "/repo/skill-stacks/work/.agents/skills"},
		{name: "root lock", got: ws.LockPath(), want: "/repo/skills-lock.json"},
		{name: "stack lock", got: ws.StackLockPath("work"), want: "/repo/skill-stacks/work/skills-lock.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "init", "-q")
	return root
}

func mustOpen(t *testing.T, dir string) *Workspace {
	t.Helper()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
