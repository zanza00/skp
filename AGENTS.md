# AGENTS.md

Project instructions for `skills-stack`. Read this before changing code or
documentation.

## What this is

`skills-stack` is a small Go CLI that activates one Agent Skills stack per Git
repository by swapping a single relative symlink. The name is provisional.

**Current state: MVP implemented.** The module is
`codefloe.com/zanza00/skills-stack`; `cmd/skills-stack/` is the entry point and
`internal/` holds the implementation. `list`, `current`, `use` and `status` are
covered by unit and integration tests. The contract lives in
[`notes/README.md`](notes/README.md); [`README.md`](README.md) is the public
summary. If they disagree, `notes/README.md` wins.

## Scope

MVP, in scope:

- stacks under `skill-stacks/<name>/`, each with `.agents/skills/` and
  `skills-lock.json`;
- the live link `.agents/skills` in the repository root, pointing to
  `../skill-stacks/<name>/.agents/skills`;
- `list`, `current`, `use <stack>`, `status`;
- relative symlinks, atomic replacement via `rename`, state derived only from
  the link target;
- Linux and macOS.

Out of scope for the MVP — do not add without an explicit decision:

- **TUI**: deferred to phase 2; no Bubble Tea or terminal UI dependency;
- **Cobra/Viper**: use the standard library (`flag` + dispatch). No config
  file, no default stack, no shell completion, no Viper. The `golang-cli`
  skill suggests Cobra+Viper; that advice does not apply to this MVP;
- installing, updating or removing skills (that stays with `npx skills`);
- stack composition/layering as a CLI feature (the current flattened trial in
  `skill-stacks/go/` is manual), multiple agents, Windows;
- commits, history changes, automatic `.gitignore` edits, automatic repair or
  cleanup;
- reading or validating `skills-lock.json`.

## Non-negotiable safety rules

- Inspect `.agents/skills` with `Lstat`, never by following it.
- Replace it only with one atomic `rename` of a temporary symlink created in
  `.agents/`.
- Never delete recursively, overwrite a real directory or file, or accept
  absolute/unexpected symlink targets.
- Refuse to create or replace the link unless Git ignores `/.agents/skills`.
- On any error, keep the previous link and remove only the temporary symlink.
- Validate stack names before building paths: ASCII letter/digit first, then
  letters, digits, `.`, `_`, `-`; reject path separators, `.` and `..`.

## Layout and skills

```text
.agents/skills -> ../skill-stacks/go/.agents/skills   # live link, gitignored
skill-stacks/go/
├── .agents/skills/                                   # committed skills
└── skills-lock.json                                  # committed lock file
```

Code layout:

```text
cmd/skills-stack/main.go   # entry point, os.Exit only
internal/
├── cli/                   # dispatch, help, exit codes, output
├── workspace/             # Git root, layout paths, check-ignore
├── stack/                 # name validation, discovery, validity
└── live/                  # Lstat, target parsing, atomic replace
```

Installed in `skill-stacks/go/`:

Go layer, from `samber/cc-skills-golang`:

- `golang-testing` — table-driven tests, `t.TempDir`, integration tests,
  `-race`;
- `golang-security` — path traversal, symlink safety, `os.Root` /
  `filepath.IsLocal`, sentinel files;
- `golang-error-handling` — wrapping, `errors.Is`/`As`, error-to-exit-code
  mapping;
- `golang-cli` — stdout/stderr discipline, argument validation, exit codes.
  Ignore its Cobra/Viper/completion advice for the MVP.

Process layer, from `addyosmani/agent-skills`:

- `planning-and-task-breakdown` — spec to ordered, verifiable tasks;
- `incremental-implementation` — thin slices, verify and commit each one;
- `test-driven-development` — red-green-refactor, test pyramid;
- `code-review-and-quality` — five-axis review, severity labels;
- `debugging-and-error-recovery` — reproduce, localize, reduce, fix, guard.

The process layer currently lives in the same stack as the Go layer as a
trial. This is a flattened composition, not the layering feature: the stack
is no longer single-purpose, and its lock file mixes two sources. Proper
layering (independent stacks composed into a live selection) is deferred to
phase 5 in the spec; do not add more cross-purpose skills without an explicit
decision.

Manage skills from the stack directory, never through the live link:

```sh
cd skill-stacks/go
npx skills add <package> -a opencode --copy -y
npx skills update
npx skills remove <skill>
```

Commit `skill-stacks/` and its lock file; never commit the live link.

## Implementation guidelines

- Use the standard library first. Add a dependency only for a concrete need.
- `cmd/skills-stack/` holds the entry point; `main.go` stays minimal and
  `internal/` holds the implementation.
- Exit codes: `0` success, `1` operational/unsafe state, `2` usage error
  (including no arguments), `3` `current` with no live link. `-h`/`--help`
  exits `0`.
- `current` prints only the stack name on stdout; diagnostics go to stderr.
- Tests: table-driven with named subtests; filesystem tests use `t.TempDir()`
  and sentinel files; no real `npx skills`; run `go test -race ./...`; CI must
  cover Linux and macOS.
- Do not create commits, change Git history or edit `.gitignore` automatically
  from the CLI.

## Session workflow

Use two or three skills per session, not all of them. The default order for
this repository:

1. **Plan** — `planning-and-task-breakdown` turns the spec into small,
   verifiable tasks. It may create `tasks/plan.md` and `tasks/todo.md`; treat
   them as working artifacts, not as the spec, and do not commit them unless
   asked.
2. **Implement one slice** — `incremental-implementation` for the smallest
   end-to-end change that leaves the tree working.
3. **Drive it with tests** — `test-driven-development` for the red-green-
   refactor loop; `golang-testing` for the Go conventions (named subtests,
   `t.TempDir`, `-race`).
4. **Harden the risky part** — `golang-security` whenever the change touches
   paths, symlinks, `Lstat`, `rename`, Git discovery or permissions.
5. **Debug** — `debugging-and-error-recovery` when a test or build fails,
   before changing code blindly.
6. **Review** — `code-review-and-quality` before committing a non-trivial
   change.

Rules:

- `golang-cli` applies when shaping commands, flags, output and exit codes, but
  its Cobra/Viper/completion advice is overridden by this file.
- `golang-error-handling` applies when wrapping errors or mapping them to exit
  codes; add only the infrastructure the MVP needs.
- Skip `planning-and-task-breakdown` for a single-file change with obvious
  scope, and skip `incremental-implementation` for a one-line fix.

## Conventions

- Code, comments and commit messages in English. User-facing docs and the
  spec in Italian.
- Conventional Commits: `docs:`, `feat:`, `fix:`, `test:`, `refactor:`.
- Keep `README.md` and `notes/README.md` aligned with behavior and scope.

## Open decisions

- Final project, binary and Go module name (`codefloe.com/zanza00/skills-stack`
  is derived from the remote and easy to change).
- Cobra adoption in phase 2 (completion, richer help).
- TUI reintroduction in phase 2.
- Layering model: independent stacks composed by the CLI vs flattened
  installation; conflict policy and state representation.
- Windows support, copy mode.
