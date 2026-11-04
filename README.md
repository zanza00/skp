# skills-stack

`skills-stack` è il nome provvisorio di una CLI per selezionare e attivare
uno stack di Agent Skills all'interno di un repository Git.

> **Stato attuale:** il repository contiene la proposta e il design della CLI,
> non ancora un binario implementato. I comandi descritti qui sono quindi il
> comportamento previsto, non funzionalità già disponibili.

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

La CLI individuerà la root tramite Git e gestirà esclusivamente il link live.
Lo switch sostituirà il symlink in modo atomico, senza attraversarlo e senza
cambiare il contenuto degli stack.

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
è attivo. `skills-stack` non farà da wrapper a `npx` e non sarà un package
manager: si occuperà soltanto di scoprire gli stack, mostrare lo stato e
cambiare la selezione. Dopo uno switch, una sessione dell'agente già aperta
potrebbe dover essere riavviata per caricare le nuove skill.

## Comandi previsti

```text
skills-stack
skills-stack list
skills-stack current
skills-stack use <stack>
skills-stack status
```

| Comando | Scopo |
| --- | --- |
| `skills-stack` | Stampare l'uso e uscire con codice `2`. |
| `list` | Elencare gli stack, indicando quello attivo e quelli non selezionabili. |
| `current` | Stampare il nome dello stack attivo, con output adatto agli script. |
| `use <stack>` | Attivare uno stack valido sostituendo soltanto il link live. |
| `status` | Diagnosticare link, target, stack corrente ed esclusione Git, senza riparare nulla. |

I comandi non interattivi condividono gli stessi controlli. In assenza di un
link non verrà scelto alcuno stack predefinito. La TUI è rinviata alla Fase 2.

## Perimetro dell'MVP

Il primo perimetro prevede:

- stack contenuti direttamente in `skill-stacks/`, con nomi e directory delle
  skill validati prima dello switch;
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
composizione di più stack, stack predefiniti, file di configurazione, gestione
simultanea di più agenti e supporto Windows. La CLI non creerà commit, non
modificherà la cronologia Git e non cambierà automaticamente `.gitignore`.

## Roadmap

Dopo l'MVP, la proposta prevede il consolidamento di diagnostiche, completion
shell, una TUI per la selezione ed eventuali esigenze di concorrenza.
Portabilità, integrazioni opzionali e composizione degli stack saranno
valutate separatamente; in particolare, un'eventuale modalità a copia non
sarà un fallback implicito del symlink.

## Specifica completa

La [specifica iniziale](notes/README.md) descrive il contratto dei comandi, gli
exit code, i casi limite, la strategia di testing, le fasi successive e le
decisioni ancora aperte, incluso il nome definitivo del progetto.
