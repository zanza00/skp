package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zanza00/skp/internal/live"
	"github.com/zanza00/skp/internal/stack"
	"github.com/zanza00/skp/internal/workspace"
)

// runInit converts an existing .agents/skills directory into a managed stack.
// It only moves the directory and the optional lock file; it never reads or
// rewrites their contents.
func runInit(ws *workspace.Workspace, name string, stdout, stderr io.Writer) int {
	return runInitWith(ws, name, stdout, stderr, live.Replace)
}

// runInitWith is runInit with an injectable link replacement, so tests can
// exercise the rollback path deterministically.
func runInitWith(ws *workspace.Workspace, name string, stdout, stderr io.Writer, replace func(string, string) error) int {
	if !stack.ValidName(name) {
		return fail(stderr, "nome stack non ammesso: %q", name)
	}

	agentsState, err := ws.AgentsDirState()
	if err != nil {
		return fail(stderr, "%v", err)
	}
	switch agentsState {
	case workspace.DirAbsent:
		return fail(stderr, "nulla da importare: .agents/skills non esiste")
	case workspace.DirReal:
		// Continue: the source must be a real directory inside it.
	default:
		return fail(stderr, ".agents esiste ma non è una directory reale: intervento manuale richiesto")
	}

	liveState, err := live.Inspect(ws.LivePath())
	if err != nil {
		return fail(stderr, "%v", err)
	}
	switch liveState.Kind {
	case live.Directory:
		// The only importable state.
	case live.Absent:
		return fail(stderr, "nulla da importare: .agents/skills non esiste")
	case live.Managed, live.UnexpectedSymlink:
		return fail(stderr, ".agents/skills è già un symlink: usa \"skills-stack use <stack>\" per cambiare stack")
	case live.File:
		return fail(stderr, ".agents/skills è un file regolare: intervento manuale richiesto")
	default:
		return fail(stderr, ".agents/skills non è una directory importabile: intervento manuale richiesto")
	}

	stacksState, err := inspectPath(ws.StacksDir())
	if err != nil {
		return fail(stderr, "%v", err)
	}
	if stacksState != pathAbsent && stacksState != pathRealDir {
		return fail(stderr, "skill-stacks esiste ma non è una directory reale: intervento manuale richiesto")
	}

	stackState, err := inspectPath(ws.StackDir(name))
	if err != nil {
		return fail(stderr, "%v", err)
	}
	if stackState != pathAbsent {
		return fail(stderr, "la destinazione skill-stacks/%s esiste già: init non unisce né sovrascrive stack", name)
	}

	lockState, err := inspectPath(ws.LockPath())
	if err != nil {
		return fail(stderr, "%v", err)
	}
	if lockState != pathAbsent && lockState != pathRegularFile {
		return fail(stderr, "skills-lock.json non è un file regolare: intervento manuale richiesto")
	}

	ignored, err := ws.IgnoresLiveLinkIncludingTracked()
	if err != nil {
		return fail(stderr, "%v", err)
	}
	if !ignored {
		fmt.Fprint(stderr, `skills-stack: .agents/skills non è ignorato da Git.

Aggiungi questa riga a .gitignore e riprova:

    /.agents/skills
`)
		return ExitFailure
	}

	tx := &initTransaction{
		sourceSkills: ws.LivePath(),
		destSkills:   ws.StackSkillsDir(name),
		sourceLock:   ws.LockPath(),
		destLock:     ws.StackLockPath(name),
	}

	if stacksState == pathAbsent {
		if err := os.Mkdir(ws.StacksDir(), 0o755); err != nil {
			return initFailure(stderr, name, fmt.Errorf("creare skill-stacks: %w", err), tx)
		}
		tx.createdDirs = append(tx.createdDirs, ws.StacksDir())
	}

	if err := os.Mkdir(ws.StackDir(name), 0o755); err != nil {
		return initFailure(stderr, name, fmt.Errorf("creare skill-stacks/%s: %w", name, err), tx)
	}
	tx.createdDirs = append(tx.createdDirs, ws.StackDir(name))

	stackAgentsDir := filepath.Join(ws.StackDir(name), ".agents")
	if err := os.Mkdir(stackAgentsDir, 0o755); err != nil {
		return initFailure(stderr, name, fmt.Errorf("creare skill-stacks/%s/.agents: %w", name, err), tx)
	}
	tx.createdDirs = append(tx.createdDirs, stackAgentsDir)

	if err := os.Rename(tx.sourceSkills, tx.destSkills); err != nil {
		return initFailure(stderr, name, fmt.Errorf("spostare .agents/skills in skill-stacks/%s/.agents/skills: %w", name, err), tx)
	}
	tx.movedSkills = true

	if lockState == pathRegularFile {
		if err := os.Rename(tx.sourceLock, tx.destLock); err != nil {
			return initFailure(stderr, name, fmt.Errorf("spostare skills-lock.json in skill-stacks/%s/skills-lock.json: %w", name, err), tx)
		}
		tx.movedLock = true
	}

	if err := replace(ws.LivePath(), live.Target(name)); err != nil {
		return initFailure(stderr, name, fmt.Errorf("creare il link .agents/skills: %w", err), tx)
	}

	fmt.Fprintf(stdout, "Stack attivo: %s\n", name)
	fmt.Fprintf(stdout, "Skill importate in skill-stacks/%s/.agents/skills\n", name)
	if tx.movedLock {
		fmt.Fprintf(stdout, "Lock file importato in skill-stacks/%s/skills-lock.json\n", name)
	}
	fmt.Fprintln(stdout, "Se una sessione dell'agente è già aperta, riavviala per caricare le skill.")
	return ExitOK
}

// pathState classifies a path without following symlinks. Only a real
// directory and a regular file are distinguished from everything else.
type pathState int

const (
	pathAbsent pathState = iota
	pathRealDir
	pathRegularFile
	pathOther
)

func inspectPath(path string) (pathState, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return pathAbsent, nil
	}
	if err != nil {
		return pathOther, fmt.Errorf("ispezionare %s: %w", path, err)
	}
	switch {
	case info.IsDir():
		return pathRealDir, nil
	case info.Mode().IsRegular():
		return pathRegularFile, nil
	default:
		return pathOther, nil
	}
}

// initTransaction records what init changed, so a failure can be undone in
// reverse order. Only empty directories created by init are removed.
type initTransaction struct {
	sourceSkills string
	destSkills   string
	sourceLock   string
	destLock     string
	movedSkills  bool
	movedLock    bool
	createdDirs  []string
}

func (tx *initTransaction) rollback() error {
	var errs []error
	if tx.movedLock {
		if err := os.Rename(tx.destLock, tx.sourceLock); err != nil {
			errs = append(errs, fmt.Errorf("ripristinare %s: %w", tx.sourceLock, err))
		}
	}
	if tx.movedSkills {
		if err := os.Rename(tx.destSkills, tx.sourceSkills); err != nil {
			errs = append(errs, fmt.Errorf("ripristinare %s: %w", tx.sourceSkills, err))
		}
	}
	// If a restore failed, its directory is not empty: stop here instead of
	// trying to remove directories that still hold the user's data.
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	for i := len(tx.createdDirs) - 1; i >= 0; i-- {
		if err := os.Remove(tx.createdDirs[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("rimuovere la directory creata %s: %w", tx.createdDirs[i], err))
		}
	}
	return errors.Join(errs...)
}

func initFailure(stderr io.Writer, name string, cause error, tx *initTransaction) int {
	fmt.Fprintf(stderr, "skills-stack: %v\n", cause)
	if err := tx.rollback(); err != nil {
		fmt.Fprintf(stderr, "skills-stack: rollback incompleto: %v\n", err)
		fmt.Fprintf(stderr, "skills-stack: controlla manualmente .agents/skills, skills-lock.json e skill-stacks/%s\n", name)
	}
	return ExitFailure
}
