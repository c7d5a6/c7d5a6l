# Backend architecture

Go API for ASL Uplink. Prefer small packages, explicit layers, and integration tests with committed fixtures.

## Layers

```
cmd/                 # binaries (server, workers, one-shot tools)
internal/
  handler/           # HTTP adapters (REST) — was “controllers”
  service/           # business use-cases
  repository/        # SQLite persistence
  model/             # domain structs (JSON + DB-facing types)
  liquipedia/        # external Liquipedia client + HTML parse
  middleware/        # auth, request ID, logging, CORS, rate limits
  config/            # env / flags
  job/               # scheduled work (cron-like)
testdata/            # committed fixtures (HTML, SQLite snapshots)
migrations/          # SQLite schema versions
```

### `handler` (HTTP) — not “controller”

Name package **`handler`** (Go convention). One file (or small group) per resource + method style is fine:

| Pattern | Example |
|---|---|
| Resource-oriented | `handler/tournament.go` with `GetTournament`, `PostParse` |
| Method-split files | `handler/tournament_get.go`, `handler/tournament_post.go` only if a resource grows large |

**Handlers own:**

- Decode/validate HTTP input
- Map auth principal from context (set by middleware)
- Call **one** service method
- Map domain errors → status codes
- Encode JSON

**Handlers do not:** open DB transactions, run SQL, call Liquipedia directly (except thin passthrough already behind a service), or embed cron logic.

Auth **checks** that need the request (bearer, cookie) live in **`middleware`**. Handlers may enforce **resource-level** rights (“can this user touch this tournament id?”) using data from the service, or call `service.Authorize…` — keep policy in service when it needs DB.

### `service`

Use-case methods: `ParseTournament`, `ListMatches`, etc.

- Orchestrate repositories + external clients
- **Any write that touches SQLite runs in a transaction** (see Transactions)
- No `net/http` types
- Return domain errors the handler can classify

### `repository` (SQLite)

Name package **`repository`** (or `store` — pick one; this project uses **`repository`**).

- All SQLite reads/writes live here
- Accept `context.Context` and optionally `DBTX` (`*sql.DB` or `*sql.Tx`) so services can pass a transaction
- No business rules beyond integrity helpers
- No HTTP

Alternative names seen elsewhere: `store`, `repo`, `sqlite`. Prefer **`repository`** for clarity.

### `model`

Shared domain types. Keep JSON tags stable for the frontend. Avoid importing handler/service.

### External packages (`liquipedia`, …)

Clients and pure parsers. Parsers stay unit/integration-testable from HTML fixtures without DB.

---

## SQLite

- **One writer** process in mind (WAL mode). Set `busy_timeout`, enable foreign keys.
- Schema changes only via **`migrations/`** (numbered SQL or a migrator). Never “fix ALTER” in random code.
- Paths:
  - **Dev DB:** e.g. `backend/devdata/app.sqlite` (gitignored or a small committed seed — see below)
  - **Test fixtures:** `backend/testdata/sqlite/*.sqlite` committed snapshots
  - **Runtime/prod:** path from config/env, never overwrite testdata

### Dev SQLite

- Committed **seed** optional: `testdata/sqlite/dev_seed.sqlite` or SQL under `devdata/seed.sql`
- Working copy for manual runs: `devdata/app.sqlite` (gitignored), created by `go run ./cmd/devdb` or migrate+seed on server start in dev
- Document how to reset: delete file + migrate + seed

---

## Transactions

Rule: **if a service method writes to the DB (one or many statements), it opens a transaction and rolls back on any error.**

```go
func (s *TournamentService) SaveParsed(ctx context.Context, page model.TournamentPage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after Commit

	if err := s.repo.UpsertTournament(ctx, tx, page); err != nil {
		return err
	}
	// …more repo calls with tx…

	return tx.Commit()
}
```

- Pass `tx` into repository methods (interface `DBTX`)
- Read-only methods may use `*sql.DB` directly
- Prefer **short** transactions; don’t hold a tx across Liquipedia HTTP calls — fetch/parse first, then tx write

---

## Scheduled jobs (“cron”)

### Recommendation

| Approach | When |
|---|---|
| **In-process scheduler** ([`robfig/cron/v3`](https://github.com/robfig/cron)) | Single server binary + SQLite (this project’s default) |
| **OS timer** (systemd `OnCalendar`, launchd) calling `go run ./cmd/job …` | Strongest isolation / ops control |
| **External** (cloud scheduler hitting an admin route) | Multi-instance or managed hosts |

**Most reliable for a solo Go + SQLite app:** one process, `robfig/cron`, jobs registered in `internal/job`, started from `cmd/server` **after** DB migrate, stopped on graceful shutdown.

Reliability rules:

1. **Idempotent** job bodies (safe if run twice)
2. **Job lock** row or `flock` so overlapping ticks don’t double-run
3. **Don’t schedule Liquipedia scrape hotter than their rate limits** (already ~30s for parse)
4. **Graceful shutdown:** stop cron, wait in-flight job, then close DB / HTTP
5. If you ever run **multiple** servers, move scheduling out of process (OS/cloud) or add leader election — SQLite + multi-writer is the wrong fight

Package: `internal/job` with one file per job (`job/refresh_tournaments.go`).

---

## Testing

### Preference: integration tests with real deps

| Fixture | Location | Use |
|---|---|---|
| Liquipedia HTML | `testdata/liquipedia/...` | Parse tests (already) |
| SQLite snapshots | `testdata/sqlite/<name>.sqlite` | API/service/repo tests |
| Optional HTTP recordings | `testdata/httprr/...` | If you mock outbound HTTP |

**Per test run:** copy snapshot to a temp file (or recreate via migrate + seed SQL), open that DB, never mutate the committed file.

```text
testdata/sqlite/empty.sqlite          # migrated schema, no rows
testdata/sqlite/tournament_asl20.sqlite
```

Helpers live in e.g. `internal/testutil/db.go`: `OpenFixture(t, "tournament_asl20") *sql.DB`.

### Live URL fetches

Allowed in integration tests **opt-in** (build tag or env), e.g.:

```bash
LIVE_LIQUIPEDIA=1 go test ./internal/liquipedia/... -count=1
```

Default CI = fixtures only (no network). Live tests must respect Liquipedia rate limits.

### Unit vs integration

- Pure parse/helpers: table tests on fixtures (fine as `*_test.go` in package)
- Handler + service + sqlite: `TestFoo_integration` with fixture DB + `httptesttest` server

---

## HTTP surface

- REST verbs in handlers: `GET` read, `POST` create/actions (e.g. parse)
- Consistent JSON errors: `{ "error": "..." }`
- Middleware chain on the mux: request ID → logging → CORS → auth → handlers
- Timeouts on server and outbound clients

---

## Things easy to forget (checklist)

- [ ] **Migrations** + version table  
- [ ] **Graceful shutdown** (HTTP + jobs + DB)  
- [ ] **Config** (DB path, listen addr, job schedules, feature flags)  
- [ ] **Context** on every service/repo/client call  
- [ ] **Domain errors** (`ErrNotFound`, `ErrForbidden`) mapped in handlers  
- [ ] **SQLite pragmas** (WAL, foreign_keys, busy_timeout)  
- [ ] **No TX across external HTTP**  
- [ ] **Idempotent cron** + overlap lock  
- [ ] **Fixture DB copy** per test  
- [ ] **gitignore** for `devdata/*.sqlite` and test temp dirs  
- [ ] **Logging** (structured) without secrets  
- [ ] **Frontend CORS** origins from config in non-dev  

---

## Quick dependency direction

```
handler → service → repository → sqlite
                ↘ liquipedia client / parse
middleware → (context values) → handler
job → service
```

`cmd/server` wires config → DB → migrate → repository → service → handler + job → listen.
