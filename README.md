# 🕵️ reposnusern

**reposnusern** er et verktøy for å analysere GitHub-repositorier i en organisasjon – med nysgjerrighet, struktur og en dæsj AI.

## 🎯 Ambisjon

Målet med dette prosjektet er å lage et fleksibelt og utvidbart analyseverktøy for utviklingsmiljøer som ønsker innsikt i kodebasen sin. Prosjektet utvikles stegvis:

### 1. Datainnhenting

- Henter metadata, språkbruk, Dockerfiles og dependency-filer fra alle repoer i en GitHub-organisasjon.
- Data lagres i en relasjonsdatabase (SQLite i PoC).
- Bruker JSON-filer som mellomlagring for å redusere GitHub API-bruk.
- Kjøres periodisk (f.eks. via cron-jobb).

### 2. Analyseverktøy

- Kjører regelbaserte analyser av:
  - Dockerfiles (best practices og sikkerhet)
  - Dependency-filer (rammeverk og versjonsbruk per språk)
  - Språkstatistikk
- Resultater lagres i databasen for effektiv spørring og videre bruk.

### 3. Tilgjengeliggjøring av data

- Tilbyr en enkel API for å hente ut data og analyseresultater.
- Tanken er at dataene kan brukes av:
  - Andre Go-programmer
  - Jupyter-notebooks
  - Visualiseringsverktøy som Power BI, Metabase eller Grafana

### Teknologier og oppsett

- 🧠 Språk: Go
- 🗃️ Database: SQLite (sqlc brukt for typesikker tilgang)
- 📦 Strukturelt monorepo – men med tydelig inndeling

## 🧪 PoC-status

Proof-of-Concept bruker følgende:
- `go + sqlc + sqlite3`
- JSON-filer med:
  - Repo-metadata
  - Språkstatistikk
  - Øverste nivå `Dockerfile`-innhold

Dette gir et godt grunnlag for å bygge videre analyser, inkludert rammeverksdeteksjon basert på språk og filstruktur.

## 📁 Prosjektstruktur
```
repo-analyzer/
│
├── cmd/
│   ├── fetch/               # Henter og lagrer nye data fra GitHub
│   ├── analyze/             # Kjører ulike analyser
│   └── api/                 # Starter opp en enkel API-server
│
├── internal/
│   ├── fetcher/             # GitHub API-klient + JSON-mellomlagring
│   ├── analyzer/            # Analyse av Dockerfiles og dependencies
│   ├── storage/             # sqlc + generell datatilgang
│   ├── models/              # Delte datastrukturer
│   └── config/              # Konfigurasjonshåndtering
│
├── migrations/              # databaseoppsett og migreringer
├── schema.sql               # SQLite-skjema
├── sqlc.yaml                # sqlc-konfigurasjon
├── go.mod / go.sum
└── data/                    # Midlertidig JSON-lagring
```

## 🤖 Erklæring om bruk av generativ KI

Under utviklingen av dette innholdet har forfatter(e) benyttet generativ KI – inkludert M365 Copilot og ChatGPT – til å omformulere og effektivisere tekst og kode. Alt innhold er deretter gjennomgått og redigert manuelt. 
