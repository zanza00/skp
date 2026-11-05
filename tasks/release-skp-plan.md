# Piano di rinomina e pubblicazione: `skp`

## Obiettivo

Rinominare il progetto, il comando e il modulo Go in `skp`, pubblicare il
repository su GitHub e distribuire la prima release `v0.1.0` tramite GitHub
Releases e il tap Homebrew personale. Il lavoro non deve cambiare il contratto
dei comandi, il layout `skill-stacks/`, il link `.agents/skills` né le regole di
sicurezza dell'MVP.

La checklist operativa con criteri di accettazione e dipendenze è in
[`tasks/release-skp-todo.md`](release-skp-todo.md). I piani MVP già presenti in
`tasks/` restano invariati come artefatti storici.

## Situazione verificata

Al 5 novembre 2026:

- non esiste ancora una configurazione GitHub Actions;
- l'account GitHub autenticato è `zanza00` e `zanza00/skp` non esiste;
- `zanza00/homebrew-tap` esiste già, è pubblico e contiene altre formule;
- `skp` non risulta presente in Homebrew Core tramite le API ufficiali di
  formule e cask; il controllo va ripetuto prima del rilascio;
- il progetto non contiene ancora un file di licenza;
- `source-fetch` usa già GoReleaser, il secret `HOMEBREW_TAP_TOKEN` e un piccolo
  publisher Python con test per aggiornare il medesimo tap: questa infrastruttura
  è il riferimento da riusare, non una dipendenza del nuovo progetto.

## Decisioni architetturali

- **Nome unico:** progetto, repository, binario e formula si chiamano `skp`.
- **Modulo Go:** `github.com/zanza00/skp`; l'entry point diventa `cmd/skp/`.
- **GitHub primario:** `https://github.com/zanza00/skp`, pubblico, con `main`
  come branch predefinito. Dopo la migrazione, GitHub è `origin`, senza
  riscrivere la cronologia.
- **Homebrew:** riusare `zanza00/homebrew-tap` e pubblicare
  `Formula/skp.rb`. L'installazione prevista è:

  ```sh
  brew install zanza00/tap/skp
  ```

  Non usare la sezione `brews` di GoReleaser, deprecata da v2.10: adattare invece
  il publisher con test già adottato da `source-fetch`, mantenendo la formula per
  binari precompilati e la dipendenza runtime da `git`.
- **Release:** Semantic Versioning, prima release stabile `v0.1.0`, archivi per
  Linux e macOS su `amd64` e `arm64`, checksum SHA-256 e note generate da
  GoReleaser. Windows resta fuori dal perimetro.
- **CI:** GitHub Actions su Linux, macOS ARM64 e macOS Intel; test con la versione
  minima Go dichiarata dal modulo e controlli di build, `vet` e `-race`.
- **Segreti:** `GITHUB_TOKEN` limitato al repository di release e
  `HOMEBREW_TAP_TOKEN` con accesso Contents read/write soltanto al tap. Nessun
  token nei file o nei log; azioni fissate tramite SHA e permessi minimi.
- **Nessuna nuova dipendenza runtime:** il binario continua a usare solo Go
  standard library e Git. Python e `gh` sono strumenti della sola automazione di
  pubblicazione, come già avviene per `source-fetch`.
- **Licenza:** MIT, confermata dal proprietario prima della pubblicazione.

## Ordine e dipendenze

```text
1. Contratto di pubblicazione e licenza
   └── 2. Rinomina modulo e import
       └── 3. Rinomina binario e messaggi
           └── 4. Documenti pubblici e licenza
               └── 5. CI GitHub Actions
                   └── 6. Migrazione del repository su GitHub
                       └── 7. Release automation e Homebrew tap
                           └── 8. Prima release e verifica utente
```

I Task 2 e 3 sono separati solo per rendere riconoscibili le due sostituzioni
meccaniche; il modulo e tutti gli import di ciascuna slice devono sempre restare
coerenti e compilabili. Documentazione e configurazione del tap possono essere
preparate in parallelo dopo il Task 3, ma la pubblicazione effettiva resta
sequenziale.

## Fasi e checkpoint

### Fase 1 — Identità e rinomina

- [ ] Task 1: confermare contratto di pubblicazione, disponibilità e licenza.
- [ ] Task 2: rinominare modulo Go e import.
- [ ] Task 3: rinominare entry point, binario, diagnostiche e test.
- [ ] Task 4: allineare README, specifica, istruzioni e licenza.

**Checkpoint — nessuna regressione funzionale**

- [ ] `go test -race ./...`, `go vet ./...` e `task build` sono verdi.
- [ ] `task build` produce `bin/skp`; aiuto e `--version` riportano `skp`.
- [ ] I test esistenti dimostrano che comandi, exit code, symlink e sicurezza non
  sono cambiati.
- [ ] Il vecchio nome resta solo nei documenti storici o nelle note di migrazione
  esplicitamente escluse dalla ricerca.

### Fase 2 — Presenza GitHub e qualità automatica

- [ ] Task 5: aggiungere CI Linux/macOS e controlli riutilizzabili dalla release.
- [ ] Task 6: pubblicare la cronologia su GitHub e impostare i remote.

**Checkpoint — repository pubblico verificato**

- [ ] `zanza00/skp` mostra codice, licenza e documentazione corretti.
- [ ] La CI obbligatoria è verde su tutte le piattaforme previste.
- [ ] `origin` punta a GitHub e non è stato eseguito alcun force push.

### Fase 3 — Release e installazione

- [ ] Task 7: configurare GoReleaser, publisher del tap, workflow e test.
- [ ] Task 8: pubblicare `v0.1.0` e validare il percorso utente completo.

**Checkpoint — pubblicazione completata**

- [ ] La release contiene i quattro archivi e `checksums.txt`, con checksum
  verificati.
- [ ] Il tap aggiorna `Formula/skp.rb`, README e test di installazione senza
  modificare le altre formule.
- [ ] Su Linux, macOS ARM64 e macOS Intel, `brew install zanza00/tap/skp`,
  `brew test zanza00/tap/skp`, `skp --help` e `skp --version` funzionano.
- [ ] `go install github.com/zanza00/skp/cmd/skp@v0.1.0` produce un binario
  funzionante.
- [ ] README e release notes riportano istruzioni effettivamente verificate.

## Verifica trasversale

Ogni slice esegue i test indicati nella checklist. Prima del tag si esegue anche
una release snapshot senza pubblicazione, si ispezionano archivi e formula
generata e si verifica che il binario compilato conservi commit e data in
`--version`. Dopo il tag non si sposta né si riusa la stessa versione: eventuali
errori si correggono con un nuovo commit e, se gli artefatti pubblici sono già
stati distribuiti, con una release patch successiva.

## Rischi e mitigazioni

| Rischio | Impatto | Mitigazione |
|---|---|---|
| Il nome `skp` diventa non disponibile prima della pubblicazione | Alto | Ripetere i controlli nel Task 1; non migrare il modulo finché il contratto non è confermato |
| La rinomina introduce modifiche funzionali accidentali | Alto | Slice meccaniche separate, nessuna modifica alla logica, suite completa con `-race` e confronto dei comandi |
| Il token del tap ha permessi eccessivi o finisce nei log | Alto | Secret dedicato, accesso al solo tap, nessun valore nei file, permessi Actions minimi e verifica manuale del workflow |
| Gli artefatti GoReleaser perdono i metadati di `--version` | Medio | Smoke test sul binario snapshot e su ogni pacchetto installato; bloccare la release se commit o data sono inattesi |
| La formula usa URL o checksum incoerenti | Alto | Test unitari del renderer, snapshot, verifica SHA-256 e installazione reale sulle tre piattaforme |
| Il workflow del tap rompe le formule esistenti | Medio | Modifica isolata a `skp`, esecuzione dei test del tap e controllo dei diff prima della release |
| Un tag pubblicato contiene un errore | Alto | Validazione di tag e ancestry, CI prima di publish, release immutabile e fix forward con SemVer |

Non sono previste altre estensioni funzionali. TUI, Cobra/Viper, completion,
composizione degli stack, Windows e pubblicazione in Homebrew Core restano fuori
da questo piano.
