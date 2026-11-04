package stack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

var (
	// ErrInvalidName rejects unsafe or unsupported stack names.
	ErrInvalidName = errors.New("nome stack non ammesso")
	// ErrMissingSkills means the stack has no .agents/skills directory.
	ErrMissingSkills = errors.New(".agents/skills mancante")
	// ErrSkillsNotDir means .agents/skills exists but is not a real directory.
	ErrSkillsNotDir = errors.New(".agents/skills non è una directory")
)

// ValidName reports whether name is an allowed stack name. Path separators,
// absolute paths, "." and ".." are rejected.
func ValidName(name string) bool {
	return namePattern.MatchString(name)
}

// Entry is one directory found immediately under skill-stacks/.
type Entry struct {
	Name       string
	Selectable bool
	Reason     string
}

// Discover lists the stack directories in stacksDir, sorted by name. A missing
// stacksDir means no stacks; any other read error is returned. Only real
// directories are considered stacks.
func Discover(stacksDir string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(stacksDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("leggere %s: %w", stacksDir, err)
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		entry := Entry{Name: dirEntry.Name(), Selectable: true}
		if err := Validate(stacksDir, entry.Name); err != nil {
			entry.Selectable = false
			entry.Reason = Reason(err)
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

// Validate checks that name is allowed and that the stack has a real
// .agents/skills directory. It never follows a symlink at that path.
func Validate(stacksDir, name string) error {
	if !ValidName(name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	skillsDir := filepath.Join(stacksDir, name, ".agents", "skills")
	info, err := os.Lstat(skillsDir)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w nello stack %q", ErrMissingSkills, name)
	}
	if err != nil {
		return fmt.Errorf("verificare .agents/skills dello stack %q: %w", name, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w nello stack %q", ErrSkillsNotDir, name)
	}
	return nil
}

// Reason returns a short reason suitable for list output.
func Reason(err error) string {
	switch {
	case errors.Is(err, ErrInvalidName):
		return "nome non ammesso"
	case errors.Is(err, ErrMissingSkills):
		return ".agents/skills mancante"
	case errors.Is(err, ErrSkillsNotDir):
		return ".agents/skills non è una directory"
	default:
		return "non selezionabile"
	}
}
