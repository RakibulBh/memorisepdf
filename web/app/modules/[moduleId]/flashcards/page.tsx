"use client";

import { ArrowLeft, ChevronLeft, ChevronRight, Layers, Shuffle } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import useSWR from "swr";

import { AppShell } from "@/components/AppShell";
import { Maths } from "@/components/Maths";
import { SessionProvider } from "@/components/SessionProvider";
import {
  Button,
  ButtonLink,
  EmptyState,
  ErrorState,
  Spinner,
} from "@/components/ui";
import { api } from "@/lib/api";

/** Puts the cards in a random order, so they are not always learned in sequence. */
function shuffle<T>(items: T[]): T[] {
  const copy = [...items];
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [copy[i], copy[j]] = [copy[j]!, copy[i]!];
  }
  return copy;
}

function FlashcardReview({ moduleId }: { moduleId: string }) {
  const { data, error, isLoading } = useSWR(["flashcards", moduleId], () =>
    api.flashcards(moduleId),
  );

  const [round, setRound] = useState(0);
  const cards = useMemo(
    () => (data ? shuffle(data.flashcards) : []),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [data, round],
  );

  const [index, setIndex] = useState(0);
  const [flipped, setFlipped] = useState(false);

  const total = cards.length;

  const go = (delta: number) => {
    setFlipped(false);
    setIndex((i) => (i + delta + total) % total);
  };

  // Arrow keys move between cards and space flips, so a deck can be worked
  // through without reaching for the mouse.
  useEffect(() => {
    if (total === 0) return;

    const onKey = (e: KeyboardEvent) => {
      if (e.key === "ArrowRight") go(1);
      else if (e.key === "ArrowLeft") go(-1);
      else if (e.key === " " || e.key === "Enter") {
        e.preventDefault();
        setFlipped((f) => !f);
      }
    };

    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [total]);

  if (isLoading) return <Spinner label="Loading your flashcards" />;
  if (error) {
    return (
      <ErrorState
        message={
          error instanceof Error ? error.message : "Could not load the flashcards"
        }
      />
    );
  }

  return (
    <>
      <Link
        href={`/modules/${moduleId}`}
        className="inline-flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-900"
      >
        <ArrowLeft className="size-4" aria-hidden />
        Back to module
      </Link>

      {total === 0 ? (
        <EmptyState
          icon={Layers}
          title="No flashcards yet"
          description="Generate flashcards from the module page first."
          action={
            <ButtonLink href={`/modules/${moduleId}`}>Back to module</ButtonLink>
          }
        />
      ) : (
        <>
          <div className="mt-4 flex items-center justify-between">
            <span className="text-sm text-stone-500">
              Card {index + 1} of {total}
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setRound((r) => r + 1);
                setIndex(0);
                setFlipped(false);
              }}
            >
              <Shuffle className="size-4" aria-hidden />
              Shuffle
            </Button>
          </div>

          <div className="card-scene mt-4">
            <button
              onClick={() => setFlipped((f) => !f)}
              aria-label={flipped ? "Show the term" : "Show the definition"}
              className="block w-full text-left"
            >
              <div
                className={`card-inner relative min-h-72 w-full ${
                  flipped ? "is-flipped" : ""
                }`}
              >
                {/* Front: the term */}
                <div className="card-face absolute inset-0 flex flex-col items-center justify-center rounded-xl bg-white p-8 text-center ring-1 ring-stone-200">
                  <p className="text-xs uppercase tracking-wide text-stone-400">
                    Term
                  </p>
                  <p className="mt-3 font-serif text-2xl font-medium text-stone-900">
                    <Maths>{cards[index]!.term}</Maths>
                  </p>
                  <p className="mt-6 text-xs text-stone-400">
                    click or press space to flip
                  </p>
                </div>

                {/* Back: the definition */}
                <div className="card-face card-face-back absolute inset-0 flex flex-col justify-center rounded-xl bg-stone-900 p-8 text-white">
                  <p className="text-xs uppercase tracking-wide text-stone-400">
                    Definition
                  </p>
                  <p className="mt-3 text-base leading-relaxed">
                    <Maths>{cards[index]!.definition}</Maths>
                  </p>
                  {cards[index]!.fileName && (
                    <p className="mt-5 text-xs text-stone-500">
                      from {cards[index]!.fileName}
                    </p>
                  )}
                </div>
              </div>
            </button>
          </div>

          <div className="mt-5 flex items-center justify-center gap-3">
            <Button variant="secondary" onClick={() => go(-1)}>
              <ChevronLeft className="size-4" aria-hidden />
              Previous
            </Button>
            <Button variant="secondary" onClick={() => go(1)}>
              Next
              <ChevronRight className="size-4" aria-hidden />
            </Button>
          </div>
        </>
      )}
    </>
  );
}

export default function FlashcardsPage() {
  const params = useParams<{ moduleId: string }>();

  return (
    <SessionProvider>
      <AppShell>
        <FlashcardReview moduleId={params.moduleId} />
      </AppShell>
    </SessionProvider>
  );
}
