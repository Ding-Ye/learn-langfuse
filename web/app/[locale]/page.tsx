import Link from "next/link";
import { notFound } from "next/navigation";
import { CURRICULUM, chapterTitle, type Locale } from "@/lib/curriculum";

export default async function Landing({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  if (locale !== "zh" && locale !== "en") notFound();
  const l = locale as Locale;

  const intro = l === "zh" ? INTRO_ZH : INTRO_EN;
  const ctaLabel = l === "zh" ? "从 s01 开始 →" : "Start at s01 →";

  return (
    <article className="prose-doc">
      <h1>learn-langfuse</h1>
      <p className="text-[var(--fg-muted)]">
        {l === "zh"
          ? "用 Go 从零渐进重建 langfuse 的摄入 + 可观测性核心，每节末尾对照上游 TypeScript 源码。"
          : "Rebuild langfuse's ingestion + observability core in Go from scratch, chapter by chapter — each one ends with an upstream TypeScript source reading."}
      </p>

      {intro.map((p, i) => (
        <p key={i}>{p}</p>
      ))}

      <p>
        <Link
          href={`/${l}/s/s01-trace-ingestion`}
          className="inline-block mt-2 px-4 py-2 rounded border border-[var(--accent-soft)] hover:border-[var(--accent)]"
        >
          {ctaLabel}
        </Link>
      </p>

      <h2>{l === "zh" ? "课程" : "Curriculum"}</h2>
      <ul>
        {CURRICULUM.map((c) => (
          <li key={c.slug}>
            <span className="font-mono text-[var(--fg-muted)] mr-2">
              {c.num}
            </span>
            {c.available ? (
              <Link href={`/${l}/s/${c.slug}`}>{chapterTitle(c, l)}</Link>
            ) : (
              <span className="text-[var(--fg-muted)]">
                {chapterTitle(c, l)}{" "}
                <span className="text-xs">
                  ({l === "zh" ? "未发布" : "not yet"})
                </span>
              </span>
            )}
          </li>
        ))}
      </ul>
    </article>
  );
}

const INTRO_ZH = [
  "Langfuse 是 YC 投的开源 LLM 可观测性平台——traces / evals / prompt management / datasets / sessions / cost。主仓库约 52 万行的 pnpm monorepo：Next.js web、BullMQ worker、shared 包、背后挂 ClickHouse + Postgres + Redis。",
  "这个仓库不是教你「部署」 langfuse，而是把它的摄入 + 可观测性核心一节一节重建出来——typed event 摄入、span/trace 父子树、rate limit、API auth、async queue、blob 缓存、ClickHouse writer、eval scores、prompt 管理、数据集、webhook、OTel bridge、sessions/users、成本、RBAC、cloud metering——每节加一个机制，用 Go 写一份精简实现。",
  "Go 实现是教学骨架，上游 TypeScript 是生产实现。每节末尾的「上游源码阅读」用固定 commit 的 permalink 锚定到上游真实代码。",
];

const INTRO_EN = [
  "Langfuse is the YC-backed open-source LLM observability platform — traces, evals, prompt management, datasets, sessions, costs. The main repo is a ~520 K-LOC pnpm monorepo: Next.js web, BullMQ worker, shared package, ClickHouse + Postgres + Redis behind it.",
  "This repo doesn't teach you how to *deploy* langfuse. It rebuilds the ingestion + observability core in Go, one mechanism per chapter — typed event ingest, span/trace parent-child trees, rate limit, API auth, async queue, blob cache, ClickHouse writer, eval scores, prompt management, datasets, webhooks, OTel bridge, sessions/users, cost, RBAC, cloud metering. After sixteen chapters, the upstream stack stops looking like magic.",
  "Go is the teaching skeleton; the upstream TypeScript is the production implementation. Every 'Upstream Source Reading' pins permalinks against a frozen upstream SHA.",
];
