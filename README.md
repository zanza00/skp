# skp

`skp` è una CLI per selezionare e attivare uno stack di Agent Skills
all'interno di un repository Git.

> **Stato attuale:** l'MVP è implementato. `list`, `current`, `init`, `use` e
> `status` funzionano da un work tree Git, usano un symlink relativo e
> sostituiscono il link live con una singola `rename`. `init` importa
> un'installazione esistente senza copiarla. TUI, completion, supporto Windows
> e gestione delle skill restano fuori dal perimetro.

## Il problema

Un repository può richiedere insiemi diversi di skill, per esempio uno per il
lavoro e uno per progetti personali. Gestire questi insiemi copiando directory
o ricreando manualmente i collegamenti rende poco chiaro quale sia attivo e
può introdurre differenze tra i file versionati e quelli effettivamente usati.

L'obiettivo è conservare più stack nello stesso repository e cambiare quello
attivo senza copiare, sincronizzare o modificare le skill.

## L'idea Git-native

Ogni stack vive in una directory separata, con le proprie skill e il proprio
lock file, e può essere versionato insieme al repository. Nella root resta un
solo percorso live, `.agents/skills`, realizzato come symlink relativo verso lo
stack attivo.

```text
<repository>/
├── .agents/
│   └── skills -> ../skill-stacks/<stack-attivo>/.agents/skills
├── skill-stacks/
│   ├── work/
│   │   ├── .agents/skills/
│   │   └── skills-lock.json
│   └── personal/
│       ├── .agents/skills/
│       └── skills-lock.json
└── .gitignore
```

Il target del symlink è anche la sorgente dello stato: non è previsto un file
di configurazione o un secondo registro che ricordi lo stack corrente. Il link
live deve essere ignorato da Git, per esempio con `/.agents/skills` in
`.gitignore`; gli stack e i rispettivi lock file restano invece committabili.

## Installazione

Il binario richiede Git ed è disponibile per Linux e macOS su AMD64 e ARM64.

Con Homebrew:

```sh
brew install zanza00/tap/skp
```

Con Go 1.24 o successivo:

```sh
go install github.com/zanza00/skp/cmd/skp@latest
```

### Build da sorgente

I task di build usano [Task](https://taskfile.dev/); in alternativa basta
`go build`.

```sh
task build     # bin/skp
task install   # ~/.local/bin/skp
```

`task install` compila e copia il binario in `~/.local/bin`, che deve essere
nel `PATH`. Senza Task:

```sh
go build -o bin/skp ./cmd/skp
```

Il binario usa solo la standard library e i comandi Git già presenti sul
sistema: non ci sono dipendenze Go esterne né un package manager interno.

## Uso rapido

Primo utilizzo, con uno stack chiamato `work`.

Se le skill sono già installate in `.agents/skills`, importale in uno stack
gestito:

```sh
# se l'esclusione non è già presente
printf '/.agents/skills\n' >> .gitignore

skp init work
skp current
```

`init` sposta `.agents/skills` e l'eventuale `skills-lock.json` in
`skill-stacks/work/`, poi crea il link live. Non copia file, non crea stack
vuoti e non modifica Git: le modifiche vanno riviste e committate manualmente.

Per creare invece lo stack manualmente:

```sh
printf '/.agents/skills\n' >> .gitignore
mkdir -p skill-stacks/work
cd skill-stacks/work
npx skills add <riferimento-skill>   # scegliendo un target che scrive in .agents/skills
cd ../..
skp use work
skp status
```

Le skill vanno sempre gestite dalla directory dello stack, mai attraverso il
link live. Dopo uno switch, una sessione dell'agente già aperta può richiedere
un riavvio.

## Gestione delle skill e switching

`npx skills` resta responsabile di aggiungere, aggiornare e rimuovere le skill.
Questi comandi vanno eseguiti direttamente nella directory dello stack:

```sh
cd skill-stacks/work
npx skills add <riferimento-skill>
npx skills update ...
npx skills remove ...
```

Ogni stack conserva così i propri file e il proprio lock file, anche quando non
è attivo. `skp` non fa da wrapper a `npx` e non è un package manager:
si occupa soltanto di scoprire gli stack, mostrare lo stato e cambiare la
selezione.

## Comandi

```text
skp
skp list
skp current
skp init <stack>
skp use <stack>
skp status
skp --version
```

| Comando | Comportamento |
| --- | --- |
| `skp` | Stampa l'uso su stderr ed esce con `2`. |
| `list` | Elenca gli stack, indicando quello attivo e quelli non selezionabili. |
| `current` | Stampa solo il nome dello stack attivo, adatto agli script. |
| `init <stack>` | Importa `.agents/skills` e l'eventuale lock in un nuovo stack, poi lo attiva. |
| `use <stack>` | Attiva uno stack valido sostituendo soltanto il link live. |
| `status` | Diagnostica link, target, stack corrente ed esclusione Git, senza riparare nulla. |

`-h` e `--help` stampano l'aiuto su stdout ed escono con `0`. I comandi non
interattivi non chiedono conferme e non leggono da stdin. `--version` stampa il
commit e la data del commit incorporati al momento della build, quindi esce con
`0`.

### Esempio di `list`

```text
! _trial (nome non ammesso)
! broken (.agents/skills mancante)
  personal
* work
```

`*` indica lo stack attivo, `!` uno stack non selezionabile con il motivo.

### Exit code

| Codice | Significato |
| --- | --- |
| `0` | Operazione completata. |
| `1` | Errore operativo o stato non sicuro: stack assente o non valido, conflitto sul link, target inatteso, errore Git o filesystem. |
| `2` | Errore d'uso: comando o argomenti non validi; include l'invocazione senza argomenti. |
| `3` | Solo per `current`: nessuno stack attivo perché il link live non esiste. |

`current` è l'interfaccia consigliata per gli script: in caso di successo
stampa soltanto il nome dello stack seguito da newline. Un link presente ma
rotto o inatteso non vale come «nessuno stack attivo»: `current` restituisce
`1`, perché è uno stato da diagnosticare.

## Cosa implementa l'MVP

- stack contenuti direttamente in `skill-stacks/`, con nomi e directory delle
  skill validati prima dello switch;
- importazione di un'installazione esistente con `init`, senza copiare file,
  creare stack vuoti o sovrascrivere stack già presenti;
- symlink relativi, stato derivato dal target e sostituzione atomica tramite
  `rename`;
- rifiuto di directory reali, file e symlink assoluti o inattesi sul percorso
  live, senza pulizie automatiche o cancellazioni ricorsive;
- verifica che il link live sia ignorato da Git prima di crearlo o sostituirlo;
- comandi non interattivi, utilizzabili anche da una sottodirectory del work
  tree;
- supporto previsto per Linux e macOS.

Restano fuori dall'MVP installazione e aggiornamento delle skill, risoluzione
delle dipendenze, validazione del loro contenuto, copia o sincronizzazione,
creazione di stack vuoti, composizione di più stack, stack predefiniti, file di
configurazione, gestione simultanea di più agenti, supporto Windows, TUI e
completion. La CLI non crea commit, non modifica la cronologia Git e non cambia
automaticamente `.gitignore`.

## Sviluppo

```sh
task build        # bin/skp
task install      # ~/.local/bin/skp
go test -race ./...
go vet ./...
```

I test di integrazione creano repository Git temporanei e verificano anche i
casi di sicurezza con file sentinella; non eseguono `npx skills`.

## Roadmap

Dopo l'MVP, la proposta prevede il consolidamento di diagnostiche, completion
shell, una TUI per la selezione ed eventuali esigenze di concorrenza.
Portabilità, integrazioni opzionali e composizione degli stack saranno
valutate separatamente; in particolare, un'eventuale modalità a copia non sarà
un fallback implicito del symlink.

## Specifica completa

La [specifica iniziale](notes/README.md) descrive il contratto dei comandi, gli
exit code, i casi limite, la strategia di testing, le fasi successive e le
decisioni ancora aperte.

## Licenza

`skp` è distribuito con [licenza MIT](LICENSE).
