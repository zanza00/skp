# skp: specifica iniziale

Documento di lavoro per una CLI Go che seleziona uno stack di Agent Skills e lo rende attivo in un repository. Il progetto e il comando usano il nome `skp`; il modulo Go è `github.com/zanza00/skp`.

Le sezioni marcate **MVP** descrivono il primo perimetro implementabile. Le estensioni sono raccolte separatamente e non sono requisiti dell'MVP.

## 1. Obiettivo e non-obiettivi

### Obiettivo MVP

Permettere a chi lavora in un repository di:

- mantenere più stack di skill versionati in directory separate;
- vedere quali stack sono disponibili e quale è attivo;
- convertire un'installazione esistente in uno stack gestito, spostando `.agents/skills` e l'eventuale lock file senza copiarne il contenuto;
- attivare uno stack tramite un symlink relativo `.agents/skills`;
- cambiare stack in modo atomico, senza modificare il contenuto degli stack;
- eseguire le operazioni da comandi non interattivi;
- rilevare stati ambigui o non sicuri prima di sostituire il link attivo.

La CLI gestisce esclusivamente la selezione dello stack. Le skill e i relativi lock file restano gestiti dagli strumenti già esistenti.

### Non-obiettivi MVP

L'MVP non deve:

- installare, aggiornare o rimuovere skill;
- risolvere dipendenze o reimplementare un package manager;
- fare da wrapper ai comandi `npx skills add/update/remove`;
- validare il contenuto delle skill, il formato del lock file o la compatibilità con un agente;
- copiare o sincronizzare skill tra directory;
- creare stack vuoti o inizializzare uno stack senza un'installazione esistente;
- comporre più stack, gestire layer o skill comuni;
- gestire contemporaneamente target di agenti diversi;
- creare commit, modificare la cronologia Git o cambiare automaticamente `.gitignore`;
- riavviare automaticamente l'agente;
- supportare Windows;
- fornire una TUI interattiva nell'MVP (rinviata alla Fase 2);
- introdurre un file di configurazione o uno stack predefinito.

## 2. Terminologia

- **Repository**: il work tree Git in cui viene usata la CLI. La root è quella restituita da Git, non necessariamente la directory corrente.
- **Stack**: un insieme di skill contenuto in `skill-stacks/<nome-stack>/`, con il proprio lock file.
- **Directory dello stack**: `skill-stacks/<nome-stack>/`. È anche la directory dalla quale si eseguono i comandi `npx skills` relativi a quello stack.
- **Directory delle skill dello stack**: `skill-stacks/<nome-stack>/.agents/skills/`.
- **Link live**: `.agents/skills` nella root del repository. Punta alla directory delle skill dello stack attivo.
- **Stack attivo** o **corrente**: lo stack ricavato dal target del link live. Non esiste un secondo file che memorizza questa selezione.
- **Stack valido per lo switch**: nel MVP, uno stack il cui nome è ammesso e la cui directory `.agents/skills` è una directory reale nel percorso previsto. La CLI non interpreta il lock file.
- **Package manager delle skill**: i comandi esterni `npx skills add/update/remove`, eseguiti direttamente dall'utente.

## 3. Layout e stato

### Layout MVP

```text
<repository>/
├── .agents/
│   └── skills -> ../skill-stacks/<stack-attivo>/.agents/skills
├── skill-stacks/
│   ├── work/
│   │   ├── .agents/
│   │   │   └── skills/
│   │   └── skills-lock.json
│   └── personal/
│       ├── .agents/
│       │   └── skills/
│       └── skills-lock.json
└── .gitignore
```

Il target del link è relativo alla directory che contiene il link, cioè `.agents/`. Per uno stack chiamato `work`, il valore del symlink è:

```text
../skill-stacks/work/.agents/skills
```

Regole del layout:

- `.agents/skills` nella root è l'unico percorso gestito dalla CLI.
- Il link live deve essere ignorato da Git. Nel caso più semplice, `.gitignore` contiene `/.agents/skills`; sono accettate anche le altre esclusioni Git valide.
- Gli stack, comprese le skill e i file `skills-lock.json`, sono contenuti nel repository e possono essere committati.
- Ogni stack ha un lock file separato nella propria directory. La CLI dell'MVP non lo legge per determinare lo stato e non ne valida il contenuto.
- `.agents/` può contenere altri file non gestiti dalla CLI, che non devono essere modificati.
- Vengono considerate solo le directory immediatamente contenute in `skill-stacks/`; non c'è ricerca ricorsiva di altri stack.
- I nomi stack dell'MVP usano lettere ASCII, cifre, `.`, `_` e `-`, con primo carattere alfanumerico. Separatori di percorso, percorsi assoluti, `.` e `..` non sono nomi validi.
- Senza link live non esiste uno stack attivo e non viene scelto alcun default implicito.

La root del repository viene individuata tramite Git. I comandi possono quindi essere lanciati anche da una sottodirectory del work tree. Fuori da un work tree Git i comandi falliscono senza modificare file.

## 4. Flussi operativi

### 4.1 Primo utilizzo tramite importazione

Se le skill sono già installate in `.agents/skills` e devono diventare uno
stack gestito, dalla root del repository (o da una sua sottodirectory):

1. Se manca, aggiungere l'esclusione del link a `.gitignore`:

   ```gitignore
   /.agents/skills
   ```

2. Importare l'installazione in uno stack:

   ```sh
   skp init work
   ```

3. Verificare il risultato:

   ```sh
   skp current
   skp status
   ```

`init` sposta `.agents/skills` in `skill-stacks/work/.agents/skills`, sposta
l'eventuale `skills-lock.json` della root in
`skill-stacks/work/skills-lock.json` e crea il link live. Non copia file, non
crea stack vuoti, non modifica `.gitignore` e non tocca l'indice o la cronologia
Git. Le skill e il lock file già tracciati da Git risultano spostati: le
modifiche vanno riviste e committate manualmente.

`init` non è idempotente: se lo stack esiste già il comando fallisce. Su un
repository già inizializzato si usa `use` per cambiare stack.

### 4.2 Primo utilizzo manuale

Esempio con uno stack chiamato `work`:

1. Dalla root del repository, aggiungere manualmente il link live a `.gitignore`:

   ```gitignore
   /.agents/skills
   ```

2. Creare la directory dello stack e entrarvi:

   ```sh
   mkdir -p skill-stacks/work
   cd skill-stacks/work
   ```

3. Aggiungere le skill con lo strumento esistente:

   ```sh
   npx skills add <riferimento-skill>
   ```

   Durante il comando va scelto un target agente che scriva in `.agents/skills`. Le opzioni effettive dipendono da `npx skills` e non fanno parte del contratto di questa CLI. L'operazione deve lasciare skill e `skills-lock.json` dentro `skill-stacks/work/`.

4. Tornare nel repository e attivare lo stack:

   ```sh
   skp use work
   ```

5. Verificare lo stato, se necessario:

   ```sh
   skp status
   ```

6. Riavviare la sessione dell'agente se era già aperta e potrebbe avere caricato le skill precedenti.

In questo flusso manuale la CLI non crea lo stack e non prepara automaticamente `.gitignore`: i passaggi restano espliciti. In alternativa, `init` importa un'installazione già presente (sezione 4.1).

### 4.3 Switch tra stack esistenti

```sh
skp list
skp current
skp use personal
skp current
```

`use` cambia soltanto il link live. Non copia file, non modifica lock file e non esegue `npx`. Dopo lo switch può essere necessario riavviare la sessione dell'agente.

### 4.4 Aggiunta, aggiornamento o rimozione di skill

Per operare su `work`:

```sh
cd skill-stacks/work
npx skills add <riferimento-skill>
# oppure: npx skills update ...
# oppure: npx skills remove ...
```

Regole del flusso:

- il comando `npx` viene eseguito nella directory dello stack, non nella root del repository e non attraverso il link live;
- il target agente selezionato deve essere compatibile con `.agents/skills`;
- skill e lock file vengono aggiornati direttamente nello stack, anche quando questo non è attivo;
- se lo stack modificato è attivo, il link live continua a puntare agli stessi file; può essere necessario riavviare l'agente;
- le modifiche vengono controllate e committate normalmente con Git, senza intervento della nuova CLI.

## 5. Contratto dei comandi MVP

### Convenzioni generali

```text
skp
skp list
skp current
skp init <stack>
skp use <stack>
skp status
skp --version
```

- I comandi non interattivi non chiedono conferme e non leggono la scelta da stdin.
- I risultati destinati a script vanno su stdout; gli errori vanno su stderr.
- `current` è il comando consigliato per gli script: in caso di successo stampa soltanto il nome dello stack seguito da newline.
- Gli output di `list` e `status` sono destinati alle persone e possono essere migliorati senza cambiare il modello di stato.
- Un errore non deve lasciare un link temporaneo né modificare uno stack.

### Exit code

| Codice | Significato |
| --- | --- |
| `0` | Operazione completata. |
| `1` | Errore operativo o stato non sicuro: stack assente/non valido, conflitto sul link, target inatteso, errore Git o filesystem. |
| `2` | Errore d'uso: comando o argomenti non validi; include l'invocazione senza argomenti. |
| `3` | Solo per `current`: nessuno stack è attivo perché il link live non esiste. |

Un link presente ma rotto o inatteso non equivale a «nessuno stack attivo»: `current` restituisce `1`, perché è uno stato da diagnosticare.

### 5.1 Nessun argomento: aiuto

```sh
skp
```

Senza argomenti il comando stampa l'uso su stderr ed esce con codice `2`; non
sceglie alcuno stack. `-h` e `--help` stampano l'aiuto su stdout ed escono con
codice `0`. `--version` stampa su stdout il nome del binario, il commit Git e la
data del commit registrati al momento della build, quindi esce con `0`. I
metadati provengono dalle informazioni VCS incorporate dal toolchain Go; se il
binario è stato compilato fuori da un work tree Git, commit e data risultano
`sconosciuto` e `sconosciuta`. `--version` non accetta altri argomenti.

La TUI interattiva è fuori dall'MVP: la progettazione è conservata nella
sezione 6 per la Fase 2.

### 5.2 `list`

```sh
skp list
```

Elenca le directory immediatamente sotto `skill-stacks/`, ordinate per nome. Per ciascuna mostra:

- se è attiva;
- se è selezionabile;
- se non è selezionabile, una motivazione sintetica, per esempio directory `.agents/skills` mancante o nome non ammesso.

Una possibile rappresentazione è:

```text
* work
  personal
! broken (.agents/skills mancante)
```

Il comando non modifica il link. Un repository senza stack produce un elenco vuoto o il messaggio `Nessuno stack trovato.` ed esce con `0`. Un errore di lettura del repository produce invece exit code `1`.

### 5.3 `current`

```sh
skp current
```

- Se il link live è un symlink relativo riconosciuto e punta a uno stack valido, stampa solo `<nome-stack>` ed esce con `0`.
- Se il link live non esiste, non stampa un nome su stdout, segnala l'assenza su stderr ed esce con `3`.
- Se il percorso live è una directory reale, un file, un symlink rotto o un symlink con target non riconosciuto, stampa una diagnostica su stderr ed esce con `1`.

Il comando non deduce lo stato dal lock file, dalla directory corrente, da una configurazione o dall'ultimo comando eseguito.

### 5.4 `init <stack>`

```sh
skp init work
```

Converte un'installazione esistente in uno stack gestito.

Prima di spostare qualsiasi file, il comando deve:

1. verificare che il nome sia ammesso e che `.agents` sia una directory reale;
2. verificare che `.agents/skills` sia una directory reale, non un symlink, un file o un altro tipo di nodo;
3. verificare che `skill-stacks` sia assente o una directory reale e che `skill-stacks/<stack>` non esista in nessuna forma;
4. accettare il `skills-lock.json` della root solo se assente o file regolare;
5. verificare che una regola Git escluda `.agents/skills` anche quando il percorso è già tracciato, usando `git check-ignore --no-index`.

Se tutti i controlli passano, il comando crea le directory necessarie, sposta `.agents/skills` e l'eventuale lock file nelle rispettive destinazioni, quindi crea il link live con la stessa sostituzione atomica di `use`.

Il contenuto delle skill e del lock file non viene letto, validato o riscritto. Il comando non copia, non crea stack vuoti, non fa merge con uno stack esistente e non modifica `.gitignore`, l'indice Git o la cronologia. Gli altri file dentro `.agents` restano al loro posto.

In caso di successo stampa `Stack attivo: <stack>`, i percorsi importati e il promemoria sul riavvio dell'agente; esce con `0`.

Se un'operazione fallisce dopo uno spostamento, il comando annulla in ordine inverso il lock e la directory delle skill, quindi rimuove soltanto le directory vuote create da lui. Non esegue mai cancellazioni ricorsive. Se anche il rollback fallisce, riporta i percorsi da controllare manualmente.

Casi particolari:

- `.agents/skills` assente: nulla da importare, exit code `1`;
- `.agents/skills` già symlink: il comando rimanda a `use`, exit code `1`;
- `skill-stacks`, la destinazione o il lock file hanno un tipo non ammesso: exit code `1` prima di ogni spostamento;
- link live non ignorato da Git: exit code `1` prima di ogni spostamento, con l'indicazione della regola da aggiungere;
- argomenti mancanti o aggiuntivi: exit code `2`.

`init` non è idempotente: su un repository già inizializzato fallisce e non simula una seconda importazione. Per cambiare stack si usa `use`.

### 5.5 `use <stack>`

```sh
skp use work
```

Attiva esattamente lo stack indicato.

Prima di modificare il link, il comando deve:

1. verificare che il nome sia ammesso e che esista `skill-stacks/work/.agents/skills` come directory reale;
2. verificare che `.agents/skills` nella root risulti ignorato da Git;
3. verificare che `.agents` sia una directory reale oppure che possa essere creato;
4. verificare che il percorso live sia assente oppure un symlink gestibile secondo le regole della sezione 7.

In caso di successo stampa `Stack attivo: work` ed esce con `0`. Un messaggio ricorda che una sessione dell'agente già aperta può richiedere un riavvio.

Casi particolari:

- se lo stack richiesto è già attivo e il link è valido, il comando è idempotente ed esce con `0`, senza riscrivere il link;
- se lo stack richiesto non esiste o non è valido, il link precedente resta invariato e l'exit code è `1`;
- se il link non è ignorato da Git, il comando fallisce prima di crearlo o sostituirlo e indica l'esclusione da configurare;
- argomenti mancanti o aggiuntivi producono exit code `2`.

### 5.6 `status`

```sh
skp status
```

Mostra una diagnostica leggibile di:

- root del repository;
- stack corrente, se riconosciuto;
- tipo del percorso live: assente, symlink, directory, file o altro;
- target grezzo e target risolto del symlink, quando disponibili;
- esistenza e validità della directory di destinazione;
- presenza dell'esclusione Git per il link live.

`status` non ripara nulla. Esce con `0` sia in uno stato valido con stack attivo sia quando il link è semplicemente assente ma il resto è coerente. Esce con `1` per link rotto, target inatteso, conflitto con una directory/file o mancata esclusione Git.

### Completion

La generazione di completion shell non fa parte dell'MVP: non è necessaria per i flussi principali. Può essere aggiunta in una fase successiva, dopo aver stabilizzato nomi e sintassi dei comandi.

## 6. Comportamento essenziale della TUI (Fase 2)

Questa sezione è fuori dall'MVP e descrive la TUI prevista per la fase
successiva. La TUI deve soltanto selezionare e attivare uno stack:

- mostra gli stessi stack e stati di `list`, evidenziando quello corrente;
- permette di spostarsi con le frecce su/giù;
- `Invio` attiva lo stack selezionato, se valido;
- le voci non valide restano visibili ma non selezionabili e mostrano una motivazione;
- `q` ed `Esc` escono senza modificare nulla;
- non attiva uno stack solo perché è l'unico presente o perché è stato evidenziato;
- non offre operazioni di installazione, aggiornamento o rimozione delle skill;
- in caso di errore durante lo switch, mostra la diagnostica e lascia invariato il link precedente;
- dopo uno switch riuscito, chiude la selezione e comunica il nome dello stack attivo e l'eventuale necessità di riavviare l'agente.

Se non ci sono stack validi, la TUI mostra la situazione senza effettuare azioni automatiche.

## 7. Switch atomico, sicurezza e casi limite

### Algoritmo richiesto

Dopo tutti i controlli preliminari:

1. creare un symlink temporaneo con nome univoco dentro `.agents/`, quindi nello stesso filesystem del link live;
2. impostare come target il percorso relativo previsto per lo stack scelto;
3. sostituire `.agents/skills` con una sola `rename` atomica dal symlink temporaneo;
4. in caso di errore prima della `rename`, eliminare soltanto il proprio symlink temporaneo e conservare il link precedente.

La CLI non deve cancellare ricorsivamente alcun percorso e non deve attraversare il link live per ripulirne il contenuto. Dopo la `rename`, chi accede al percorso vede il vecchio o il nuovo target, non una copia parziale.

### Importazione con `init`

`init` produce lo stesso link finale di `use`, ma prima deve liberare il percorso live, che è una directory reale. L'ordine è:

1. eseguire tutti i controlli preliminari, inclusa l'esclusione Git con `--no-index`;
2. creare `skill-stacks/`, `skill-stacks/<stack>/` e il relativo `.agents/` solo se assenti e con i tipi attesi;
3. spostare `.agents/skills` nella destinazione con una `rename`;
4. spostare l'eventuale `skills-lock.json` della root con una `rename`;
5. creare il link live con lo stesso algoritmo atomico di `use`.

In caso di errore, le operazioni già completate vengono annullate in ordine inverso. Se il ripristino riesce, vengono rimosse con `os.Remove` soltanto le directory vuote create dal comando; se il ripristino fallisce, non viene rimossa alcuna directory che possa contenere i dati originali e la diagnostica indica i percorsi da controllare. Non viene mai usata una cancellazione ricorsiva.

### Classificazione del percorso live

Il tipo del percorso va rilevato senza seguire il symlink.

| Stato di `.agents/skills` | Comportamento di `use` |
| --- | --- |
| Assente | Crea il link dopo i controlli. |
| Symlink relativo verso il percorso atteso di uno stack | Può sostituirlo atomicamente. |
| Symlink gestito ma temporaneamente rotto perché il vecchio stack è stato rimosso | Può sostituirlo con un nuovo stack valido. |
| Symlink assoluto, anche se punta a uno stack esistente | Non lo sostituisce automaticamente; segnala il target inatteso. |
| Symlink relativo verso un percorso non previsto | Non lo sostituisce automaticamente. |
| Symlink rotto con target non riconosciuto | Non lo sostituisce automaticamente. |
| Directory reale, anche vuota o apparentemente generata | Non la rinomina, sovrascrive o cancella; fallisce senza modifiche. |
| File regolare o altro tipo di nodo | Non lo sostituisce; fallisce senza modifiche. |

Il recupero dei casi non gestibili è manuale: l'utente deve verificare provenienza e contenuto del percorso, quindi spostarlo o rimuoverlo consapevolmente. La CLI non deve proporre una pulizia automatica.

### Ulteriori regole e casi limite

- **Directory di destinazione non valida**: lo switch fallisce prima di toccare il link attivo. Una directory `.agents/skills` dello stack che è a sua volta un symlink non è accettata nell'MVP.
- **Link rotto riconosciuto**: `use <altro-stack>` può ripararlo; `use <vecchio-stack>` fallisce se la destinazione richiesta non è valida.
- **`.agents` assente**: `use` può crearlo. Se esiste ma non è una directory reale, oppure è un symlink, il comando fallisce senza seguirlo.
- **Target fuori dal repository**: non è mai generato dalla CLI. Un link live esistente con target esterno o non riconosciuto richiede intervento manuale.
- **Percorsi ambigui**: varianti del target con segmenti `./` o `..` aggiuntivi, doppi separatori o percorsi assoluti non vengono normalizzate per poi essere considerate gestite; viene accettato solo il target relativo previsto.
- **Permessi insufficienti o filesystem senza symlink**: il comando fallisce e conserva il vecchio link. Non esiste fallback automatico alla copia.
- **Errore durante la sostituzione**: il vecchio link resta valido; gli eventuali temporanei creati dalla CLI vengono rimossi individualmente, senza operazioni ricorsive.
- **Stack rimosso dopo lo switch**: il link diventa rotto; `current` e `status` lo segnalano. Nessun altro stack viene attivato automaticamente.
- **Sessione agente aperta**: lo switch non garantisce il ricaricamento delle skill. La necessità di riavvio dipende dall'agente.

## 8. Scelta del symlink e alternativa futura

Per l'MVP il symlink è preferito alla copia perché:

- mantiene una sola sorgente dei file per stack;
- rende lo switch un'operazione sul solo link, senza duplicare le skill;
- permette di ricavare lo stack corrente dal target;
- evita problemi di drift tra una copia live e lo stack committato;
- rende immediatamente visibili nello stack le modifiche effettuate con `npx skills` nella sua directory.

La copia non è un fallback dell'MVP. Una futura modalità a copia dovrà definire separatamente sorgente di verità, rilevamento delle modifiche locali, sincronizzazione, aggiornamento atomico e protezione delle directory preesistenti. Non deve essere introdotta come comportamento implicito quando un symlink fallisce.

## 9. Testing essenziale dell'MVP

I test non devono richiedere l'esecuzione reale di `npx skills`: il package manager è esterno al contratto della CLI.

### Test unitari

- validazione dei nomi stack e rifiuto di path traversal;
- costruzione del target relativo e riconoscimento dei soli link gestiti;
- discovery degli stack validi e non validi;
- mapping tra stati del link, diagnostiche ed exit code;
- distinzione tra `IgnoresLiveLink` e il controllo con `--no-index` usato da `init` per i percorsi tracciati;
- lettura dei metadati VCS e fallback di `--version` quando commit o data non sono disponibili;
- idempotenza di `use` sullo stack già attivo.

### Test di integrazione su filesystem temporaneo

- primo `use`: creazione del symlink relativo corretto e `current` coerente;
- switch da stack A a stack B: cambia solo il link; contenuti e lock file di A e B restano invariati;
- esecuzione da una sottodirectory del repository;
- `list`, `current`, `init`, `use` e `status` nei casi senza stack, senza link, con link valido e con link non valido;
- stack richiesto assente o privo di `.agents/skills`: vecchio link invariato;
- directory reale live con un file sentinella: directory e file restano intatti;
- file regolare live, `.agents` non directory e `.agents` symlink: nessuna modifica automatica;
- symlink assoluto o con target inatteso: rifiuto senza sostituzione;
- symlink gestito rotto: riparazione possibile verso uno stack valido;
- symlink rotto non riconosciuto: rifiuto senza sostituzione;
- path live non ignorato da Git: rifiuto prima dello switch;
- errore di creazione o `rename`: conservazione del vecchio link e assenza di temporanei residui;
- seconda esecuzione di `use` sullo stesso stack: successo senza cambiamenti;
- `init` con directory reale e lock opzionale: spostamento, link corretto e `current`, `list` e `status` coerenti;
- `init` con skill già tracciate da Git ma coperte da una regola `.gitignore`: importazione consentita;
- `init` con sorgente assente, symlink, file, destinazione esistente, `skill-stacks` non directory, lock non regolare o link non ignorato: rifiuto prima degli spostamenti, con sentinelle intatte;
- errore nella creazione del link dopo gli spostamenti: ripristino di skill e lock, rimozione delle sole directory vuote create e assenza di temporanei;
- seconda esecuzione di `init` sullo stesso stack: fallimento senza modifiche.

### Test TUI (Fase 2)

Questi test non fanno parte della suite MVP.

- avvio senza argomenti in un pseudo-terminale;
- navigazione, conferma e annullamento senza switch;
- impossibilità di selezionare una voce non valida;
- errore di switch mostrato senza perdere lo stato precedente;
- invocazione senza TTY: nessuna scelta implicita ed exit code `2`.

La CI dell'MVP deve coprire almeno Linux e macOS. I test di sicurezza devono usare file sentinella per verificare non solo il codice di ritorno, ma anche l'assenza di modifiche indesiderate.

## 10. Roadmap a fasi

### Fase 1 — MVP

- layout a stack singolo e lock file separati;
- importazione di un'installazione esistente con `init`;
- stato rappresentato esclusivamente dal link live;
- `list`, `current`, `use` e `status`;
- symlink relativi e switch atomico;
- protezioni per directory reali, file e symlink inattesi;
- supporto Linux/macOS e test essenziali.

### Fase 2 — Consolidamento

Solo dopo l'uso dell'MVP e senza cambiare il modello di stato:

- miglioramento delle diagnostiche e dei messaggi di recupero;
- completion shell;
- TUI per la selezione dello stack (progettazione nella sezione 6);
- eventuali opzioni di output per script, se `current` e i comandi esistenti non bastano;
- gestione esplicita di eventuali problemi di concorrenza emersi durante l'uso.

### Fase 3 — Portabilità

Da progettare e valutare separatamente:

- supporto Windows, tenendo conto di permessi e differenze tra symlink e junction;
- eventuale modalità a copia come alternativa esplicita, non come fallback silenzioso.

### Fase 4 — Integrazioni opzionali

In base a necessità effettive:

- wrapper trasparenti per `npx skills`, senza reimplementarne la logica;
- target per più agenti;
- validazione opzionale di skill, lock file e compatibilità dei target.

### Fase 5 — Composizione e configurazione

Funzioni che richiedono una revisione del modello di stato e non devono essere anticipate nell'MVP:

- stack base più stack specifico, layering e skill comuni;
- regole di precedenza e gestione dei conflitti;
- configurazione di uno stack default;
- eventuale rappresentazione dello stato quando un singolo symlink non è più sufficiente.

## 11. Decisioni aperte

Le seguenti questioni restano aperte e non bloccano l'MVP descritto:

- **Supporto Windows**: se e quando supportarlo, e con quale meccanismo sul filesystem.
- **Composizione base + stack**: semantiche di layering, precedenza, lock file e rappresentazione dello stato.
- **Configurazione e default**: presenza, formato e posizione di un file di configurazione; eventuale stack predefinito e precedenza tra default e scelta esplicita.

Nell'MVP queste decisioni non vengono risolte implicitamente: nessun default viene scelto, nessuna composizione viene simulata e nessun comportamento Windows viene dichiarato supportato.
