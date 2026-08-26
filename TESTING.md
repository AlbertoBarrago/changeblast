# ChangeBlast: Manual Test Guide

Guida per una sessione di test manuale dal vivo, con `blast` installato via Homebrew.

## 0. Setup

```bash
brew update
brew upgrade changeblast   # o: brew install changeblast se non ancora installato
blast version
blast doctor
```

**Nota importante:** verifica sempre `blast version` prima di iniziare. Le feature più recenti (directory analysis, `--output`, supporto Go) richiedono un tag rilasciato dopo la loro implementazione: se `blast version` mostra una versione precedente, aggiorna con `brew upgrade changeblast` o attendi la release corrispondente.

## 1. Repository di prova

Repository reali per test su scala diversa e linguaggi diversi (nello scratchpad, non nel repo):

| Repo | Linguaggio | File | Note |
|---|---|---|---|
| `sindresorhus/got` | JS/TS | ~85 | libreria HTTP piccola/media, tsconfig presente |
| `date-fns/date-fns` | JS/TS | ~1600 | libreria grande, buon test di scala |
| ChangeBlast stesso (`/Users/albz/Projects/Blast`) | Go | ~40 | dogfooding diretto, go.mod già presente |

```bash
cd <scratchpad>/bench-got     # o bench-datefns
blast doctor
```

Per il test Go, basta restare nella cartella del progetto stesso:
```bash
cd /Users/albz/Projects/Blast
blast inspect internal/graph/graph.go
```

## 2. Funzionalità core: checklist

- [ ] `blast inspect <path>` su un file con dipendenti diretti/indiretti noti
- [ ] `blast inspect <path> --json`: valida con `jq .` che il JSON sia corretto
- [ ] `blast <path>` (alias): deve dare lo stesso output di `blast inspect <path>`
- [ ] `blast <path> --json` (alias con flag) → deve funzionare esattamente come `blast inspect <path> --json`
- [ ] `blast diff` (senza modifiche) → "No JS/TS module changes found"
- [ ] Modifica un file, `blast diff` → mostra impatto del file modificato
- [ ] `blast diff HEAD~1` (o un ref più vecchio) → mostra impatto di un range più ampio
- [ ] `blast graph <path>` → dipendenze dirette + dipendenti diretti
- [ ] `blast history <path>` → churn e co-change coerenti con `git log`
- [ ] `blast doctor` in una directory non-git → deve fallire con messaggio chiaro
- [ ] `blast inspect <path inesistente>` → errore chiaro, exit code 1
- [ ] `blast inspect <path> --fail-on high` su un file a basso rischio → exit 0
- [ ] `blast inspect <path> --fail-on low` → dovrebbe quasi sempre uscire con exit code 2

Verifica exit code con:
```bash
blast inspect <path> --fail-on low; echo "exit: $?"
```

### Directory analysis (`blast inspect <dir>`)

- [ ] `blast inspect .` sulla root del progetto → riepilogo ordinato per rischio decrescente, con conteggio finale HIGH/MEDIUM/LOW
- [ ] `blast inspect <sottocartella>` → solo i file dentro quella cartella, non l'intero repo
- [ ] `blast inspect . --json` → array JSON, stessa forma di `blast diff --json`
- [ ] `blast inspect . --fail-on high` → gating sul file col rischio peggiore trovato
- [ ] `blast inspect` (senza argomenti) → equivalente a `blast inspect .`
- [ ] `blast history` (senza argomenti) → equivalente a `blast history .`, non deve dare errore

### Spiegazione AI (`--explain`, richiede Ollama locale)

- [ ] `ollama serve` attivo e un modello scaricato (`ollama pull llama3.2` o simile)
- [ ] `blast inspect <file> --explain` → dopo il report deterministico, sezione "Explanation (ollama)" con testo coerente ai segnali reali (non generico)
- [ ] `blast inspect <file> --explain --explain-model <altro-modello>` → usa il modello specificato
- [ ] `blast inspect <file> --explain --json` → JSON con forma `{"analysis": {...}, "explanation": "..."}` invece della forma piatta
- [ ] `blast inspect <file> --json` (senza `--explain`) → deve restare nella forma piatta di sempre (nessun campo `analysis`)
- [ ] Ollama spento, `blast inspect <file> --explain` → il report deterministico appare comunque, con "unavailable: ..." al posto della spiegazione, exit code invariato
- [ ] `blast doctor` → riga "Ollama" con stato reachable/not reachable coerente con se `ollama serve` è attivo

### Output su file (`--output`/`-o`)

- [ ] `blast inspect <path> --output report.txt` → il file contiene lo stesso output che andrebbe su stdout, senza codici colore ANSI anche se lanciato da un terminale
- [ ] `blast inspect <path> --output report.json --json` → JSON valido su file
- [ ] `blast diff --output diff-report.txt`, `blast graph <path> -o graph.txt`, `blast history <path> -o hist.txt` → stesso comportamento

### Supporto Go

- [ ] `blast inspect internal/graph/graph.go` (dal repo di ChangeBlast) → mostra correttamente i file che importano quel package
- [ ] `blast doctor` in un repo Go senza `go.mod` → import Go non risolti (nessun errore, solo nessuna dipendenza risolta)
- [ ] Import della standard library (`fmt`, `os`, ecc.) → non generano edge nel grafo (correttamente trattati come esterni)
- [ ] Import di un modulo esterno (es. `github.com/spf13/cobra`) → non traversato, trattato come esterno

## 3. Colori e terminale

```bash
blast inspect <path>                          # colori se il terminale è una TTY
blast inspect <path> | cat                    # nessun colore (output in pipe)
NO_COLOR=1 blast inspect <path>               # nessun colore anche in TTY
```

## 4. Benchmark di performance

Misura i tempi su entrambi i repository:

```bash
cd <scratchpad>/bench-got
time blast inspect source/index.ts
time blast diff HEAD~5

cd <scratchpad>/bench-datefns
time blast inspect src/index.ts
time blast diff HEAD~5
```

Cosa osservare:
- Tempo di scansione totale (dominato da `filepath.WalkDir` + regex per import su ogni file)
- Se `blast diff` con più file modificati resta vicino al tempo di un singolo scan (fix recente: uno scan solo, riusato per ogni file)
- Eventuali rallentamenti anomali su repository con `node_modules` non escluso correttamente

Per un confronto più preciso, ripeti 3 volte e prendi il tempo migliore (il primo run scalda la cache del filesystem):

```bash
for i in 1 2 3; do time blast inspect src/index.ts; done
```

## 5. Homebrew: verifica pacchetto

```bash
brew info changeblast
brew test changeblast     # esegue il test della formula (blast version)
brew audit --strict changeblast   # opzionale, verifica qualità formula
```

## 6. Cosa segnalare

Per ogni problema trovato, annota: comando esatto eseguito, output ottenuto, output atteso, versione (`blast version`), e se riproducibile sia su `got` che `date-fns` o solo su uno dei due.
