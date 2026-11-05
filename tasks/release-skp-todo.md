# Todo: rinomina e pubblicazione di `skp`

Piano e decisioni: [`tasks/release-skp-plan.md`](release-skp-plan.md).

## Task 1 — Confermare identità, disponibilità e licenza

**Descrizione:** trasformare il nome scelto in un contratto di pubblicazione
verificato, senza modificare ancora codice o remote. Rivedere la disponibilità il
giorno dell'implementazione e scegliere la licenza prima di creare contenuti
pubblici.

**Criteri di accettazione:**

- [x] Sono confermati `skp` per progetto/binario/formula,
  `github.com/zanza00/skp` per il modulo e `v0.1.0` per la prima release.
- [x] `zanza00/skp` è creabile, `zanza00/homebrew-tap` è quello verificato e non
  esiste un conflitto `skp` in Homebrew Core o nel tap personale.
- [x] La licenza è scelta esplicitamente; la proposta MIT è confermata o
  sostituita prima dei task successivi.
- [x] Si decide consapevolmente di riusare la formula e il publisher con test del
  tap esistente, non la configurazione GoReleaser `brews` deprecata.

**Verifica:**

- [x] Controllare repository, formula e cask con GitHub/Homebrew API e registrare
  nel task eventuali differenze rispetto alla fotografia del piano.
- [x] Rileggere README del tap e il riferimento `source-fetch` per confermare
  convenzioni, secret e piattaforme.

**Dipendenze:** nessuna.

**File probabili:** nessun file applicativo; aggiornare soltanto questo checklist
se emergono decisioni diverse.

**Dimensione:** XS, una decisione operativa con controlli esterni.

### Checkpoint umano

- [x] Nome, owner GitHub, licenza e comando di installazione sono approvati prima
  di iniziare la rinomina.

## Task 2 — Rinominare modulo Go e import

**Descrizione:** sostituire in modo meccanico il modulo precedente con
`github.com/zanza00/skp` nel modulo e in tutti gli import, lasciando
temporaneamente invariati entry point e nome stampato dal binario. Non modificare
alcuna logica.

**Criteri di accettazione:**

- [x] `go list -m` restituisce `github.com/zanza00/skp`.
- [x] Nessun import o riferimento di compilazione usa il vecchio modulo.
- [x] Il codice compila e tutti i test esistenti passano senza cambiamenti ai
  comportamenti attesi, tranne i soli riferimenti al modulo quando necessari.

**Verifica:**

- [x] `go test -race ./...`
- [x] `go vet ./...`
- [x] `go build ./cmd/skills-stack`
- [x] Nessun riferimento al modulo precedente in `go.mod`, `cmd` e `internal`.

**Dipendenze:** Task 1.

**File probabili:**

- `go.mod`
- `cmd/skills-stack/main.go`
- file `internal/**/*.go` che contengono import del modulo.

**Dimensione:** S come sforzo, refactor meccanico atomico su più file; non
spezzare modulo e import in stati non compilabili.

## Task 3 — Rinominare binario, entry point e diagnostiche

**Descrizione:** spostare `cmd/skills-stack/` in `cmd/skp/`, cambiare il nome del
binario nei task di build e sostituire il nome mostrato in aiuto, errori,
`--version` e prefissi temporanei. Aggiornare soltanto le attese dei test legate
al nome.

**Criteri di accettazione:**

- [x] `task build` produce `bin/skp` e `task install` installa `skp`.
- [x] `skp --help` e la prima riga di `skp --version` riportano `skp`.
- [x] Sintassi, output (a parte il nome), exit code, symlink temporanei e
  comportamenti di `list`, `current`, `init`, `use` e `status` restano invariati.
- [x] Non restano riferimenti attivi a `skills-stack` o `.skills-stack-tmp-` nel
  codice e nei test; il layout `skill-stacks/` non viene rinominato.

**Verifica:**

- [x] `go test -race ./...`
- [x] `go vet ./...`
- [x] `task build` e smoke test di `bin/skp --help`, `bin/skp --version` e
  `bin/skp current` in un repository temporaneo.
- [x] `rg 'skills-stack|\.skills-stack-tmp-' cmd internal Taskfile.yml` senza
  risultati.

**Dipendenze:** Task 2.

**File probabili:**

- `Taskfile.yml`
- `cmd/skp/main.go` (rinominato)
- `internal/cli/{cli.go,commands.go,init.go,version.go}` e relativi test
- `internal/live/live.go` e `internal/live/live_test.go` per il solo prefisso
  temporaneo.

**Dimensione:** M, refactor meccanico coordinato; nessuna nuova funzionalità.

## Task 4 — Allineare documenti e licenza

**Descrizione:** aggiornare documentazione pubblica, specifica e istruzioni del
progetto al nome definitivo e ai nuovi percorsi di build/installazione. Aggiungere
la licenza scelta e documentare la pubblicazione prevista senza anticipare
funzionalità non ancora rilasciate.

**Criteri di accettazione:**

- [x] `README.md`, `notes/README.md` e `AGENTS.md` usano `skp`, `cmd/skp/` e il
  modulo GitHub; la decisione sul nome provvisorio è chiusa.
- [x] `LICENSE` contiene la licenza scelta con anno e titolare corretti; README e
  formula useranno lo stesso identificativo SPDX.
- [x] Gli esempi conservano invariati `skill-stacks/`, `.agents/skills`, comandi,
  exit code e limiti dell'MVP.
- [x] Le istruzioni mostrano `brew install zanza00/tap/skp` e
  `go install github.com/zanza00/skp/cmd/skp@latest`; la loro esecuzione resta un
  criterio del Task 8, non una verifica anticipata.

**Verifica:**

- [x] Controllo incrociato tra README, specifica, `Taskfile.yml` e comportamento del
  binario.
- [x] `go test -race ./...`, `go vet ./...` e `task build`.
- [x] Ricerca del vecchio nome escludendo `tasks/` e note di migrazione intenzionali.

**Dipendenze:** Task 3.

**File probabili:**

- `README.md`
- `notes/README.md`
- `AGENTS.md`
- `LICENSE`

**Dimensione:** S, 4 file di documentazione/metadati.

### Checkpoint rinomina

- [x] Suite completa e build verdi; nessun comportamento funzionale modificato.
- [x] Review del diff per accertare che le uniche variazioni siano rinomina,
  documentazione e licenza.

## Task 5 — Aggiungere CI GitHub Actions

**Descrizione:** creare un workflow di controlli riutilizzabile sia su push e pull
request sia dal workflow di release. Coprire le piattaforme supportate e la
versione minima Go senza aggiungere servizi o dipendenze al binario.

**Criteri di accettazione:**

- [x] Linux, macOS ARM64 e macOS Intel eseguono `go test -race ./...`,
  `go vet ./...` e `go build ./cmd/skp` usando la versione Go dichiarata in
  `go.mod`.
- [x] Il workflow supporta `push`, `pull_request` e `workflow_call`, con permessi
  `contents: read`, timeout e azioni fissate tramite SHA.
- [x] La release può richiamare lo stesso workflow, evitando una seconda definizione
  dei controlli.

**Verifica:**

- [x] Validare YAML e workflow localmente con gli strumenti disponibili.
- [x] Dopo il push del Task 6, tutti i job GitHub Actions risultano verdi sulle tre
  piattaforme.

**Dipendenze:** Task 4; l'esecuzione remota dipende dal Task 6.

**File probabili:**

- `.github/workflows/checks.yml`

**Dimensione:** S, un workflow.

## Task 6 — Migrare il repository su GitHub

**Descrizione:** creare `zanza00/skp` pubblico senza inizializzarlo con una storia
indipendente, pubblicare la cronologia esistente e rendere GitHub il remote
primario.

**Criteri di accettazione:**

- [x] GitHub contiene la cronologia di `main`, il file di licenza, la CI e la
  documentazione rinominata; non vengono pubblicati `bin/`, `dist/` o artefatti
  locali.
- [x] `origin` punta a `git@github.com:zanza00/skp.git` e il branch locale segue
  `origin/main`.
- [x] Non sono eseguiti force push, rewrite o commit automatici dalla CLI.
- [x] Descrizione, visibilità, branch predefinito e homepage del repository sono
  coerenti; i check del Task 5 sono verdi.

**Verifica:**

- [x] `git remote -v`, `git status --short --branch` e `git ls-remote origin main`.
- [x] `gh repo view zanza00/skp` e verifica dei check tramite `gh run list` /
  `gh run watch`.
- [x] Clone fresco in una directory temporanea, seguito da `go test -race ./...` e
  `go build ./cmd/skp`.

**Dipendenze:** Task 4 e definizione locale del Task 5.

**File probabili:** nessuna modifica ai sorgenti; impostazioni Git/GitHub e
esecuzione del workflow `.github/workflows/checks.yml`.

**Dimensione:** S, migrazione operativa con clone di verifica.

### Checkpoint GitHub

- [x] Repository pubblico leggibile e clone fresco funzionante.
- [x] CI verde prima di configurare credenziali o automazioni di release.

## Task 7 — Configurare release e Homebrew tap

**Descrizione:** aggiungere GoReleaser per i quattro target, adattare il publisher
Python con test al nome `skp`, creare il workflow di tag e preparare nel tap la
formula, la documentazione e i test di installazione. Nessuna pubblicazione reale
avviene in questo task.

**Criteri di accettazione:**

- [ ] GoReleaser costruisce `skp` per Linux/macOS `amd64` e `arm64`, produce
  archivi `.tar.gz`, `checksums.txt` e conserva i metadati VCS di `--version`.
- [ ] Il renderer produce `Formula/skp.rb` con URL, versione, quattro SHA-256,
  licenza e dipendenza `git`; rifiuta tag non stabili, checksum mancanti,
  duplicati o non validi.
- [ ] Il workflow accetta solo tag `vX.Y.Z` appartenenti a `main`, esegue i check,
  richiede `HOMEBREW_TAP_TOKEN`, pubblica con permessi minimi e verifica gli
  artefatti. `GITHUB_TOKEN` non viene usato per scrivere nel tap.
- [ ] `dist/` è ignorato; azioni fissate tramite SHA e nessun segreto compare in
  configurazione o output.
- [ ] Nel tap, README e workflow includono `skp` senza cambiare il contenuto delle
  altre formule; i test coprono Linux, macOS ARM64 e macOS Intel.

**Verifica:**

- [ ] `python3 -m unittest scripts/publish_homebrew_test.py`
- [ ] `goreleaser check`
- [ ] `goreleaser release --snapshot --clean` con working tree pulito.
- [ ] Estrarre gli archivi snapshot, verificare i checksum ed eseguire
  `skp --help` e `skp --version` sul target disponibile.
- [ ] Generare la formula con il publisher in `--dry-run`, ispezionarla e validarne
  sintassi/audit senza caricarla nel tap.
- [ ] Validare i workflow e rivedere il diff del tap prima del primo tag.

**Dipendenze:** Task 6.

**File probabili in `skp`:**

- `.goreleaser.yaml`
- `.github/workflows/release.yml`
- `scripts/publish-homebrew.py`
- `scripts/publish_homebrew_test.py`
- `.gitignore`

**File probabili in `homebrew-tap`:**

- `Formula/skp.rb` (generato alla prima release; il template è verificato qui)
- `README.md`
- `.github/workflows/test.yml`

**Dimensione:** M, 5 file nel progetto più due documenti/configurazioni nel tap;
la formula non viene scritta a mano né pubblicata in questa fase.

### Checkpoint pre-release

- [ ] Snapshot e test del publisher completamente verdi.
- [ ] Secret del tap configurato con scope minimo e workflow rivisto.
- [ ] Approvazione umana esplicita prima di creare e pubblicare il primo tag.

## Task 8 — Pubblicare e validare `v0.1.0`

**Descrizione:** creare il tag stabile dal commit approvato su `main`, avviare il
workflow e verificare il percorso completo dalla release all'installazione reale.
Correggere gli errori in fix forward, senza riutilizzare tag o artefatti già
pubblici.

**Criteri di accettazione:**

- [ ] Il tag `v0.1.0` punta al commit approvato e la release non è draft né
  prerelease.
- [ ] Sono presenti quattro archivi e `checksums.txt`; i checksum scaricati da
  GitHub Releases coincidono con quelli della formula.
- [ ] `Formula/skp.rb` viene aggiornato da un commit di automazione nel tap e i
  test del tap passano senza regressioni sulle altre formule.
- [ ] Installazioni pulite su Linux, macOS ARM64 e macOS Intel eseguono
  correttamente `brew install zanza00/tap/skp`,
  `brew test zanza00/tap/skp`, `skp --help` e `skp --version`.
- [ ] `go install github.com/zanza00/skp/cmd/skp@v0.1.0` funziona da una directory
  temporanea e il binario installato risponde correttamente.
- [ ] README, release e tap riportano gli stessi comandi verificati; il checklist e
  eventuali note di rilascio vengono chiusi con i riferimenti alla release.

**Verifica:**

- [ ] `gh release view v0.1.0` e `gh run watch` per release e tap.
- [ ] Download degli artefatti e controllo SHA-256 su ogni runner prima di
  `brew install`.
- [ ] Smoke test funzionale in un repository Git temporaneo: `skp list`,
  `skp current`, `skp status` e `skp use` mantengono il contratto.
- [ ] `git status --short --branch` pulito nel repository principale e assenza di
  modifiche non previste nel tap.

**Dipendenze:** Task 7 e approvazione del tag.

**File probabili:**

- `tasks/release-skp-todo.md` per la chiusura e i riferimenti finali.
- Solo eventuali correzioni documentali emerse dalla verifica, in commit separati.

**Dimensione:** S, operazione di release con verifica su tre runner.

## Checkpoint finale

- [ ] `https://github.com/zanza00/skp` e la release `v0.1.0` sono pubblici.
- [ ] Homebrew e `go install` forniscono lo stesso binario funzionante.
- [ ] Tutti i criteri precedenti sono completi; nessun elemento fuori perimetro è
  stato aggiunto.
