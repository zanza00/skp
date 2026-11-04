package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Workspace is a Git work tree that can host skill stacks.
type Workspace struct {
	Root string
}

// Open locates the Git work tree containing start. An empty start means the
// current directory. The returned root is the one reported by Git, so commands
// work from any subdirectory of the work tree.
func Open(start string) (*Workspace, error) {
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("determinare la directory corrente: %w", err)
		}
	}

	abs, err := filepath.Abs(start)
	if err != nil {
		return nil, fmt.Errorf("risolvere %q: %w", start, err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("il comando git non è disponibile nel PATH")
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("%s non è un work tree Git: %s", abs, detail)
	}

	root := strings.TrimSpace(stdout.String())
	if root == "" {
		return nil, fmt.Errorf("Git non ha restituito la root del work tree per %s", abs)
	}
	return &Workspace{Root: filepath.Clean(root)}, nil
}

// AgentsDir is the directory that contains the live link.
func (w *Workspace) AgentsDir() string {
	return filepath.Join(w.Root, ".agents")
}

// LivePath is the only path managed by the CLI.
func (w *Workspace) LivePath() string {
	return filepath.Join(w.AgentsDir(), "skills")
}

// StacksDir contains one directory per stack.
func (w *Workspace) StacksDir() string {
	return filepath.Join(w.Root, "skill-stacks")
}

// StackDir is the directory of a stack. The caller must validate name first.
func (w *Workspace) StackDir(name string) string {
	return filepath.Join(w.StacksDir(), name)
}

// StackSkillsDir is the skills directory of a stack. The caller must validate
// name first.
func (w *Workspace) StackSkillsDir(name string) string {
	return filepath.Join(w.StackDir(name), ".agents", "skills")
}

// DirState classifies one of the layout directories without following
// symlinks.
type DirState int

const (
	DirAbsent DirState = iota
	DirReal
	DirOther
)

// AgentsDirState classifies .agents with Lstat, so a symlink is never
// followed.
func (w *Workspace) AgentsDirState() (DirState, error) {
	info, err := os.Lstat(w.AgentsDir())
	if errors.Is(err, os.ErrNotExist) {
		return DirAbsent, nil
	}
	if err != nil {
		return DirOther, fmt.Errorf("ispezionare .agents: %w", err)
	}
	if info.IsDir() {
		return DirReal, nil
	}
	return DirOther, nil
}

// IgnoresLiveLink reports whether Git excludes .agents/skills. A tracked live
// link is reported as not ignored, because it must not be committed.
func (w *Workspace) IgnoresLiveLink() (bool, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", "-C", w.Root, "check-ignore", "-q", "--", ".agents/skills")
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return false, errors.New("il comando git non è disponibile nel PATH")
	}

	detail := strings.TrimSpace(stderr.String())
	if detail == "" {
		detail = err.Error()
	}
	return false, fmt.Errorf("verificare l'esclusione Git di .agents/skills: %s", detail)
}
