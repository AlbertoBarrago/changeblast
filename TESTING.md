# ChangeBlast — Manual Test Guide

Guida per una sessione di test manuale dal vivo, con `blast` installato via Homebrew.

## 0. Setup

```bash
brew update
brew upgrade changeblast   # o: brew install changeblast se non ancora installato
blast version
blast doctor
```

**Nota importante:** la release pubblicata su Homebrew è **v0.1.0**. Le rifiniture dell'ultima sessione (colori, `blast version` con commit/build date, fix dell'alias `blast <path>` con `--json`/`--fail-on`, fix performance di `diff`) sono su `main` ma **non ancora rilasciate come tag**. Se vuoi testarle dal binario brew, serve prima un nuovo tag (es. `v0.1.1`) e attendere che la release GitHub Actions completi — altrimenti costruisci in locale con `make build` e usa `./blast` invece del binario brew.

## 1. Repository di prova

Due repository reali clonati per test su scala diversa (nello scratchpad, non nel repo):

| Repo | File JS/TS | Note |
|---|---|---|
| `sindresorhus/got` | ~85 | libreria HTTP piccola/media, tsconfig presente |
| `date-fns/date-fns` | ~1600 | libreria grande, buon test di scala |

```bash
cd <scratchpad>/bench-got     # o bench-datefns
blast doctor
```

## 2. Funzionalità core — checklist

- [ ] `blast inspect <path>` su un file con dipendenti diretti/indiretti noti
- [ ] `blast inspect <path> --json` — valida con `jq .` che il JSON sia corretto
- [ ] `blast <path>` (alias) — deve dare lo stesso output di `blast inspect <path>`
- [ ] `blast <path> --json` (alias con flag) — **nota sopra**: funziona solo con build da `main`, non con brew 0.1.0
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

## 3. Colori e terminale (solo build da `main`, non brew 0.1.0)

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

## 5. Homebrew — verifica pacchetto

```bash
brew info changeblast
brew test changeblast     # esegue il test della formula (blast version)
brew audit --strict changeblast   # opzionale, verifica qualità formula
```

## 6. Cosa segnalare

Per ogni problema trovato, annota: comando esatto eseguito, output ottenuto, output atteso, versione (`blast version`), e se riproducibile sia su `got` che `date-fns` o solo su uno dei due.
