import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "learn-langfuse",
  description:
    "Learn how langfuse's ingestion + observability backend really works by building a Go mini-version, chapter by chapter.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh">
      <body>{children}</body>
    </html>
  );
}
