"use client";

import { ArrowLeft, Check, RotateCcw, X } from "lucide-react";
import Link from "next/link";
import { useParams, useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";
import useSWR from "swr";

import { AppShell } from "@/components/AppShell";
import { Maths } from "@/components/Maths";
import { SessionProvider } from "@/components/SessionProvider";
import {
  Button,
  ButtonLink,
  Card,
  EmptyState,
  ErrorState,
  Pill,
  Spinner,
  cn,
} from "@/components/ui";
import { api } from "@/lib/api";
import type { Difficulty } from "@/lib/types";

const difficulties: Difficulty[] = ["easy", "medium", "hard"];

/** Reads the difficulty out of the URL, falling back to medium. */
function readDifficulty(value: string | null): Difficulty {
  return difficulties.includes(value as Difficulty)
    ? (value as Difficulty)
    : "medium";
}

/**
 * Shuffles the questions so the quiz is not in the same order every time.
 * Answer order comes from the model, which is told to vary it.
 */
function shuffle<T>(items: T[]): T[] {
  const copy = [...items];
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [copy[i], copy[j]] = [copy[j]!, copy[i]!];
  }
  return copy;
}

function QuizRunner({ moduleId }: { moduleId: string }) {
  const params = useSearchParams();
  const difficulty = readDifficulty(params.get("difficulty"));

  const { data, error, isLoading } = useSWR(["quiz", moduleId, difficulty], () =>
    api.quiz(moduleId, difficulty),
  );

  // Re-shuffled only when a new set of questions arrives, so answering one does
  // not reorder the rest underneath you.
  const [round, setRound] = useState(0);
  const questions = useMemo(
    () => (data ? shuffle(data.questions) : []),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [data, round],
  );

  const [index, setIndex] = useState(0);
  const [chosen, setChosen] = useState<string | null>(null);
  const [score, setScore] = useState({ right: 0, total: 0 });
  const [finished, setFinished] = useState(false);

  if (isLoading) return <Spinner label="Loading your quiz" />;
  if (error) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Could not load the quiz"}
      />
    );
  }

  const restart = () => {
    setRound((r) => r + 1);
    setIndex(0);
    setChosen(null);
    setScore({ right: 0, total: 0 });
    setFinished(false);
  };

  if (questions.length === 0) {
    return (
      <>
        <BackLink moduleId={moduleId} />
        <EmptyState
          icon={RotateCcw}
          title={`No ${difficulty} questions yet`}
          description="Generate this difficulty from the module page first."
          action={
            <ButtonLink href={`/modules/${moduleId}`}>Back to module</ButtonLink>
          }
        />
      </>
    );
  }

  if (finished) {
    const percent = Math.round((score.right / score.total) * 100);
    return (
      <>
        <BackLink moduleId={moduleId} />
        <Card className="mt-6 text-center">
          <p className="font-serif text-5xl font-semibold text-stone-900">
            {percent}%
          </p>
          <p className="mt-2 text-stone-600">
            {score.right} of {score.total} correct on {difficulty}
          </p>
          <div className="mt-6 flex justify-center gap-3">
            <Button onClick={restart}>
              <RotateCcw className="size-4" aria-hidden />
              Go again
            </Button>
            <ButtonLink variant="secondary" href={`/modules/${moduleId}`}>
              Back to module
            </ButtonLink>
          </div>
        </Card>
      </>
    );
  }

  const question = questions[index]!;
  const answered = chosen !== null;

  const choose = (answerId: string, correct: boolean) => {
    if (answered) return;
    setChosen(answerId);
    setScore((s) => ({ right: s.right + (correct ? 1 : 0), total: s.total + 1 }));
  };

  const next = () => {
    if (index + 1 >= questions.length) {
      setFinished(true);
      return;
    }
    setIndex((i) => i + 1);
    setChosen(null);
  };

  return (
    <>
      <BackLink moduleId={moduleId} />

      <div className="mt-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Pill>{difficulty}</Pill>
          <span className="text-sm text-stone-500">
            Question {index + 1} of {questions.length}
          </span>
        </div>
        <span className="text-sm text-stone-500">
          {score.right}/{score.total} correct
        </span>
      </div>

      <div className="mt-2 h-1 w-full overflow-hidden rounded-full bg-stone-200">
        <div
          className="h-full bg-stone-900 transition-[width] duration-300"
          style={{ width: `${((index + 1) / questions.length) * 100}%` }}
        />
      </div>

      <Card className="mt-6">
        <h1 className="font-serif text-xl font-medium leading-snug text-stone-900">
          <Maths>{question.question}</Maths>
        </h1>
        {question.fileName && (
          <p className="mt-2 text-xs text-stone-400">from {question.fileName}</p>
        )}

        <ul className="mt-5 space-y-2">
          {question.answers.map((answer) => {
            const picked = chosen === answer.id;
            // After answering, the correct option is always highlighted, so a
            // wrong pick still shows what the right answer was.
            const reveal = answered && (picked || answer.correct);

            return (
              <li key={answer.id}>
                <button
                  onClick={() => choose(answer.id, answer.correct)}
                  disabled={answered}
                  className={cn(
                    "w-full rounded-lg px-4 py-3 text-left text-sm transition-colors",
                    "ring-1 disabled:cursor-default",
                    !answered && "bg-white ring-stone-300 hover:bg-stone-50",
                    answered && !reveal && "bg-white ring-stone-200 opacity-60",
                    reveal && answer.correct && "bg-emerald-50 ring-emerald-300",
                    reveal && !answer.correct && "bg-red-50 ring-red-300",
                  )}
                >
                  <span className="flex items-start gap-2">
                    {reveal &&
                      (answer.correct ? (
                        <Check
                          className="mt-0.5 size-4 shrink-0 text-emerald-700"
                          aria-hidden
                        />
                      ) : (
                        <X className="mt-0.5 size-4 shrink-0 text-red-700" aria-hidden />
                      ))}
                    <span className="flex-1">
                      <Maths>{answer.text}</Maths>
                    </span>
                  </span>

                  {reveal && answer.explanation && (
                    <span className="mt-2 block border-t border-black/5 pt-2 text-xs text-stone-600">
                      <Maths>{answer.explanation}</Maths>
                    </span>
                  )}
                </button>
              </li>
            );
          })}
        </ul>

        {answered && (
          <div className="mt-5 flex justify-end">
            <Button onClick={next}>
              {index + 1 >= questions.length ? "See result" : "Next question"}
            </Button>
          </div>
        )}
      </Card>
    </>
  );
}

function BackLink({ moduleId }: { moduleId: string }) {
  return (
    <Link
      href={`/modules/${moduleId}`}
      className="inline-flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-900"
    >
      <ArrowLeft className="size-4" aria-hidden />
      Back to module
    </Link>
  );
}

export default function QuizPage() {
  const params = useParams<{ moduleId: string }>();

  return (
    <SessionProvider>
      <AppShell>
        {/* useSearchParams needs a boundary, because the page is streamed. */}
        <Suspense fallback={<Spinner label="Loading your quiz" />}>
          <QuizRunner moduleId={params.moduleId} />
        </Suspense>
      </AppShell>
    </SessionProvider>
  );
}
