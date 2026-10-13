"use client";

import { ArrowLeft, GraduationCap } from "lucide-react";
import Link from "next/link";
import { useParams, useSearchParams } from "next/navigation";
import { Suspense } from "react";
import useSWR from "swr";

import { AppShell } from "@/components/AppShell";
import { Maths } from "@/components/Maths";
import { SessionProvider } from "@/components/SessionProvider";
import {
  ButtonLink,
  Card,
  EmptyState,
  ErrorState,
  Pill,
  Spinner,
} from "@/components/ui";
import { api } from "@/lib/api";
import type { TeachingStyle } from "@/lib/types";

const styles: TeachingStyle[] = [
  "simple-language",
  "analogy-driven",
  "scaffolded-learning",
];

/** Reads the teaching style out of the URL, falling back to simple language. */
function readStyle(value: string | null): TeachingStyle {
  return styles.includes(value as TeachingStyle)
    ? (value as TeachingStyle)
    : "simple-language";
}

function TeachingReader({ moduleId }: { moduleId: string }) {
  const params = useSearchParams();
  const style = readStyle(params.get("style"));

  const { data, error, isLoading } = useSWR(["teaching", moduleId, style], () =>
    api.teaching(moduleId, style),
  );

  if (isLoading) return <Spinner label="Loading your teaching cards" />;
  if (error) {
    return (
      <ErrorState
        message={
          error instanceof Error ? error.message : "Could not load the cards"
        }
      />
    );
  }

  const cards = data?.teachingCards ?? [];

  return (
    <>
      <Link
        href={`/modules/${moduleId}`}
        className="inline-flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-900"
      >
        <ArrowLeft className="size-4" aria-hidden />
        Back to module
      </Link>

      {cards.length === 0 ? (
        <EmptyState
          icon={GraduationCap}
          title="No teaching cards yet"
          description="Generate this style from the module page first."
          action={
            <ButtonLink href={`/modules/${moduleId}`}>Back to module</ButtonLink>
          }
        />
      ) : (
        <>
          <div className="mt-4 flex items-center gap-2">
            <Pill>{style.replace(/-/g, " ")}</Pill>
            <span className="text-sm text-stone-500">
              {cards.length} {cards.length === 1 ? "card" : "cards"}
            </span>
          </div>

          {/* Ordered so foundations come before what depends on them, which is
              what the scaffolded style relies on. */}
          <ol className="mt-6 space-y-4">
            {cards.map((card, i) => (
              <li key={card.id}>
                <Card>
                  <div className="flex items-baseline gap-3">
                    <span
                      aria-hidden
                      className="font-serif text-sm text-stone-400"
                    >
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    <h2 className="font-serif text-xl font-medium text-stone-900">
                      <Maths>{card.subtopic}</Maths>
                    </h2>
                  </div>

                  <div className="mt-3 whitespace-pre-line text-[0.95rem] leading-relaxed text-stone-700">
                    <Maths>{card.teaching}</Maths>
                  </div>

                  {card.fileName && (
                    <p className="mt-4 text-xs text-stone-400">
                      from {card.fileName}
                    </p>
                  )}
                </Card>
              </li>
            ))}
          </ol>
        </>
      )}
    </>
  );
}

export default function TeachPage() {
  const params = useParams<{ moduleId: string }>();

  return (
    <SessionProvider>
      <AppShell>
        {/* useSearchParams needs a boundary, because the page is streamed. */}
        <Suspense fallback={<Spinner label="Loading your teaching cards" />}>
          <TeachingReader moduleId={params.moduleId} />
        </Suspense>
      </AppShell>
    </SessionProvider>
  );
}
