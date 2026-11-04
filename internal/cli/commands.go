package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"codefloe.com/zanza00/skills-stack/internal/live"
	"codefloe.com/zanza00/skills-stack/internal/stack"
	"codefloe.com/zanza00/skills-stack/internal/workspace"
)

func runList(ws *workspace.Workspace, stdout, stderr io.Writer) int {
	entries, err := stack.Discover(ws.StacksDir())
	if err != nil {
		return fail(stderr, "%v", err)
	}

	active, err := detectActive(ws)
	if err != nil {
		return fail(stderr, "%v", err)
	}

	if len(entries) == 0 {
		fmt.Fprintln(stdout, "Nessuno stack trovato.")
		return ExitOK
	}

	for _, entry := range entries {
		marker := " "
		if !entry.Selectable {
			marker = "!"
		}
		if active.Kind == live.Managed && active.Stack == entry.Name {
			marker = "*"
		}

		line := entry.Name
		if !entry.Selectable {
			line = fmt.Sprintf("%s (%s)", entry.Name, entry.Reason)
		}
		fmt.Fprintf(stdout, "%s %s\n", marker, line)
	}
	return ExitOK
}

// detectActive interprets the live link only when .agents is a real directory.
// The active marker is informational, so an unsafe or missing .agents results
// in no active stack rather than an error.
func detectActive(ws *workspace.Workspace) (live.State, error) {
	state, err := ws.AgentsDirState()
	if err != nil {
		return live.State{}, err
	}
	if state != workspace.DirReal {
		return live.State{Kind: live.Absent}, nil
	}
	return live.Inspect(ws.LivePath())
}

func runCurrent(ws *workspace.Workspace, stdout, stderr io.Writer) int {
	agentsState, err := ws.AgentsDirState()
	if err != nil {
		return fail(stderr, "%v", err)
	}
	if agentsState == workspace.DirOther {
		return fail(stderr, ".agents esiste ma non è una directory reale: stato non sicuro")
	}
	if agentsState == workspace.DirAbsent {
		return noActiveStack(stderr)
	}

	state, err := live.Inspect(ws.LivePath())
	if err != nil {
		return fail(stderr, "%v", err)
	}

	switch state.Kind {
	case live.Absent:
		return noActiveStack(stderr)
	case live.Managed:
		if err := stack.Validate(ws.StacksDir(), state.Stack); err != nil {
			return fail(stderr, "il link live punta a %q ma non è valido: %v", state.RawTarget, err)
		}
		fmt.Fprintln(stdout, state.Stack)
		return ExitOK
	case live.UnexpectedSymlink:
		return fail(stderr, ".agents/skills punta a un target non gestito: %s", state.RawTarget)
	case live.Directory:
		return fail(stderr, ".agents/skills è una directory reale, non un symlink")
	case live.File:
		return fail(stderr, ".agents/skills è un file regolare, non un symlink")
	default:
		return fail(stderr, ".agents/skills non è un symlink gestibile")
	}
}

func noActiveStack(stderr io.Writer) int {
	fmt.Fprintln(stderr, "Nessuno stack attivo: .agents/skills non esiste.")
	return ExitNoStack
}

func runStatus(ws *workspace.Workspace, stdout, stderr io.Writer) int {
	agentsState, err := ws.AgentsDirState()
	if err != nil {
		return fail(stderr, "%v", err)
	}

	ignored := false
	ignoreKnown := agentsState != workspace.DirOther
	if ignoreKnown {
		ignored, err = ws.IgnoresLiveLink()
		if err != nil {
			return fail(stderr, "%v", err)
		}
	}

	state := live.State{Kind: live.Absent}
	if agentsState == workspace.DirReal {
		state, err = live.Inspect(ws.LivePath())
		if err != nil {
			return fail(stderr, "%v", err)
		}
	}

	code := ExitOK
	if agentsState == workspace.DirOther {
		code = ExitFailure
	}

	fmt.Fprintf(stdout, "Repository: %s\n", ws.Root)
	fmt.Fprintf(stdout, "Directory .agents: %s\n", agentsLabel(agentsState))
	liveLabel := kindLabel(state.Kind)
	if agentsState == workspace.DirOther {
		liveLabel = "n/d"
	}
	fmt.Fprintf(stdout, "Percorso live: %s\n", liveLabel)

	switch state.Kind {
	case live.Managed:
		fmt.Fprintf(stdout, "Stack corrente: %s\n", state.Stack)
		fmt.Fprintf(stdout, "Target: %s\n", state.RawTarget)
		fmt.Fprintf(stdout, "Risolto: %s\n", live.Resolve(ws.LivePath(), state.RawTarget))
		if err := stack.Validate(ws.StacksDir(), state.Stack); err != nil {
			fmt.Fprintf(stdout, "Destinazione: non valida (%s)\n", err)
			code = ExitFailure
		} else {
			fmt.Fprintln(stdout, "Destinazione: valida")
		}
	case live.UnexpectedSymlink:
		fmt.Fprintln(stdout, "Stack corrente: nessuno")
		fmt.Fprintf(stdout, "Target: %s\n", state.RawTarget)
		fmt.Fprintf(stdout, "Risolto: %s\n", live.Resolve(ws.LivePath(), state.RawTarget))
		fmt.Fprintln(stdout, "Destinazione: non gestita")
		code = ExitFailure
	case live.Directory, live.File, live.Other:
		fmt.Fprintln(stdout, "Stack corrente: nessuno")
		fmt.Fprintln(stdout, "Destinazione: n/d")
		code = ExitFailure
	default:
		fmt.Fprintln(stdout, "Stack corrente: nessuno")
		fmt.Fprintln(stdout, "Destinazione: n/d")
	}

	if !ignoreKnown {
		fmt.Fprintln(stdout, "Esclusione Git: non verificabile")
		code = ExitFailure
	} else if ignored {
		fmt.Fprintln(stdout, "Esclusione Git: presente")
	} else {
		fmt.Fprintln(stdout, "Esclusione Git: assente")
		code = ExitFailure
	}
	return code
}

func agentsLabel(state workspace.DirState) string {
	switch state {
	case workspace.DirReal:
		return "directory reale"
	case workspace.DirAbsent:
		return "assente"
	default:
		return "non è una directory reale"
	}
}

func kindLabel(kind live.Kind) string {
	switch kind {
	case live.Managed:
		return "symlink"
	case live.UnexpectedSymlink:
		return "symlink non gestito"
	case live.Directory:
		return "directory"
	case live.File:
		return "file"
	case live.Other:
		return "altro"
	default:
		return "assente"
	}
}

func runUse(ws *workspace.Workspace, name string, stdout, stderr io.Writer) int {
	if err := stack.Validate(ws.StacksDir(), name); err != nil {
		return fail(stderr, "%v", err)
	}

	agentsState, err := ws.AgentsDirState()
	if err != nil {
		return fail(stderr, "%v", err)
	}
	if agentsState == workspace.DirOther {
		return fail(stderr, ".agents esiste ma non è una directory reale: intervento manuale richiesto")
	}

	state, err := live.Inspect(ws.LivePath())
	if err != nil {
		return fail(stderr, "%v", err)
	}

	target := live.Target(name)
	switch state.Kind {
	case live.Absent, live.Managed:
		// The only switchable states.
	case live.UnexpectedSymlink:
		return fail(stderr, ".agents/skills punta a un target non gestito (%s): intervento manuale richiesto", state.RawTarget)
	case live.Directory:
		return fail(stderr, ".agents/skills è una directory reale: non verrà sostituita, spostala o rimuovila manualmente")
	case live.File:
		return fail(stderr, ".agents/skills è un file regolare: non verrà sostituito, spostalo o rimuovilo manualmente")
	default:
		return fail(stderr, ".agents/skills non è un symlink gestibile: intervento manuale richiesto")
	}

	ignored, err := ws.IgnoresLiveLink()
	if err != nil {
		return fail(stderr, "%v", err)
	}
	if !ignored {
		fmt.Fprint(stderr, `skills-stack: .agents/skills non è ignorato da Git.

Aggiungi questa riga a .gitignore e riprova:

    /.agents/skills

Se il link è già tracciato da Git, rimuovilo dall'indice:

    git rm --cached .agents/skills
`)
		return ExitFailure
	}

	if state.Kind == live.Managed && state.RawTarget == target {
		fmt.Fprintf(stdout, "Stack attivo: %s\n", name)
		return ExitOK
	}

	if agentsState == workspace.DirAbsent {
		if err := os.Mkdir(ws.AgentsDir(), 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return fail(stderr, "creare .agents: %v", err)
		}
		if created, err := ws.AgentsDirState(); err != nil || created != workspace.DirReal {
			return fail(stderr, "creare .agents: la directory non è utilizzabile")
		}
	}

	if err := live.Replace(ws.LivePath(), target); err != nil {
		return fail(stderr, "sostituire .agents/skills: %v", err)
	}

	fmt.Fprintf(stdout, "Stack attivo: %s\n", name)
	fmt.Fprintln(stdout, "Se una sessione dell'agente è già aperta, riavviala per caricare le skill.")
	return ExitOK
}
