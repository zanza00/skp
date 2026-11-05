package live

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zanza00/skp/internal/stack"
)

const targetPrefix = "../skill-stacks/"

// Kind is the type of the live path, detected without following symlinks.
type Kind int

const (
	Absent Kind = iota
	Managed
	UnexpectedSymlink
	Directory
	File
	Other
)

// State is the inspection result for the live path.
type State struct {
	Kind      Kind
	RawTarget string
	Stack     string
}

// Target is the relative target expected for a stack. name must be valid.
func Target(name string) string {
	return targetPrefix + name + "/.agents/skills"
}

// Inspect classifies path with Lstat. Only the exact relative target produced
// by Target is considered managed; no normalisation is applied to other
// targets.
func Inspect(path string) (State, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Kind: Absent}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("ispezionare .agents/skills: %w", err)
	}

	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			return State{}, fmt.Errorf("leggere .agents/skills: %w", err)
		}
		name, ok := parseTarget(target)
		if !ok {
			return State{Kind: UnexpectedSymlink, RawTarget: target}, nil
		}
		return State{Kind: Managed, RawTarget: target, Stack: name}, nil
	case info.IsDir():
		return State{Kind: Directory}, nil
	case info.Mode().IsRegular():
		return State{Kind: File}, nil
	default:
		return State{Kind: Other}, nil
	}
}

// Resolve returns the lexical destination of a relative link target. It does
// not follow symlinks and does not require the destination to exist.
func Resolve(linkPath, target string) string {
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(linkPath), target))
}

// Replace atomically replaces livePath with a symlink to target. The temporary
// symlink lives in the same directory, so the final step is a single rename.
// On failure only the temporary symlink is removed. Concurrency hardening with
// os.Root is deferred: the MVP checks path types before writing and never
// removes anything recursively.
func Replace(livePath, target string) error {
	tempPath, err := createTempSymlink(filepath.Dir(livePath), target)
	if err != nil {
		return err
	}

	if err := os.Rename(tempPath, livePath); err != nil {
		removeErr := os.Remove(tempPath)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("%w (rimozione del temporaneo fallita: %v)", err, removeErr)
		}
		return err
	}
	return nil
}

func createTempSymlink(dir, target string) (string, error) {
	// os.CreateTemp cannot create symlinks. A crypto-random name plus
	// os.Symlink gives the same guarantees: the name is unpredictable and
	// creation fails if anything already exists at that path.
	var random [8]byte
	for range 10 {
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("generare il nome del symlink temporaneo: %w", err)
		}
		path := filepath.Join(dir, ".skp-tmp-"+hex.EncodeToString(random[:]))
		err := os.Symlink(target, path)
		if err == nil {
			return path, nil
		}
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return "", fmt.Errorf("creare il symlink temporaneo: %w", err)
	}
	return "", errors.New("creare il symlink temporaneo: troppi conflitti di nome")
}

// parseTarget accepts exactly ../skill-stacks/<valid-name>/.agents/skills.
func parseTarget(target string) (string, bool) {
	parts := strings.Split(target, "/")
	if len(parts) != 5 {
		return "", false
	}
	if parts[0] != ".." || parts[1] != "skill-stacks" || parts[3] != ".agents" || parts[4] != "skills" {
		return "", false
	}
	if !stack.ValidName(parts[2]) {
		return "", false
	}
	return parts[2], true
}
