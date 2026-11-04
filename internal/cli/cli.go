package cli

import (
	"fmt"
	"io"

	"codefloe.com/zanza00/skills-stack/internal/workspace"
)

// Exit codes defined by the command contract.
const (
	ExitOK      = 0
	ExitFailure = 1
	ExitUsage   = 2
	ExitNoStack = 3
)

// Run executes one command and returns its exit code. dir is the starting
// directory used to locate the Git work tree; an empty dir means the current
// directory.
func Run(dir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "-h", "--help":
		if len(args) != 1 {
			return usageError(stderr, "%s non accetta argomenti", args[0])
		}
		printHelp(stdout)
		return ExitOK
	case "list":
		if len(args) != 1 {
			return usageError(stderr, "list non accetta argomenti")
		}
		ws, code := openWorkspace(dir, stderr)
		if ws == nil {
			return code
		}
		return runList(ws, stdout, stderr)
	case "current":
		if len(args) != 1 {
			return usageError(stderr, "current non accetta argomenti")
		}
		ws, code := openWorkspace(dir, stderr)
		if ws == nil {
			return code
		}
		return runCurrent(ws, stdout, stderr)
	case "status":
		if len(args) != 1 {
			return usageError(stderr, "status non accetta argomenti")
		}
		ws, code := openWorkspace(dir, stderr)
		if ws == nil {
			return code
		}
		return runStatus(ws, stdout, stderr)
	case "use":
		if len(args) != 2 {
			return usageError(stderr, "uso: skills-stack use <stack>")
		}
		ws, code := openWorkspace(dir, stderr)
		if ws == nil {
			return code
		}
		return runUse(ws, args[1], stdout, stderr)
	default:
		return usageError(stderr, "comando sconosciuto %q", args[0])
	}
}

func openWorkspace(dir string, stderr io.Writer) (*workspace.Workspace, int) {
	ws, err := workspace.Open(dir)
	if err != nil {
		return nil, fail(stderr, "%v", err)
	}
	return ws, ExitOK
}

func fail(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "skills-stack: "+format+"\n", args...)
	return ExitFailure
}

func usageError(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "skills-stack: "+format+"\n\n", args...)
	printUsage(stderr)
	return ExitUsage
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `Uso:
  skills-stack list
  skills-stack current
  skills-stack use <stack>
  skills-stack status

Esegui "skills-stack -h" per l'aiuto completo.
`)
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `skills-stack gestisce lo stack di Agent Skills attivo in un repository Git.

Uso:
  skills-stack list
  skills-stack current
  skills-stack use <stack>
  skills-stack status
  skills-stack -h | --help

Comandi:
  list       elenca gli stack e indica quello attivo
  current    stampa il nome dello stack attivo
  use        attiva uno stack sostituendo il link .agents/skills
  status     mostra una diagnostica senza modificare nulla

Exit code:
  0  operazione completata
  1  errore operativo o stato non sicuro
  2  errore d'uso
  3  nessuno stack attivo (solo current)
`)
}
