# s01 — Trace ingestion

The smallest interesting subset of langfuse's `POST /api/public/ingestion`
endpoint, in pure Go.

## Run

```sh
go test ./...        # 5 tests, all green
go run ./cmd/demo    # boot HTTP server, POST batch, GET trace back
```

## What this teaches

- **Batch-shaped ingest with discriminated events.** One HTTP POST,
  `Event[]` with `type` ∈ {trace-create, observation-create,
  score-create}. Per-event validation; partial failure returns 207.
- **Materialised store with three rows.** `Trace`, `Observation`
  (with parent_id → tree), `Score`. The store is an interface; s01
  ships a `sync.RWMutex`-guarded in-memory implementation, s07 will
  swap it for a column-oriented writer.
- **Public-API projection.** `GET /api/public/traces/<id>` returns a
  nested view (trace + sorted observations + scores) — the same shape
  the upstream UI's trace-detail page renders.
- **Zero dependencies.** stdlib only.

## Files

| File | What | Lines |
|------|------|------|
| `ingest.go` | `Event` envelope + discriminated `Body` + `Validate` | ~140 |
| `store.go` | `Store` interface + `MemStore` + `Apply` switch | ~165 |
| `server.go` | HTTP routes (`/api/public/ingestion`, `/api/public/traces/`) | ~95 |
| `server_test.go` | 5 tests covering single-batch / partial-failure / projection / 415 / healthz | ~170 |
| `cmd/demo/main.go` | scripted 4-event batch + pretty-printed projection | ~70 |

## Six-section spine

Full mental model, ASCII diagram, diff-from-empty, hands-on, and
upstream source reading live in
[`docs/en/s01-trace-ingestion.md`](../../docs/en/s01-trace-ingestion.md) /
[`docs/zh/s01-trace-ingestion.md`](../../docs/zh/s01-trace-ingestion.md).
