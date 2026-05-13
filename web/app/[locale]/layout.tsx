import { notFound } from "next/navigation";
import Link from "next/link";
import SessionNav from "@/components/SessionNav";
import LangSwitch from "@/components/LangSwitch";
import type { Locale } from "@/lib/curriculum";

const LOCALES: Locale[] = ["zh", "en"];

export async function generateStaticParams() {
  return LOCALES.map((l) => ({ locale: l }));
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  if (!LOCALES.includes(locale as Locale)) notFound();
  const l = locale as Locale;

  return (
    <div className="min-h-screen flex">
      <aside className="w-72 shrink-0 border-r border-[var(--border)] bg-[var(--bg)] sticky top-0 h-screen overflow-y-auto p-5">
        <div className="flex items-center justify-between mb-5">
          <Link href={`/${l}`} className="font-semibold text-base">
            learn-langfuse
          </Link>
          <LangSwitch locale={l} />
        </div>
        <SessionNav locale={l} />
        <div className="mt-8 text-xs text-[var(--fg-muted)] leading-relaxed">
          {l === "zh"
            ? "用 Go 从零渐进重建 langfuse 的摄入 + 可观测性核心。每节末尾对照上游 TypeScript。"
            : "Rebuild langfuse's ingestion + observability core in Go, chapter by chapter. Each chapter ends with an upstream TypeScript source reading."}
        </div>
      </aside>
      <main className="flex-1 px-8 py-10 overflow-x-hidden">{children}</main>
    </div>
  );
}
