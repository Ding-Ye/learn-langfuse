# Upstream excerpt: web/src/pages/api/public/ingestion.ts
# (langfuse is TypeScript — we keep the .py filename for tool consistency
# across the learn-* series, but the snippets below are .ts.)
#
# Pinned to upstream commit 1e3535e126fc918781f45753706ff0f576b12175.
# Permalinks:
#   https://github.com/langfuse/langfuse/blob/1e3535e126fc918781f45753706ff0f576b12175/web/src/pages/api/public/ingestion.ts
#   https://github.com/langfuse/langfuse/blob/1e3535e126fc918781f45753706ff0f576b12175/packages/shared/src/server/ingestion/processEventBatch.ts
#
# License: MIT (langfuse's main tree; `ee/` is excluded).
# The annotations below are mine; the code is upstream's, lightly trimmed.


# ------------------------------------------------------------------
# web/src/pages/api/public/ingestion.ts — the Next.js handler.
# Three stages: validate → async dispatch → sync fallback. s01 collapses
# stage 2 (queue + S3) into stage 3 (sync). They come back in s05 + s06.
# ------------------------------------------------------------------

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse,
) {
  try {
    await runMiddleware(req, res, cors);

    // 1. AUTH — s04 ApiAuthService. In s01 we trust the caller.
    const authCheck = await new ApiAuthService(
      prisma, redis,
    ).verifyAuthHeaderAndReturnScope(req.headers.authorization);
    if (!authCheck.validKey) throw new UnauthorizedError(authCheck.error);

    // 2. RATE LIMIT — s03. In s01 we never throttle.
    const rateLimit = await RateLimitService.getInstance().rateLimitRequest(...);

    // 3. PARSE — Zod discriminated union per-event.
    //    Same shape as our Go BatchRequest/Event/Body. Upstream's Zod
    //    schema lives in shared/src/server/ingestion/types.ts.
    const eventBatch = ingestionApiSchema.parse(req.body);

    // 4. PROCESS — async path uploads each event to S3 then enqueues to
    //    BullMQ ingestionQueue; on error falls back to sync. s01 is the
    //    sync path only.
    const result = await processEventBatch({
      events: eventBatch.batch,
      authCheck,
    });

    // 5. RESPONSE — 207 Multi-Status if any error / any success.
    if (result.errors.length && result.successes.length)
      return res.status(207).json(result);
    if (result.errors.length)
      return res.status(400).json(result);
    return res.status(200).json(result);
  } catch (e) {
    ...
  }
}


# ------------------------------------------------------------------
# packages/shared/src/server/ingestion/processEventBatch.ts (shape only).
# This is the seam we model with our Go `Apply(store, event)`. Upstream
# does fan-out to per-event handlers; we collapse to a switch on Type.
# ------------------------------------------------------------------

export async function processEventBatch({ events, authCheck }: Params) {
  const successes: EventAck[] = [];
  const errors: EventReject[] = [];

  for (const e of events) {
    try {
      // Validate per-event. Same shape as our Event.Validate() in Go.
      ingestionEventSchema.parse(e);

      switch (e.type) {
        case "trace-create":
          // ↔ Go: UpsertTrace(Trace{...})
          await prisma.trace.upsert({ where: { id: e.body.trace_id }, ... });
          break;
        case "observation-create":
          // ↔ Go: UpsertObservation(Observation{...})
          await prisma.observation.upsert({ ... });
          break;
        case "score-create":
          // ↔ Go: UpsertScore(Score{...})
          await prisma.score.upsert({ ... });
          break;
        ...
      }
      successes.push({ id: e.id, status: 200 });
    } catch (err) {
      errors.push({ id: e.id, status: 400, message: err.message });
    }
  }
  return { successes, errors };
}


# ------------------------------------------------------------------
# Deliberate omissions in s01 (we'll come back to these):
# - S3 blob cache for replay → s06 blob-storage
# - BullMQ queue + worker drain → s05 queue-worker
# - ClickHouse column-oriented writer → s07 clickhouse-writer
# - per-trace OTel context propagation → s12 otel-bridge
# - ApiAuthService (sk- key lookup, project scoping) → s04 api-auth
# - RateLimitService (per-key token bucket) → s03 rate-limiter
# ------------------------------------------------------------------
