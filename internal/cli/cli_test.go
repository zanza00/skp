package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"codefloe.com/zanza00/skills-stack/internal/live"
)

// TestMain isolates tests from the user's global Git configuration, which
// could otherwise add ignore rules or change repository defaults.
func TestMain(m *testing.M) {
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	_ = os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	os.Exit(m.Run())
}

func TestRunWithoutArguments(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	got := repo.run()

	if got.code != ExitUsage {
		t.Fatalf("code = %d, want %d", got.code, ExitUsage)
	}
	if got.stdout != "" {
		t.Errorf("stdout = %q, want empty", got.stdout)
	}
	if !strings.Contains(got.stderr, "Uso:") {
		t.Errorf("stderr = %q, want usage", got.stderr)
	}
}

func TestHelp(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			t.Parallel()
			repo := newTestRepo(t)
			got := repo.run(arg)

			if got.code != ExitOK {
				t.Fatalf("code = %d, want %d", got.code, ExitOK)
			}
			if !strings.Contains(got.stdout, "Uso:") {
				t.Errorf("stdout = %q, want help", got.stdout)
			}
			if got.stderr != "" {
				t.Errorf("stderr = %q, want empty", got.stderr)
			}
		})
	}
}

func TestUnknownCommand(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	got := repo.run("switch", "work")

	if got.code != ExitUsage {
		t.Fatalf("code = %d, want %d", got.code, ExitUsage)
	}
	if !strings.Contains(got.stderr, "comando sconosciuto") {
		t.Errorf("stderr = %q, want unknown command", got.stderr)
	}
}

func TestUsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "list with argument", args: []string{"list", "extra"}},
		{name: "current with argument", args: []string{"current", "extra"}},
		{name: "status with argument", args: []string{"status", "extra"}},
		{name: "use without stack", args: []string{"use"}},
		{name: "use with two stacks", args: []string{"use", "a", "b"}},
		{name: "help with argument", args: []string{"-h", "x"}},
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
			if got.stderr == "" {
				t.Error("stderr is empty, want usage diagnostic")
			}
		})
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	repo.addStack("personal")
	repo.addStack("_bad")
	if err := os.MkdirAll(filepath.Join(repo.root, "skill-stacks", "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo.root, "skill-stacks", "notes.txt"), "not a stack")
	repo.symlinkLive(live.Target("work"))

	got := repo.run("list")

	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitOK, got.stderr)
	}
	want := "! _bad (nome non ammesso)\n" +
		"! broken (.agents/skills mancante)\n" +
		"  personal\n" +
		"* work\n"
	if got.stdout != want {
		t.Errorf("stdout = %q, want %q", got.stdout, want)
	}
}

func TestListNoStacks(t *testing.T) {
	t.Parallel()

	got := newTestRepo(t).run("list")

	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d", got.code, ExitOK)
	}
	if got.stdout != "Nessuno stack trovato.\n" {
		t.Errorf("stdout = %q", got.stdout)
	}
}

func TestListMarksActiveStackEvenWhenBroken(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	repo.symlinkLive(live.Target("work"))
	skillsDir := filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills")
	if err := os.RemoveAll(skillsDir); err != nil {
		t.Fatal(err)
	}

	got := repo.run("list")
	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitOK, got.stderr)
	}
	want := "* work (.agents/skills mancante)\n"
	if got.stdout != want {
		t.Errorf("stdout = %q, want %q", got.stdout, want)
	}
}

func TestCurrent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, repo *testRepo)
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name: "managed link",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
				repo.symlinkLive(live.Target("work"))
			},
			wantCode:   ExitOK,
			wantStdout: "work\n",
		},
		{
			name: "missing link",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
			},
			wantCode:   ExitNoStack,
			wantStderr: "Nessuno stack attivo",
		},
		{
			name:       "missing agents directory",
			setup:      func(t *testing.T, repo *testRepo) {},
			wantCode:   ExitNoStack,
			wantStderr: "Nessuno stack attivo",
		},
		{
			name: "absolute symlink",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
				repo.symlinkLive(filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills"))
			},
			wantCode:   ExitFailure,
			wantStderr: "target non gestito",
		},
		{
			name: "unexpected relative symlink",
			setup: func(t *testing.T, repo *testRepo) {
				repo.symlinkLive("../elsewhere")
			},
			wantCode:   ExitFailure,
			wantStderr: "target non gestito",
		},
		{
			name: "broken managed link",
			setup: func(t *testing.T, repo *testRepo) {
				repo.symlinkLive(live.Target("gone"))
			},
			wantCode:   ExitFailure,
			wantStderr: "non è valido",
		},
		{
			name: "real directory",
			setup: func(t *testing.T, repo *testRepo) {
				if err := os.MkdirAll(repo.livePath(), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:   ExitFailure,
			wantStderr: "directory reale",
		},
		{
			name: "regular file",
			setup: func(t *testing.T, repo *testRepo) {
				writeFile(t, repo.livePath(), "sentinel")
			},
			wantCode:   ExitFailure,
			wantStderr: "file regolare",
		},
		{
			name: "invalid stack name in target",
			setup: func(t *testing.T, repo *testRepo) {
				repo.symlinkLive("../skill-stacks/_bad/.agents/skills")
			},
			wantCode:   ExitFailure,
			wantStderr: "target non gestito",
		},
		{
			name: "agents is a symlink",
			setup: func(t *testing.T, repo *testRepo) {
				if err := os.Symlink(t.TempDir(), filepath.Join(repo.root, ".agents")); err != nil {
					t.Fatal(err)
				}
			},
			wantCode:   ExitFailure,
			wantStderr: "non è una directory reale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestRepo(t)
			tt.setup(t, repo)
			got := repo.run("current")

			if got.code != tt.wantCode {
				t.Fatalf("code = %d, want %d; stderr = %q", got.code, tt.wantCode, got.stderr)
			}
			if got.stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", got.stdout, tt.wantStdout)
			}
			if tt.wantStderr != "" && !strings.Contains(got.stderr, tt.wantStderr) {
				t.Errorf("stderr = %q, want substring %q", got.stderr, tt.wantStderr)
			}
			if tt.wantStderr == "" && got.stderr != "" {
				t.Errorf("stderr = %q, want empty", got.stderr)
			}
		})
	}
}

func TestCurrentFromSubdirectory(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	repo.symlinkLive(live.Target("work"))
	deep := filepath.Join(repo.root, "sub", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	got := repo.runIn(deep, "current")
	if got.code != ExitOK || got.stdout != "work\n" {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", got.code, got.stdout, got.stderr)
	}
}

func TestStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, repo *testRepo)
		wantCode   int
		wantOutput []string
	}{
		{
			name: "valid active stack",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
				repo.symlinkLive(live.Target("work"))
			},
			wantCode: ExitOK,
			wantOutput: []string{
				"Percorso live: symlink",
				"Stack corrente: work",
				"Target: ../skill-stacks/work/.agents/skills",
				"Destinazione: valida",
				"Esclusione Git: presente",
			},
		},
		{
			name: "absent link with ignore rule",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
			},
			wantCode: ExitOK,
			wantOutput: []string{
				"Percorso live: assente",
				"Stack corrente: nessuno",
				"Esclusione Git: presente",
			},
		},
		{
			name: "absent link without ignore rule",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
				if err := os.Remove(filepath.Join(repo.root, ".gitignore")); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: ExitFailure,
			wantOutput: []string{
				"Percorso live: assente",
				"Esclusione Git: assente",
			},
		},
		{
			name: "broken managed link",
			setup: func(t *testing.T, repo *testRepo) {
				repo.symlinkLive(live.Target("gone"))
			},
			wantCode: ExitFailure,
			wantOutput: []string{
				"Percorso live: symlink",
				"Stack corrente: gone",
				"Destinazione: non valida",
			},
		},
		{
			name: "absolute symlink",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
				repo.symlinkLive(filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills"))
			},
			wantCode: ExitFailure,
			wantOutput: []string{
				"Percorso live: symlink non gestito",
				"Destinazione: non gestita",
			},
		},
		{
			name: "real directory",
			setup: func(t *testing.T, repo *testRepo) {
				if err := os.MkdirAll(repo.livePath(), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: ExitFailure,
			wantOutput: []string{
				"Percorso live: directory",
			},
		},
		{
			name: "regular file",
			setup: func(t *testing.T, repo *testRepo) {
				writeFile(t, repo.livePath(), "sentinel")
			},
			wantCode: ExitFailure,
			wantOutput: []string{
				"Percorso live: file",
			},
		},
		{
			name: "agents is a file",
			setup: func(t *testing.T, repo *testRepo) {
				writeFile(t, filepath.Join(repo.root, ".agents"), "sentinel")
			},
			wantCode: ExitFailure,
			wantOutput: []string{
				"Percorso live: n/d",
				"Directory .agents: non è una directory reale",
			},
		},
		{
			name: "agents is a symlink",
			setup: func(t *testing.T, repo *testRepo) {
				if err := os.Symlink(t.TempDir(), filepath.Join(repo.root, ".agents")); err != nil {
					t.Fatal(err)
				}
			},
			wantCode: ExitFailure,
			wantOutput: []string{
				"Percorso live: n/d",
				"Directory .agents: non è una directory reale",
				"Esclusione Git: non verificabile",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestRepo(t)
			tt.setup(t, repo)
			got := repo.run("status")

			if got.code != tt.wantCode {
				t.Fatalf("code = %d, want %d; stdout = %q; stderr = %q", got.code, tt.wantCode, got.stdout, got.stderr)
			}
			if !strings.Contains(got.stdout, "Repository: "+repo.root) {
				t.Errorf("stdout = %q, want repository root %q", got.stdout, repo.root)
			}
			for _, want := range tt.wantOutput {
				if !strings.Contains(got.stdout, want) {
					t.Errorf("stdout = %q, want substring %q", got.stdout, want)
				}
			}
		})
	}
}

func TestStatusDoesNotRepair(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.symlinkLive(live.Target("gone"))
	before, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}

	got := repo.run("status")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d", got.code, ExitFailure)
	}

	after, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("status changed link from %q to %q", before, after)
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestUseCreatesLink(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	skillPath := filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills", "SKILL.md")
	lockPath := filepath.Join(repo.root, "skill-stacks", "work", "skills-lock.json")
	skillBefore := readFile(t, skillPath)
	lockBefore := readFile(t, lockPath)

	got := repo.run("use", "work")
	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitOK, got.stderr)
	}
	if !strings.Contains(got.stdout, "Stack attivo: work") {
		t.Errorf("stdout = %q, want success message", got.stdout)
	}
	if !strings.Contains(got.stdout, "riavviala") {
		t.Errorf("stdout = %q, want agent restart reminder", got.stdout)
	}

	target, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if target != live.Target("work") {
		t.Errorf("link target = %q, want %q", target, live.Target("work"))
	}
	if got := readFile(t, skillPath); got != skillBefore {
		t.Errorf("SKILL.md changed from %q to %q", skillBefore, got)
	}
	if got := readFile(t, lockPath); got != lockBefore {
		t.Errorf("skills-lock.json changed from %q to %q", lockBefore, got)
	}

	current := repo.run("current")
	if current.code != ExitOK || current.stdout != "work\n" {
		t.Errorf("current: code = %d, stdout = %q, stderr = %q", current.code, current.stdout, current.stderr)
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestUseIsIdempotent(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	if got := repo.run("use", "work"); got.code != ExitOK {
		t.Fatalf("first use: code = %d, stderr = %q", got.code, got.stderr)
	}
	before, err := os.Lstat(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}

	got := repo.run("use", "work")
	if got.code != ExitOK {
		t.Fatalf("second use: code = %d, stderr = %q", got.code, got.stderr)
	}
	after, err := os.Lstat(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Error("second use rewrote the live link")
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestUseSwitchesStacksWithoutTouchingThem(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("a")
	repo.addStack("b")
	writeFile(t, filepath.Join(repo.root, ".agents", "other"), "keep")

	if got := repo.run("use", "a"); got.code != ExitOK {
		t.Fatalf("use a: code = %d, stderr = %q", got.code, got.stderr)
	}
	aSkillPath := filepath.Join(repo.root, "skill-stacks", "a", ".agents", "skills", "SKILL.md")
	aSkill := readFile(t, aSkillPath)
	aLock := readFile(t, filepath.Join(repo.root, "skill-stacks", "a", "skills-lock.json"))
	bSkill := readFile(t, filepath.Join(repo.root, "skill-stacks", "b", ".agents", "skills", "SKILL.md"))

	if got := repo.run("use", "b"); got.code != ExitOK {
		t.Fatalf("use b: code = %d, stderr = %q", got.code, got.stderr)
	}

	target, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if target != live.Target("b") {
		t.Errorf("link target = %q, want %q", target, live.Target("b"))
	}
	if got := readFile(t, aSkillPath); got != aSkill {
		t.Errorf("stack a SKILL.md changed from %q to %q", aSkill, got)
	}
	if got := readFile(t, filepath.Join(repo.root, "skill-stacks", "a", "skills-lock.json")); got != aLock {
		t.Errorf("stack a lock changed from %q to %q", aLock, got)
	}
	if got := readFile(t, filepath.Join(repo.root, "skill-stacks", "b", ".agents", "skills", "SKILL.md")); got != bSkill {
		t.Errorf("stack b SKILL.md changed from %q to %q", bSkill, got)
	}
	if got := readFile(t, filepath.Join(repo.root, ".agents", "other")); got != "keep" {
		t.Errorf(".agents/other changed to %q", got)
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestUseFromSubdirectory(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	deep := filepath.Join(repo.root, "sub", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	got := repo.runIn(deep, "use", "work")
	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitOK, got.stderr)
	}
	target, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if target != live.Target("work") {
		t.Errorf("link target = %q, want %q", target, live.Target("work"))
	}
}

func TestUseRejectsInvalidStack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		stack      string
		setup      func(t *testing.T, repo *testRepo)
		wantStderr string
	}{
		{
			name:  "missing stack",
			stack: "personal",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
			},
			wantStderr: "mancante",
		},
		{
			name:  "empty stack directory",
			stack: "broken",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
				if err := os.MkdirAll(filepath.Join(repo.root, "skill-stacks", "broken"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantStderr: "mancante",
		},
		{
			name:  "skills is a file",
			stack: "file",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
				writeFile(t, filepath.Join(repo.root, "skill-stacks", "file", ".agents", "skills"), "x")
			},
			wantStderr: "non è una directory",
		},
		{
			name:  "path traversal",
			stack: "../work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
			},
			wantStderr: "non ammesso",
		},
		{
			name:  "invalid leading character",
			stack: "_work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("_work")
			},
			wantStderr: "non ammesso",
		},
		{
			name:  "live link not ignored",
			stack: "work",
			setup: func(t *testing.T, repo *testRepo) {
				repo.addStack("work")
				if err := os.Remove(filepath.Join(repo.root, ".gitignore")); err != nil {
					t.Fatal(err)
				}
			},
			wantStderr: "/.agents/skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestRepo(t)
			tt.setup(t, repo)

			got := repo.run("use", tt.stack)
			if got.code != ExitFailure {
				t.Fatalf("code = %d, want %d; stdout = %q; stderr = %q", got.code, ExitFailure, got.stdout, got.stderr)
			}
			if got.stdout != "" {
				t.Errorf("stdout = %q, want empty", got.stdout)
			}
			if !strings.Contains(got.stderr, tt.wantStderr) {
				t.Errorf("stderr = %q, want substring %q", got.stderr, tt.wantStderr)
			}
			if _, err := os.Lstat(repo.livePath()); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("live path exists after failed use: %v", err)
			}
			assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
		})
	}
}

func TestUseKeepsRealDirectory(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	if err := os.MkdirAll(repo.livePath(), 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(repo.livePath(), "keep")
	writeFile(t, sentinel, "keep")

	got := repo.run("use", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d", got.code, ExitFailure)
	}
	if !strings.Contains(got.stderr, "directory reale") {
		t.Errorf("stderr = %q", got.stderr)
	}
	info, err := os.Lstat(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatal("live path is no longer a directory")
	}
	if content := readFile(t, sentinel); content != "keep" {
		t.Errorf("sentinel = %q, want %q", content, "keep")
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestUseKeepsRegularFile(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	writeFile(t, repo.livePath(), "keep")

	got := repo.run("use", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d", got.code, ExitFailure)
	}
	if !strings.Contains(got.stderr, "file regolare") {
		t.Errorf("stderr = %q", got.stderr)
	}
	if content := readFile(t, repo.livePath()); content != "keep" {
		t.Errorf("file content = %q, want %q", content, "keep")
	}
	assertNoTempLinks(t, filepath.Join(repo.root, ".agents"))
}

func TestUseRejectsAbsoluteSymlink(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	absolute := filepath.Join(repo.root, "skill-stacks", "work", ".agents", "skills")
	repo.symlinkLive(absolute)

	got := repo.run("use", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d", got.code, ExitFailure)
	}
	if !strings.Contains(got.stderr, "target non gestito") {
		t.Errorf("stderr = %q", got.stderr)
	}
	target, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if target != absolute {
		t.Errorf("link target = %q, want %q", target, absolute)
	}
}

func TestUseRejectsUnexpectedRelativeSymlink(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	repo.symlinkLive("../elsewhere")

	got := repo.run("use", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d", got.code, ExitFailure)
	}
	target, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if target != "../elsewhere" {
		t.Errorf("link target = %q, want %q", target, "../elsewhere")
	}
}

func TestUseRejectsBrokenUnrecognizedSymlink(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	repo.symlinkLive("../../gone")

	got := repo.run("use", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d", got.code, ExitFailure)
	}
	target, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if target != "../../gone" {
		t.Errorf("link target = %q, want %q", target, "../../gone")
	}
}

func TestUseRepairsBrokenManagedLink(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("new")
	repo.symlinkLive(live.Target("old"))

	got := repo.run("use", "new")
	if got.code != ExitOK {
		t.Fatalf("code = %d, want %d; stderr = %q", got.code, ExitOK, got.stderr)
	}
	target, err := os.Readlink(repo.livePath())
	if err != nil {
		t.Fatal(err)
	}
	if target != live.Target("new") {
		t.Errorf("link target = %q, want %q", target, live.Target("new"))
	}
}

func TestUseRejectsAgentsFile(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	writeFile(t, filepath.Join(repo.root, ".agents"), "keep")

	got := repo.run("use", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d", got.code, ExitFailure)
	}
	if !strings.Contains(got.stderr, "non è una directory reale") {
		t.Errorf("stderr = %q", got.stderr)
	}
	if content := readFile(t, filepath.Join(repo.root, ".agents")); content != "keep" {
		t.Errorf(".agents content = %q, want %q", content, "keep")
	}
}

func TestUseRejectsAgentsSymlink(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	repo.addStack("work")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repo.root, ".agents")); err != nil {
		t.Fatal(err)
	}

	got := repo.run("use", "work")
	if got.code != ExitFailure {
		t.Fatalf("code = %d, want %d", got.code, ExitFailure)
	}
	if !strings.Contains(got.stderr, "non è una directory reale") {
		t.Errorf("stderr = %q", got.stderr)
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("outside directory was modified: %v", entries)
	}
}

type testRepo struct {
	t    *testing.T
	root string
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "init", "-q")
	writeFile(t, filepath.Join(root, ".gitignore"), "/.agents/skills\n")
	return &testRepo{t: t, root: root}
}

func (r *testRepo) addStack(name string) {
	r.t.Helper()
	writeFile(r.t, filepath.Join(r.root, "skill-stacks", name, ".agents", "skills", "SKILL.md"), "# "+name+"\n")
	writeFile(r.t, filepath.Join(r.root, "skill-stacks", name, "skills-lock.json"), "{\"stack\":\""+name+"\"}\n")
}

func (r *testRepo) symlinkLive(target string) {
	r.t.Helper()
	if err := os.MkdirAll(filepath.Join(r.root, ".agents"), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.Remove(r.livePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		r.t.Fatal(err)
	}
	if err := os.Symlink(target, r.livePath()); err != nil {
		r.t.Fatal(err)
	}
}

func (r *testRepo) livePath() string {
	return filepath.Join(r.root, ".agents", "skills")
}

type result struct {
	code   int
	stdout string
	stderr string
}

func (r *testRepo) run(args ...string) result {
	r.t.Helper()
	return r.runIn(r.root, args...)
}

func (r *testRepo) runIn(dir string, args ...string) result {
	r.t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(dir, args, &stdout, &stderr)
	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func assertNoTempLinks(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".skills-stack-tmp-") {
			t.Errorf("temporary link left behind: %s", entry.Name())
		}
	}
}
