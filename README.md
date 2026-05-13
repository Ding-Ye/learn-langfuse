# learn-langfuse

> Re-grow the langfuse ingestion + observability backend from scratch in Go — one mechanism per chapter, each ending with a permalinked reading of the upstream TypeScript.

[![Go](https://github.com/Ding-Ye/learn-langfuse/actions/workflows/go.yml/badge.svg)](https://github.com/Ding-Ye/learn-langfuse/actions/workflows/go.yml)
[![Docs](https://github.com/Ding-Ye/learn-langfuse/actions/workflows/docs.yml/badge.svg)](https://github.com/Ding-Ye/learn-langfuse/actions/workflows/docs.yml)
[![Web](https://github.com/Ding-Ye/learn-langfuse/actions/workflows/web.yml/badge.svg)](https://github.com/Ding-Ye/learn-langfuse/actions/workflows/web.yml)

## Why

[`langfuse/langfuse`](https://github.com/langfuse/langfuse) is the YC-backed open-source LLM observability platform — traces, evals, prompt management, datasets, sessions, costs. The main repo is a ~520 K-LOC pnpm monorepo: Next.js web app, BullMQ worker, shared package, ClickHouse + Postgres + Redis behind it. Reading it cold is rough: the SDK protocol, the Zod schemas, the worker pipeline, the ClickHouse column model, and the OTel bridge all live in different packages, and the most interesting code paths weave through all of them.

This repo rebuilds the **ingestion + observability core** in Go, one chapter at a time. Each chapter is ≈ 300–700 lines of code that compiles independently, with a `## Upstream Source Reading` section at the end pinning permalinks against a frozen upstream SHA. You don't read about discriminated-union ingest — you write a handler that 207s on partial failure, then go look at the TypeScript it's modelled on.

## 为什么

[`langfuse/langfuse`](https://github.com/langfuse/langfuse) 是 YC 投的开源 LLM 可观测性平台——traces / evals / prompt management / datasets / sessions / cost。主仓库是约 52 万行的 pnpm monorepo：Next.js web、BullMQ worker、shared 包、背后挂 ClickHouse + Postgres + Redis。直接冷读很难：SDK 协议、Zod schema、worker pipeline、ClickHouse 列模型、OTel bridge 分散在不同 package，最有意思的代码路径要穿过所有这些。

本仓库用 **Go** 一节一节把 **摄入 + 可观测性核心** 重建出来。每节 ≈ 300–700 行独立编译的代码，每节末尾的 `## 上游源码阅读` 用固定 commit 的 permalink 锚定到上游 TypeScript。你不是"读到 discriminated-union 摄入"——你是先写一个能在部分失败时返 207 的 handler，再回头看上游怎么写。

## Curriculum / 课程

| # | Slug | Title (EN) | 标题（中文） | Status |
|---|---|---|---|---|
| s01 | [`s01-trace-ingestion`](docs/en/s01-trace-ingestion.md) ([中](docs/zh/s01-trace-ingestion.md)) | Trace ingestion | Trace 摄入 | ✅ |
| s02 | `s02-spans-and-traces` | Spans, traces, and parent-child trees | Span / Trace 父子树 | ⏳ |
| s03 | `s03-rate-limiter` | Per-key token-bucket rate limiter | 按 key 的令牌桶限流 | ⏳ |
| s04 | `s04-api-auth` | Public-key API auth (sk-…) + project scope | Public-key 认证 + project scope | ⏳ |
| s05 | `s05-queue-worker` | Async queue + worker drain | 异步队列 + worker 消费 | ⏳ |
| s06 | `s06-blob-storage` | S3-style blob cache for replay | S3 风格 blob 缓存 | ⏳ |
| s07 | `s07-clickhouse-writer` | Column-oriented batch writer | 列式批量写入 | ⏳ |
| s08 | `s08-scores` | Eval scores attached to trace/observation | 绑定到 trace 的 eval score | ⏳ |
| s09 | `s09-prompt-management` | Versioned prompt registry | 版本化 prompt 注册表 | ⏳ |
| s10 | `s10-dataset-experiments` | Dataset replay against a model | 数据集回放评测 | ⏳ |
| s11 | `s11-webhook-fanout` | Per-trace webhook dispatch | 按 trace 的 webhook 扇出 | ⏳ |
| s12 | `s12-otel-bridge` | OTLP → langfuse event translator | OTLP → langfuse 事件桥 | ⏳ |
| s13 | `s13-sessions-and-users` | Cross-trace session rollup | 跨 trace 的 session 汇总 | ⏳ |
| s14 | `s14-cost-attribution` | Per-model token cost calc | 按模型的 token 成本结算 | ⏳ |
| s15 | `s15-rbac-and-projects` | Multi-tenant project RBAC | 多租户 project RBAC | ⏳ |
| s16 | `s16-cloud-metering` | Billing usage rollup | 计费用量汇总 | ⏳ |
| s_full | `s_full-integration` | End-to-end integration | 端到端集成 | ⏳ |
| A | `appendix-a-observability-as-events` | Observability-as-events pattern | 把可观测性建模为事件流 | ⏳ |
| B | `appendix-b-upstream-map` | Upstream source-reading map | 上游源码导读地图 | ⏳ |

> ⏳ = curriculum slot reserved, not yet implemented. The schedule drips them in via the `learn-repo-generator` skill.

## Quickstart

```sh
cd agents/s01-trace-ingestion
go test ./...              # 5 tests, all green
go run ./cmd/demo          # starts HTTP server, POSTs a batch, GETs the trace
```

Browse the curriculum docs:

```sh
cd web
npm install
npm run dev   # http://localhost:3000/en or /zh
```

## Layout

```
.
├── agents/                          # one Go module per chapter
│   └── s01-trace-ingestion/
│       ├── ingest.go                # Event envelope + discriminated Body + Validate
│       ├── store.go                 # Store interface + MemStore + Apply switch
│       ├── server.go                # HTTP routes (/api/public/ingestion, /traces/)
│       ├── server_test.go           # 5 tests
│       └── cmd/demo/main.go         # scripted 4-event batch + projection
├── docs/
│   ├── en/s01-trace-ingestion.md
│   └── zh/s01-trace-ingestion.md
├── upstream-readings/
│   └── s01-trace-ingestion.py       # annotated TS excerpt (.py extension for tool consistency)
├── web/                             # Next.js bilingual doc viewer
├── .github/workflows/               # Go / web / docs CI
├── go.work
├── LICENSE                          # MIT, attributes upstream; excludes upstream's ee/
└── README.md
```

## Acknowledgements / 致谢

This repo is a learning derivative of [`langfuse/langfuse`](https://github.com/langfuse/langfuse), pinned at commit [`1e3535e1`](https://github.com/langfuse/langfuse/tree/1e3535e126fc918781f45753706ff0f576b12175). All upstream credit to the Langfuse team. Annotations, Go reimplementations, and bilingual docs are the contribution of this repo and ship under the same MIT license.

## License

MIT — see [LICENSE](./LICENSE). Upstream langfuse is also MIT outside its `ee/`, `web/src/ee/`, and `worker/src/ee/` directories; those directories are excluded from this learn-* derivative.
