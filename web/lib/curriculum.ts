// Curriculum locked from .learn/plan.md.

export type ChapterMeta = {
  slug: string;
  num: string;
  title: { zh: string; en: string };
  available: boolean;
};

export const CURRICULUM: ChapterMeta[] = [
  {
    slug: "s01-trace-ingestion",
    num: "s01",
    title: { zh: "Trace 摄入", en: "Trace ingestion" },
    available: true,
  },
  {
    slug: "s02-spans-and-traces",
    num: "s02",
    title: { zh: "Span / Trace 父子树", en: "Spans, traces, and parent-child trees" },
    available: false,
  },
  {
    slug: "s03-rate-limiter",
    num: "s03",
    title: { zh: "按 key 的令牌桶限流", en: "Per-key token-bucket rate limiter" },
    available: false,
  },
  {
    slug: "s04-api-auth",
    num: "s04",
    title: { zh: "Public-key 认证 + project scope", en: "Public-key API auth + project scope" },
    available: false,
  },
  {
    slug: "s05-queue-worker",
    num: "s05",
    title: { zh: "异步队列 + worker 消费", en: "Async queue + worker drain" },
    available: false,
  },
  {
    slug: "s06-blob-storage",
    num: "s06",
    title: { zh: "S3 风格 blob 缓存", en: "S3-style blob cache for replay" },
    available: false,
  },
  {
    slug: "s07-clickhouse-writer",
    num: "s07",
    title: { zh: "列式批量写入", en: "Column-oriented batch writer" },
    available: false,
  },
  {
    slug: "s08-scores",
    num: "s08",
    title: { zh: "绑定到 trace 的 eval score", en: "Eval scores attached to trace/observation" },
    available: false,
  },
  {
    slug: "s09-prompt-management",
    num: "s09",
    title: { zh: "版本化 prompt 注册表", en: "Versioned prompt registry" },
    available: false,
  },
  {
    slug: "s10-dataset-experiments",
    num: "s10",
    title: { zh: "数据集回放评测", en: "Dataset replay against a model" },
    available: false,
  },
  {
    slug: "s11-webhook-fanout",
    num: "s11",
    title: { zh: "按 trace 的 webhook 扇出", en: "Per-trace webhook dispatch" },
    available: false,
  },
  {
    slug: "s12-otel-bridge",
    num: "s12",
    title: { zh: "OTLP → langfuse 事件桥", en: "OTLP → langfuse event translator" },
    available: false,
  },
  {
    slug: "s13-sessions-and-users",
    num: "s13",
    title: { zh: "跨 trace 的 session 汇总", en: "Cross-trace session rollup" },
    available: false,
  },
  {
    slug: "s14-cost-attribution",
    num: "s14",
    title: { zh: "按模型的 token 成本结算", en: "Per-model token cost calc" },
    available: false,
  },
  {
    slug: "s15-rbac-and-projects",
    num: "s15",
    title: { zh: "多租户 project RBAC", en: "Multi-tenant project RBAC" },
    available: false,
  },
  {
    slug: "s16-cloud-metering",
    num: "s16",
    title: { zh: "计费用量汇总", en: "Billing usage rollup" },
    available: false,
  },
  {
    slug: "s_full-integration",
    num: "s_full",
    title: { zh: "端到端集成", en: "End-to-end integration" },
    available: false,
  },
  {
    slug: "appendix-a-observability-as-events",
    num: "A",
    title: {
      zh: "附录 A · 把可观测性建模为事件流",
      en: "Appendix A · Observability-as-events pattern",
    },
    available: false,
  },
  {
    slug: "appendix-b-upstream-map",
    num: "B",
    title: { zh: "附录 B · 上游源码导读地图", en: "Appendix B · Upstream source-reading map" },
    available: false,
  },
];

export type Locale = "zh" | "en";

export function chapterTitle(c: ChapterMeta, locale: Locale): string {
  return c.title[locale];
}
