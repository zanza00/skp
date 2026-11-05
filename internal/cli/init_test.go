package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zanza00/skp/internal/live"
	"github.com/zanza00/skp/internal/workspace"
)

func TestInitImportsRealDirectory(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addUnmanagedSkills(t, "SKILL.md", "# work\n")
	if err := os.Symlink("SKILL.md", filepath.Join(repo.root, ".agents", "skills", "ref")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo.root, ".agents", "other"), "keep")
	writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "{\"lock\":true}\n")

	got := repo.run("init", "work")
	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stdout = %q; stderr = %q", got.code, ExitOK, got.stdout, got.stderr)
	}
	if got.stderr != "" {
		t.Errorf("stderr = %q, want empty", got.stderr)
	}
	if !strings.Contains(got.stdout, "Stack attivo: work") {
		t.Errorf("stdout = %q, want success message", got.stdout)
	}

	target, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatalf("read live link: %v", err)
	}
	if target != live.Target("work") {
		t.Errorf("link target = %q, want %q", target, live.Target("work"))
	}

	skillPath := filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills", "SKILL.md")
	if content := readFile(t, skillPath); content != "# work\n" {
		t.Errorf("imported skill = %q, want %q", content, "# work\n")
	}
	lockPath := filepath.Join(repo.root, "skill-stacks", "work", "skills-lock.json")
	if content := readFile(t, lockPath); content != "{\"lock\":true}\n" {
		t.Errorf("imported lock = %q, want %q", content, "{\"lock\":true}\n")
	}
	if target, err := os.Readlink(filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills", "ref")); err != nil || target != "SKILL.md" {
		t.Errorf("imported symlink = %q, err = %v, want %q", target, err, "SKILL.md")
	}
	if _, err := os.Lstat(filepath.Join(repo.root, "skills-lock.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("root lock still present after init: %v", err)
	}
	if content := readFile(t, filepath.Join(repo.root, ".agents", "other")); content != "keep" {
		t.Errorf(".agents/other = %q, want %q", content, "keep")
	}

	current := repo.run("current")
	if current.code != ExitOK || current.stdout != "work\n" {
		t.Errorf("current: code = %d, stdout = %q, stderr = %q", current.code, current.stdout, current.stderr)
	}
	list := repo.run("list")
	if !strings.Contains(list.stdout, "* work") {
		t.Errorf("list stdout = %q, want active work", list.stdout)
	}
	status := repo.run("status")
	if status.code != ExitOK {
		t.Errorf("status: code = %d, stdout = %q, stderr = %q", status.code, status.stdout, status.stderr)
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestInitWithoutLockFile(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addUnmanagedSkills(t, "SKILL.md", "# work\n")

	got := repo.run("init", "work")
	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitOK, got.stderr)
	}
	if _, err := os.Lstat(filepath.Join(repo.root, "skill-stacks", "work", "skills-lock.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("lock file created without a source lock: %v", err)
	}
	if current := repo.run("current"); current.code != ExitOK || current.stdout != "work\n" {
		t.Errorf("current: code = %d, stdout = %q, stderr = %q", current.code, current.stdout, current.stderr)
	}
}

func TestInitUsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "without stack", args: []string{"init"}},
		{name: "with two stacks", args: []string{"init", "a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := newTestRepo(t).run(tt.args...)
			if got.code != ExitUsage {
				t.Fatalf("code = %d, want %d", got.code, ExitUsage)
			}
			if got.stdout != "" {
				t.Errorf("stdout = %q, want empty", got.stdout)
			}
			if !strings.Contains(got.stderr, "init") {
				t.Errorf("stderr = %q, want init usage", got.stderr)
			}
		})
	}
}

func TestInitRejectsMissingSource(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	got := repo.run("init", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitFailure, got.stderr)
	}
	if !strings.Contains(got.stderr, "nulla da importare") {
		t.Errorf("stderr = %q, want missing source diagnostic", got.stderr)
	}
	if _, err := os.Lstat(filepath.Join(repo.root, "skill-stacks")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("skill-stacks created after failed init: %v", err)
	}
	if _, err := os.Lstat(repo.livePath()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("live path created after failed init: %v", err)
	}
}

func TestInitDoesNotOverwriteExistingStack(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addUnmanagedSkills(t, "SKILL.md", "# incoming\n")
	repo.addStack("work")
	existing := filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills", "SKILL.md")

	got := repo.run("init", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitFailure, got.stderr)
	}
	if !strings.Contains(got.stderr, "esiste già") {
		t.Errorf("stderr = %q, want destination conflict", got.stderr)
	}
	if content := readFile(t, existing); content != "# work\n" {
		t.Errorf("existing stack changed to %q", content)
	}
	if content := readFile(t, filepath.Join(repo.root, ".agents", "skills", "SKILL.md")); content != "# incoming\n" {
		t.Errorf("source skills changed to %q", content)
	}
	if _, err := os.Lstat(repo.livePath()); err != nil {
		t.Fatalf("live path changed after failed init: %v", err)
	}
	if info, err := os.Lstat(repo.livePath()); err != nil || !info.IsDir() {
		t.Errorf("live path is no longer a real directory")
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestInitRollsBackWhenLiveLinkFails(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addUnmanagedSkills(t, "SKILL.md", "# work\n")
	writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "{\"lock\":true}\n")
	writeFile(t, filepath.Join(repo.root, ".agents", "other"), "keep")

	got := repo.runInitWithReplace(func(string, string) error {
		return errors.New("replace failed")
	}, "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d; stdout = %q; stderr = %q", got.code, ExitFailure, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "replace failed") {
		t.Errorf("stderr = %q, want replace failure", got.stderr)
	}

	if content := readFile(t, filepath.Join(repo.root, ".agents", "skills", "SKILL.md")); content != "# work\n" {
		t.Errorf("restored skill = %q, want %q", content, "# work\n")
	}
	if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "{\"lock\":true}\n" {
		t.Errorf("restored lock = %q, want %q", content, "{\"lock\":true}\n")
	}
	if content := readFile(t, filepath.Join(repo.root, ".agents", "other")); content != "keep" {
		t.Errorf(".agents/other = %q, want %q", content, "keep")
	}
	if _, err := os.Lstat(repo.livePath()); err != nil {
		t.Errorf("live path missing after rollback: %v", err)
	}
	if info, err := os.Lstat(repo.livePath()); err != nil || !info.IsDir() {
		t.Errorf("live path is not a real directory after rollback")
	}
	if _, err := os.Lstat(filepath.Join(repo.root, "skill-stacks")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("skill-stacks left behind after rollback: %v", err)
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestInitFromSubdirectory(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addUnmanagedSkills(t, "SKILL.md", "# work\n")
	deep := filepath.Join(repo.root, "sub", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	got := repo.runIn(deep, "init", "work")
	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitOK, got.stderr)
	}
	if target, err := os.Readlink(repo.livePath()); err != nil || target != live.Target("work") {
		t.Errorf("link = %q, err = %v, want %q", target, err, live.Target("work"))
	}
	if current := repo.run("current"); current.code != ExitOK || current.stdout != "work\n" {
		t.Errorf("current: code = %d, stdout = %q, stderr = %q", current.code, current.stdout, current.stderr)
	}
}

func TestInitIsNotIdempotent(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addUnmanagedSkills(t, "SKILL.md", "# work\n")
	writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "{\"lock\":true}\n")

	if got := repo.run("init", "work"); got.code != ExitOK {
		t.Fatalf("first init: code = %d, stderr = %q", got.code, got.stderr)
	}
	before, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}

	got := repo.run("init", "work")
	if got.code != ExitFailure {
		t.Fatalf("second init: code = %d, want %d; stderr = %q", got.code, ExitFailure, got.stderr)
	}
	if !strings.Contains(got.stderr, "già un symlink") {
		t.Errorf("stderr = %q, want already-managed diagnostic", got.stderr)
	}
	after, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("link changed from %q to %q", before, after)
	}
	if content := readFile(t, filepath.Join(repo.root, "skill-stacks", "work", "skills-lock.json")); content != "{\"lock\":true}\n" {
		t.Errorf("lock changed to %q", content)
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestInitImportsTrackedDirectory(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addUnmanagedSkills(t, "SKILL.md", "# work\n")
	runGit(t, repo.root, "add", "-f", ".agents/skills")

	ws, err := workspace.Open(repo.root)
	if err != nil {
		t.Fatal(err)
	}
	if ignored, err := ws.IgnoresLiveLink(); err != nil {
		t.Fatal(err)
	} else if ignored {
		t.Fatal("IgnoresLiveLink() = true for a tracked directory, want false")
	}
	if ignored, err := ws.IgnoresLiveLinkIncludingTracked(); err != nil {
		t.Fatal(err)
	} else if !ignored {
		t.Fatal("IgnoresLiveLinkIncludingTracked() = false, want true")
	}

	got := repo.run("init", "work")
	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stdout = %q; stderr = %q", got.code, ExitOK, got.stdout, got.stderr)
	}
	if target, err := os.Readlink(repo.livePath()); err != nil || target != live.Target("work") {
		t.Errorf("link = %q, err = %v, want %q", target, err, live.Target("work"))
	}
	if current := repo.run("current"); current.code != ExitOK || current.stdout != "work\n" {
		t.Errorf("current: code = %d, stdout = %q, stderr = %q", current.code, current.stdout, current.stderr)
	}
}

func TestInitRejectsUnsafeStateWithoutChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		stack      string
		setup      func(t *testing.T, repo *testRepo)
		wantStderr string
		check      func(t *testing.T, repo *testRepo)
	}{
		{
			name:  "missing live directory",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				if err := os.MkdirAll(filepath.Join(repo.root, ".agents"), 0o755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
			},
			wantStderr: "nulla da importare",
			check: func(t *testing.T, repo *testRepo) {
				if _, err := os.Lstat(filepath.Join(repo.root, "skill-stacks")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("skill-stacks created: %v", err)
				}
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
			},
		},
		{
			name:  "missing agents directory",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
			},
			wantStderr: "nulla da importare",
			check: func(t *testing.T, repo *testRepo) {
				if _, err := os.Lstat(filepath.Join(repo.root, "skill-stacks")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("skill-stacks created: %v", err)
				}
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
			},
		},
		{
			name:  "agents is a file",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				writeFile(t, filepath.Join(repo.root, ".agents"), "agents keep")
			},
			wantStderr: "non è una directory reale",
			check: func(t *testing.T, repo *testRepo) {
				if content := readFile(t, filepath.Join(repo.root, ".agents")); content != "agents keep" {
					t.Errorf(".agents = %q, want %q", content, "agents keep")
				}
			},
		},
		{
			name:  "agents is a symlink",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkillsAt(t, "agents-real", "SKILL.md", "# source\n")
				if err := os.Symlink("agents-real", filepath.Join(repo.root, ".agents")); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
			},
			wantStderr: "non è una directory reale",
			check: func(t *testing.T, repo *testRepo) {
				if content := readFile(t, filepath.Join(repo.root, "agents-real", "skills", "SKILL.md")); content != "# source\n" {
					t.Errorf("skills behind .agents symlink = %q, want %q", content, "# source\n")
				}
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
			},
		},
		{
			name:  "live path is a symlink",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				if err := os.MkdirAll(filepath.Join(repo.root, ".agents"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../skill-stacks/old/.agents/skills", repo.livePath()); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
			},
			wantStderr: "già un symlink",
			check: func(t *testing.T, repo *testRepo) {
				if target, err := os.Readlink(repo.livePath()); err != nil || target != "../skill-stacks/old/.agents/skills" {
					t.Errorf("link = %q, err = %v", target, err)
				}
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
			},
		},
		{
			name:  "live path is a file",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				writeFile(t, repo.livePath(), "live keep")
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
			},
			wantStderr: "file regolare",
			check: func(t *testing.T, repo *testRepo) {
				if content := readFile(t, repo.livePath()); content != "live keep" {
					t.Errorf("live file = %q, want %q", content, "live keep")
				}
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
			},
		},
		{
			name:  "skill-stacks is a file",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkills(t, "SKILL.md", "# source\n")
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
				writeFile(t, filepath.Join(repo.root, "skill-stacks"), "stacks keep")
			},
			wantStderr: "non è una directory reale",
			check: func(t *testing.T, repo *testRepo) {
				if content := readFile(t, filepath.Join(repo.root, "skill-stacks")); content != "stacks keep" {
					t.Errorf("skill-stacks = %q, want %q", content, "stacks keep")
				}
				assertUntouchedSource(t, repo, "# source\n")
			},
		},
		{
			name:  "skill-stacks is a symlink",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkills(t, "SKILL.md", "# source\n")
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
				if err := os.MkdirAll(filepath.Join(repo.root, "stacks-real"), 0o755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(repo.root, "stacks-real", "keep"), "stacks keep")
				if err := os.Symlink("stacks-real", filepath.Join(repo.root, "skill-stacks")); err != nil {
					t.Fatal(err)
				}
			},
			wantStderr: "non è una directory reale",
			check: func(t *testing.T, repo *testRepo) {
				if content := readFile(t, filepath.Join(repo.root, "stacks-real", "keep")); content != "stacks keep" {
					t.Errorf("symlink target = %q, want %q", content, "stacks keep")
				}
				assertUntouchedSource(t, repo, "# source\n")
			},
		},
		{
			name:  "destination stack exists",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkills(t, "SKILL.md", "# source\n")
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
				repo.addStack("work")
			},
			wantStderr: "esiste già",
			check: func(t *testing.T, repo *testRepo) {
				if content := readFile(t, filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills", "SKILL.md")); content != "# work\n" {
					t.Errorf("existing stack = %q, want %q", content, "# work\n")
				}
				if content := readFile(t, filepath.Join(repo.root, "skill-stacks", "work", "skills-lock.json")); content != "{\"stack\":\"work\"}\n" {
					t.Errorf("existing stack lock = %q", content)
				}
				assertUntouchedSource(t, repo, "# source\n")
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
			},
		},
		{
			name:  "destination stack is a file",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkills(t, "SKILL.md", "# source\n")
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
				writeFile(t, filepath.Join(repo.root, "skill-stacks", "work"), "work keep")
			},
			wantStderr: "esiste già",
			check: func(t *testing.T, repo *testRepo) {
				if content := readFile(t, filepath.Join(repo.root, "skill-stacks", "work")); content != "work keep" {
					t.Errorf("destination file = %q, want %q", content, "work keep")
				}
				assertUntouchedSource(t, repo, "# source\n")
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
			},
		},
		{
			name:  "root lock is a directory",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkills(t, "SKILL.md", "# source\n")
				writeFile(t, filepath.Join(repo.root, "skills-lock.json", "keep"), "lock keep")
			},
			wantStderr: "non è un file regolare",
			check: func(t *testing.T, repo *testRepo) {
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json", "keep")); content != "lock keep" {
					t.Errorf("lock directory = %q, want %q", content, "lock keep")
				}
				assertUntouchedSource(t, repo, "# source\n")
				if _, err := os.Lstat(filepath.Join(repo.root, "skill-stacks")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("skill-stacks created: %v", err)
				}
			},
		},
		{
			name:  "root lock is a symlink",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkills(t, "SKILL.md", "# source\n")
				writeFile(t, filepath.Join(repo.root, "lock-real"), "lock keep")
				if err := os.Symlink("lock-real", filepath.Join(repo.root, "skills-lock.json")); err != nil {
					t.Fatal(err)
				}
			},
			wantStderr: "non è un file regolare",
			check: func(t *testing.T, repo *testRepo) {
				target, err := os.Readlink(filepath.Join(repo.root, "skills-lock.json"))
				if err != nil || target != "lock-real" {
					t.Errorf("lock link = %q, err = %v", target, err)
				}
				if content := readFile(t, filepath.Join(repo.root, "lock-real")); content != "lock keep" {
					t.Errorf("lock target = %q, want %q", content, "lock keep")
				}
				assertUntouchedSource(t, repo, "# source\n")
			},
		},
		{
			name:  "live path not ignored by Git",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkills(t, "SKILL.md", "# source\n")
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
				if err := os.Remove(filepath.Join(repo.root, ".gitignore")); err != nil {
					t.Fatal(err)
				}
			},
			wantStderr: "/.agents/skills",
			check: func(t *testing.T, repo *testRepo) {
				assertUntouchedSource(t, repo, "# source\n")
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
				if _, err := os.Lstat(filepath.Join(repo.root, "skill-stacks")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("skill-stacks created: %v", err)
				}
			},
		},
		{
			name:  "invalid stack name",
			stack: "../work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addUnmanagedSkills(t, "SKILL.md", "# source\n")
				writeFile(t, filepath.Join(repo.root, "skills-lock.json"), "lock keep")
			},
			wantStderr: "non ammesso",
			check: func(t *testing.T, repo *testRepo) {
				assertUntouchedSource(t, repo, "# source\n")
				if content := readFile(t, filepath.Join(repo.root, "skills-lock.json")); content != "lock keep" {
					t.Errorf("root lock = %q, want %q", content, "lock keep")
				}
				if _, err := os.Lstat(filepath.Join(repo.root, "skill-stacks")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("skill-stacks created: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestRepo(t)
			tt.setup(t, repo)

			got := repo.run("init", tt.stack)
			if got.code != ExitFailure {
				t.Fatalf("code = %d, want %d; stdout = %q; stderr = %q", got.code, ExitFailure, got.stdout, got.stderr)
			}
			if got.stdout != "" {
				t.Errorf("stdout = %q, want empty", got.stdout)
			}
			if !strings.Contains(got.stderr, tt.wantStderr) {
				t.Errorf("stderr = %q, want substring %q", got.stderr, tt.wantStderr)
			}
			tt.check(t, repo)
			if info, err := os.Lstat(filepath.Join(repo.root, ".agents")); err == nil && info.IsDir() {
				assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
			}
		})
	}
}

func (r *testRepo) addUnmanagedSkillsAt(t *testing.T, dir, name, content string) {
	t.Helper()
	writeFile(t, filepath.Join(r.root, dir, "skills", name), content)
}

func assertUntouchedSource(t *testing.T, repo *testRepo, want string) {
	t.Helper()
	info, err := os.Lstat(repo.livePath())
	if err != nil {
		t.Fatalf("source live path missing: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("source live path is not a real directory")
	}
	if content := readFile(t, filepath.Join(repo.livePath(), "SKILL.md")); content != want {
		t.Errorf("source skill = %q, want %q", content, want)
	}
}

func (r *testRepo) addUnmanagedSkills(t *testing.T, name, content string) {
	t.Helper()
	writeFile(t, filepath.Join(r.root, ".agents", "skills", name), content)
}

func (r *testRepo) runInitWithReplace(replace func(string, string) error, name string) result {
	r.t.Helper()
	ws, err := workspace.Open(r.root)
	if err != nil {
		r.t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runInitWith(ws, name, &stdout, &stderr, replace)
	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
}
