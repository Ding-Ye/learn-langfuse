# s01 — Trace 摄入

> **上游：** `web/src/pages/api/public/ingestion.ts` 与
> `packages/shared/src/server/ingestion/processEventBatch.ts`，commit
> [`1e3535e1`](https://github.com/langfuse/langfuse/tree/1e3535e126fc918781f45753706ff0f576b12175)。

## 问题

Langfuse 是 LLM 应用的可观测性平台。多语言 SDK（Python / TS / Java / …）讲的是同一个线协议：周期性 POST 一个 *batch* 的 typed event。下游所有有意思的功能——traces UI、eval 分数、prompt 管理、数据集、webhook——都不过是这些事件的不同投影。所以站在摄入门口的那段代码必须做到：

1. **接收 batch。** 网络 round-trip 贵，SDK 端会做缓冲。
2. **类型化但不僵化。** trace-create / observation-create / score-create / sdk-log / …。schema 会演进，老 SDK 不能跪。
3. **容忍部分失败。** 200 条 batch 里坏一条不能拖死整批，客户端按 `id` 重试。
4. **存储层可替换。** 今天是 SQL 主库 + ClickHouse 镜像，明天可能换。

s01 把这四件事里 *最小够用* 的一份移植到 Go：一个 HTTP handler、一个 typed `Event` envelope、一个 `Store` 接口、一份内存实现。

## 解法

```go
type Server struct { Store Store }
//   POST /api/public/ingestion      → BatchResponse
//   GET  /api/public/traces/<id>    → {trace, observations, scores}

type Event struct {
    ID        string
    Type      EventType  // trace-create | observation-create | score-create
    Timestamp time.Time
    Body      Body       // 区分式 payload
}

func Apply(s Store, e Event) error  // 先验证，再 upsert。
```

`Store` 是接口；s01 只发布 `MemStore`。上游把 `Apply` 扇到 Prisma（Postgres）+ ClickHouse writer；s07 把 ClickHouse 那半边还原。

## 工作原理

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
                              status = 200 / 207 / 400 视混合情况而定
                ◀──────
    {successes:[...], errors:[...]}

GET /api/public/traces/T
                ──────▶
                            handler.getTrace:
                              t   = store.GetTrace(T)
                              obs = store.ListObservations(T)   ← 按 StartNS 排序
                              sc  = store.ListScores(T)
                ◀──────
    {trace:..., observations:[...], scores:[...]}
```

几个关键选择，以及它们对应上游哪段：

- **`Event.Type` 区分 + 一个大 `Body`。** 对应上游 Zod 的 discriminated-union schema。比 sealed hierarchy 便宜，因为我们只需要一个 decoder。
- **逐事件 Validate + 逐事件 success/error。** 207 Multi-Status 是上游返的，也是 SDK 已经认识的。
- **Store 是接口、MemStore 是实现。** HTTP 层不引用 Postgres 或 ClickHouse。s07 换写者时 handler 一行不动。
- **`StartNS / EndNS` 用纳秒，不用 `time.Time`。** 对齐 ClickHouse 最终要的形态；s07 接手存储时省一次转换。

## 与上一节的 diff

这是第一章，diff 相对空 workspace。承重决定有三条：

- `Event` 的形状（一个 envelope + typed body + 服务端验证）。
- `Store` 的形状（三个 upsert + 三个 list）。
- HTTP 表面（一个 POST、一个 GET、混合 batch 返 207）。

后续每章都消费这三条——auth（s04）包 handler，queue（s05）插在 handler 和 `Apply` 之间，ClickHouse writer（s07）实现 `Store`。

## 动手试

```sh
cd agents/s01-trace-ingestion
go test ./...         # 5 个测试
go run ./cmd/demo     # 起在 127.0.0.1:NNNN，POST 一个 batch，GET trace
```

demo 会把完整投影的 trace 打印出来——和上游 UI 的 trace 详情页对 ClickHouse 渲染出来的 JSON 是同一份形状。

## 上游源码阅读

打开 [`upstream-readings/s01-trace-ingestion.py`](../../upstream-readings/s01-trace-ingestion.py)。它节选了 Next.js handler 和 `processEventBatch` switch，注释里标出了每个 Go 构造对应的 TS 函数。

之后把上游文件完整看一遍：

- [`web/src/pages/api/public/ingestion.ts`](https://github.com/langfuse/langfuse/blob/1e3535e126fc918781f45753706ff0f576b12175/web/src/pages/api/public/ingestion.ts)
- [`packages/shared/src/server/ingestion/processEventBatch.ts`](https://github.com/langfuse/langfuse/blob/1e3535e126fc918781f45753706ff0f576b12175/packages/shared/src/server/ingestion)
- [`packages/shared/src/eventsTable.ts`](https://github.com/langfuse/langfuse/blob/1e3535e126fc918781f45753706ff0f576b12175/packages/shared/src/eventsTable.ts)——ClickHouse 列定义，s07 接手时有用。

特别留意上游 handler 是怎么把 auth + rate-limit + OTel context 干净地从 validation + write 里隔出来的。正因为这种分层，这个仓库才有可能每节挑一件事来教。
