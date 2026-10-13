"use client";

import {
  ArrowLeft,
  BrainCircuit,
  FileText,
  GraduationCap,
  Layers,
  Trash2,
} from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import useSWR from "swr";
import { toast } from "sonner";

import { AppShell } from "@/components/AppShell";
import { FileDropzone } from "@/components/FileDropzone";
import { GenerateButton } from "@/components/GenerateButton";
import { SessionProvider } from "@/components/SessionProvider";
import {
  Button,
  ButtonLink,
  Card,
  EmptyState,
  ErrorState,
  Input,
  Pill,
  Select,
  Spinner,
} from "@/components/ui";
import { api, formatBytes } from "@/lib/api";
import type { Difficulty, Kind, LectureFile, TeachingStyle } from "@/lib/types";

/** What each kind of study material is called in the UI. */
const kindLabels: Record<Kind, string> = {
  quiz: "Quiz",
  flashcard: "Flashcards",
  teaching: "Teaching",
};

function ModuleView({ moduleId }: { moduleId: string }) {
  // Named mod rather than module, which shadows the CommonJS global.
  const mod = useSWR(["module", moduleId], () => api.module(moduleId));
  const files = useSWR(["files", moduleId], () => api.files(moduleId));
  const meta = useSWR("meta", api.meta);
  const usage = useSWR("usage", api.usage);

  const [difficulty, setDifficulty] = useState<Difficulty>("medium");
  const [style, setStyle] = useState<TeachingStyle>("simple-language");

  const quiz = useSWR(["quiz", moduleId, difficulty], () =>
    api.quiz(moduleId, difficulty),
  );
  const flashcards = useSWR(["flashcards", moduleId], () =>
    api.flashcards(moduleId),
  );
  const teaching = useSWR(["teaching", moduleId, style], () =>
    api.teaching(moduleId, style),
  );

  const [renaming, setRenaming] = useState(false);
  const [newName, setNewName] = useState("");

  const refreshAll = () =>
    Promise.all([
      mod.mutate(),
      files.mutate(),
      usage.mutate(),
      quiz.mutate(),
      flashcards.mutate(),
      teaching.mutate(),
    ]);

  const rename = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!newName.trim()) return;

    try {
      await api.updateModule(moduleId, newName.trim());
      setRenaming(false);
      await mod.mutate();
      toast.success("Renamed");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not rename it");
    }
  };

  const removeFile = async (file: LectureFile) => {
    const confirmed = window.confirm(
      `Delete "${file.originalName}"? Every question, flashcard and teaching card made from it is deleted too.`,
    );
    if (!confirmed) return;

    try {
      await api.deleteFile(file.id);
      await refreshAll();
      toast.success(`Deleted ${file.originalName}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not delete it");
    }
  };

  if (mod.error) {
    return (
      <ErrorState
        message={
          mod.error instanceof Error
            ? mod.error.message
            : "Could not load this module"
        }
      />
    );
  }
  if (!mod.data || !meta.data) {
    return <Spinner label="Loading module" />;
  }

  const hasFiles = (files.data?.length ?? 0) > 0;

  return (
    <>
      <Link
        href={`/subjects/${mod.data.subjectId}`}
        className="inline-flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-900"
      >
        <ArrowLeft className="size-4" aria-hidden />
        {mod.data.subjectName ?? "Back"}
      </Link>

      <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
        {renaming ? (
          <form onSubmit={rename} className="flex flex-1 items-center gap-2">
            <Input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              maxLength={120}
              autoFocus
              required
            />
            <Button type="submit" size="sm">
              Save
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setRenaming(false)}
            >
              Cancel
            </Button>
          </form>
        ) : (
          <>
            <h1 className="font-serif text-3xl font-semibold text-stone-900">
              {mod.data.name}
            </h1>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setNewName(mod.data!.name);
                setRenaming(true);
              }}
            >
              Rename
            </Button>
          </>
        )}
      </div>

      {/* ── Lecture files ─────────────────────────────────────────────── */}
      <section className="mt-8">
        <div className="flex items-end justify-between gap-4">
          <h2 className="font-medium text-stone-900">Lecture files</h2>
          {usage.data && (
            <p className="text-xs text-stone-500">
              {formatBytes(usage.data.storageUsed)} of{" "}
              {formatBytes(usage.data.storageQuota)} used
            </p>
          )}
        </div>

        <div className="mt-3 space-y-3">
          {files.isLoading && <Spinner label="Loading files" />}

          {files.data?.map((file) => (
            <Card key={file.id} className="flex items-center gap-3 py-3">
              <FileText className="size-4 shrink-0 text-stone-400" aria-hidden />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-stone-900">
                  {file.originalName}
                </p>
                <div className="mt-1 flex flex-wrap items-center gap-1.5">
                  <span className="text-xs text-stone-500">
                    {formatBytes(file.byteSize)}
                  </span>
                  {file.analysed.length === 0 ? (
                    <Pill tone="warn">not analysed</Pill>
                  ) : (
                    file.analysed.map((kind) => (
                      <Pill key={kind} tone="good">
                        {kindLabels[kind]}
                      </Pill>
                    ))
                  )}
                </div>
              </div>

              <ButtonLink
                variant="ghost"
                size="sm"
                href={api.downloadUrl(file.id)}
              >
                Download
              </ButtonLink>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => removeFile(file)}
                aria-label={`Delete ${file.originalName}`}
              >
                <Trash2 className="size-4" aria-hidden />
              </Button>
            </Card>
          ))}

          <FileDropzone
            moduleId={moduleId}
            accept={meta.data.supportedExtensions}
            maxFileSize={meta.data.maxFileSize}
            onUploaded={refreshAll}
          />
        </div>
      </section>

      {/* ── Study ─────────────────────────────────────────────────────── */}
      <section className="mt-10">
        <h2 className="font-medium text-stone-900">Study this module</h2>

        {!hasFiles ? (
          <div className="mt-3">
            <EmptyState
              icon={Layers}
              title="Upload a lecture first"
              description="Once this module has lecture files, you can quiz yourself, review flashcards, or have the material taught back to you."
            />
          </div>
        ) : (
          <div className="mt-3 grid gap-4 lg:grid-cols-3">
            {/* Quiz */}
            <Card className="flex flex-col">
              <div className="flex items-center gap-2">
                <BrainCircuit className="size-5 text-stone-900" aria-hidden />
                <h3 className="font-medium text-stone-900">Quiz me</h3>
              </div>
              <p className="mt-2 flex-1 text-sm text-stone-600">
                Multiple-choice questions with an explanation for every option.
              </p>

              <label
                htmlFor="difficulty"
                className="mt-4 mb-1.5 block text-xs font-medium text-stone-700"
              >
                Difficulty
              </label>
              <Select
                id="difficulty"
                value={difficulty}
                onChange={(e) => setDifficulty(e.target.value as Difficulty)}
              >
                {meta.data.difficulties.map((d) => (
                  <option key={d} value={d}>
                    {d}
                  </option>
                ))}
              </Select>

              <div className="mt-4">
                <GenerateButton
                  moduleId={moduleId}
                  kind="quiz"
                  variant={difficulty}
                  label="Generate"
                  filesPending={quiz.data?.filesPending ?? 0}
                  onFinished={refreshAll}
                />
              </div>

              <div className="mt-4 flex items-center justify-between border-t border-stone-200 pt-4">
                <span className="text-sm text-stone-600">
                  {quiz.data?.questions.length ?? 0} questions ready
                </span>
                {(quiz.data?.questions.length ?? 0) > 0 && (
                  <ButtonLink
                    variant="secondary"
                    size="sm"
                    href={`/modules/${moduleId}/quiz?difficulty=${difficulty}`}
                  >
                    Start
                  </ButtonLink>
                )}
              </div>
            </Card>

            {/* Flashcards */}
            <Card className="flex flex-col">
              <div className="flex items-center gap-2">
                <Layers className="size-5 text-stone-900" aria-hidden />
                <h3 className="font-medium text-stone-900">Flashcards</h3>
              </div>
              <p className="mt-2 flex-1 text-sm text-stone-600">
                Term on the front, definition on the back. No settings to pick.
              </p>

              <div className="mt-4">
                <GenerateButton
                  moduleId={moduleId}
                  kind="flashcard"
                  variant=""
                  label="Generate"
                  filesPending={flashcards.data?.filesPending ?? 0}
                  onFinished={refreshAll}
                />
              </div>

              <div className="mt-4 flex items-center justify-between border-t border-stone-200 pt-4">
                <span className="text-sm text-stone-600">
                  {flashcards.data?.flashcards.length ?? 0} cards ready
                </span>
                {(flashcards.data?.flashcards.length ?? 0) > 0 && (
                  <ButtonLink
                    variant="secondary"
                    size="sm"
                    href={`/modules/${moduleId}/flashcards`}
                  >
                    Review
                  </ButtonLink>
                )}
              </div>
            </Card>

            {/* Teaching */}
            <Card className="flex flex-col">
              <div className="flex items-center gap-2">
                <GraduationCap className="size-5 text-stone-900" aria-hidden />
                <h3 className="font-medium text-stone-900">Teach me</h3>
              </div>
              <p className="mt-2 flex-1 text-sm text-stone-600">
                The material explained from the ground up, in the style you pick.
              </p>

              <label
                htmlFor="style"
                className="mt-4 mb-1.5 block text-xs font-medium text-stone-700"
              >
                Style
              </label>
              <Select
                id="style"
                value={style}
                onChange={(e) => setStyle(e.target.value as TeachingStyle)}
              >
                {meta.data.teachingStyles.map((s) => (
                  <option key={s} value={s}>
                    {s.replace(/-/g, " ")}
                  </option>
                ))}
              </Select>

              <div className="mt-4">
                <GenerateButton
                  moduleId={moduleId}
                  kind="teaching"
                  variant={style}
                  label="Generate"
                  filesPending={teaching.data?.filesPending ?? 0}
                  onFinished={refreshAll}
                />
              </div>

              <div className="mt-4 flex items-center justify-between border-t border-stone-200 pt-4">
                <span className="text-sm text-stone-600">
                  {teaching.data?.teachingCards.length ?? 0} cards ready
                </span>
                {(teaching.data?.teachingCards.length ?? 0) > 0 && (
                  <ButtonLink
                    variant="secondary"
                    size="sm"
                    href={`/modules/${moduleId}/teach?style=${style}`}
                  >
                    Read
                  </ButtonLink>
                )}
              </div>
            </Card>
          </div>
        )}
      </section>
    </>
  );
}

export default function ModulePage() {
  const params = useParams<{ moduleId: string }>();

  return (
    <SessionProvider>
      <AppShell>
        <ModuleView moduleId={params.moduleId} />
      </AppShell>
    </SessionProvider>
  );
}
