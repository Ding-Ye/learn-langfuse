# s01 — Trace ingestion

> **Upstream:** `web/src/pages/api/public/ingestion.ts` and
> `packages/shared/src/server/ingestion/processEventBatch.ts` at SHA
> [`1e3535e1`](https://github.com/langfuse/langfuse/tree/1e3535e126fc918781f45753706ff0f576b12175).

## Problem

Langfuse is an LLM observability platform. The SDKs (Python, TS, Java, …)
each speak the same wire protocol: a periodic `POST` of a *batch* of
typed events. Every interesting feature downstream — traces UI, eval
scores, prompt management, datasets, webhooks — is just a different
projection over those events. So whatever sits at the ingestion door
has to:

1. **Accept a batch.** Network round-trips are expensive; SDKs
   buffer.
2. **Type events without freezing them.** Trace create, observation
   create, score create, sdk-log, …. The schema grows; old SDKs keep
   working.
3. **Tolerate partial failure.** One bad event in a batch of 200 must
   not kill the batch. Clients retry by `id`.
4. **Keep the storage layer pluggable.** Today it's a SQL primary +
   ClickHouse mirror; tomorrow it's something else.

s01 ports the *minimum subset* of those four requirements to Go: an
HTTP handler, a typed `Event` envelope, a `Store` interface, and one
in-memory implementation.

## Solution

```go
type Server struct { Store Store }
//   POST /api/public/ingestion      → BatchResponse
//   GET  /api/public/traces/<id>    → {trace, observations, scores}

type Event struct {
    ID        string
    Type      EventType  // trace-create | observation-create | score-create
    Timestamp time.Time
    Body      Body       // discriminated payload
}

func Apply(s Store, e Event) error  // validates, then upserts.
```

`Store` is an interface; `MemStore` is the only implementation in s01.
Upstream fans `Apply` out to Prisma (Postgres) + a ClickHouse writer;
s07 will reproduce the ClickHouse half.

## How it works

```
SDK client                  ingest server                      MemStore
─────────                   ──────────────                     ────────
POST /api/public/ingestion
    {batch: [{e1, ...}, ...]}
                ──────▶
                            handler.ingest:
                              decode JSON
                              for each e in batch:
                                Apply(store, e)
                                  → Event.Validate()
                                  → switch e.Type {
                                      trace-create → UpsertTrace
                                      observation-create → UpsertObs
                                      score-create → UpsertScore
                                    }
                                if err:  errors  += {id, msg}
                                else:    successes += {id, 200}
                                                                  ↓
                                                              traces[id]
                                                              obs[traceID][id]
                                                              scores[traceID]
                              status = 200 / 207 / 400 depending on mix
                ◀──────
    {successes:[...], errors:[...]}

GET /api/public/traces/T
                ──────▶
                            handler.getTrace:
                              t   = store.GetTrace(T)
                              obs = store.ListObservations(T)   ← sorted by StartNS
                              sc  = store.ListScores(T)
                ◀──────
    {trace:..., observations:[...], scores:[...]}
```

Key choices and their upstream counterparts:

- **Discriminated `Event.Type` with one big `Body`.** Mirrors
  upstream's Zod discriminated-union schema. Cheaper than a sealed
  hierarchy because we only need one decoder.
- **Per-event Validate; per-event success/error.** 207 Multi-Status
  is what upstream returns and what the SDKs already understand.
- **Store as interface, MemStore as impl.** The HTTP layer never
  references Postgres or ClickHouse. s07 swaps in a column-oriented
  writer with no handler changes.
- **`StartNS / EndNS` in nanoseconds, not `time.Time`.** Mirrors what
  ClickHouse will eventually want; saves a round-trip when s07 takes
  over the store.

## What changed

This is the first chapter. The diff is against an empty workspace.
The load-bearing decisions are:

- The `Event` shape (one envelope, typed body, server-side validation).
- The `Store` shape (three upsert methods + three list methods).
- The HTTP surface (one POST, one GET, 207 for mixed batches).

Every later chapter consumes these — auth (s04) wraps the handler,
queue (s05) sits between handler and `Apply`, ClickHouse writer (s07)
implements `Store`.

## Try it

```sh
cd agents/s01-trace-ingestion
go test ./...         # 5 tests
go run ./cmd/demo     # starts on 127.0.0.1:NNNN, POSTs a batch, GETs the trace
```

The demo prints the full projected trace shape — that's the same JSON
the upstream UI's trace-detail page would render against ClickHouse.

## Upstream Source Reading

Open [`upstream-readings/s01-trace-ingestion.py`](../../upstream-readings/s01-trace-ingestion.py).
It excerpts the Next.js handler and the `processEventBatch` switch,
with comments showing exactly which Go construct corresponds to which
TS function.

Then read the upstream files in full:

- [`web/src/pages/api/public/ingestion.ts`](https://github.com/langfuse/langfuse/blob/1e3535e126fc918781f45753706ff0f576b12175/web/src/pages/api/public/ingestion.ts)
- [`packages/shared/src/server/ingestion/processEventBatch.ts`](https://github.com/langfuse/langfuse/blob/1e3535e126fc918781f45753706ff0f576b12175/packages/shared/src/server/ingestion)
- [`packages/shared/src/eventsTable.ts`](https://github.com/langfuse/langfuse/blob/1e3535e126fc918781f45753706ff0f576b12175/packages/shared/src/eventsTable.ts) — the ClickHouse column shape, useful when s07 takes over.

Pay attention to how the upstream handler keeps its auth + rate-limit
+ OTel context-propagation cleanly separable from the validation +
write logic. That's why each can be a chapter of its own in this repo.
