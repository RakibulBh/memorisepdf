"use client";

import { ArrowLeft, ChevronRight, Layers, Plus, Trash2 } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import useSWR from "swr";
import { toast } from "sonner";

import { AppShell } from "@/components/AppShell";
import { SessionProvider } from "@/components/SessionProvider";
import {
  Button,
  Card,
  EmptyState,
  ErrorState,
  Input,
  Pill,
  Spinner,
} from "@/components/ui";
import { api, formatBytes } from "@/lib/api";
import type { Module } from "@/lib/types";

function Modules({ subjectId }: { subjectId: string }) {
  const subject = useSWR(["subject", subjectId], () => api.subject(subjectId));
  const modules = useSWR(["modules", subjectId], () => api.modules(subjectId));

  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);

  const create = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!name.trim()) return;

    setSaving(true);
    try {
      await api.createModule(subjectId, name.trim());
      setName("");
      await Promise.all([modules.mutate(), subject.mutate()]);
      toast.success("Module created");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not create it");
    } finally {
      setSaving(false);
    }
  };

  const remove = async (module: Module) => {
    const confirmed = window.confirm(
      `Delete "${module.name}"? Its lecture files and every quiz, flashcard and teaching card made from them are deleted too. This cannot be undone.`,
    );
    if (!confirmed) return;

    try {
      await api.deleteModule(module.id);
      await Promise.all([modules.mutate(), subject.mutate()]);
      toast.success(`Deleted ${module.name}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not delete it");
    }
  };

  if (subject.error) {
    return (
      <ErrorState
        message={
          subject.error instanceof Error
            ? subject.error.message
            : "Could not load this subject"
        }
      />
    );
  }

  return (
    <>
      <Link
        href="/dashboard"
        className="inline-flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-900"
      >
        <ArrowLeft className="size-4" aria-hidden />
        All subjects
      </Link>

      <div className="mt-4 flex items-center gap-3">
        {subject.data && (
          <span
            aria-hidden
            style={{ backgroundColor: subject.data.colour || "#78716c" }}
            className="size-8 shrink-0 rounded-lg"
          />
        )}
        <h1 className="font-serif text-3xl font-semibold text-stone-900">
          {subject.data?.name ?? "Loading…"}
        </h1>
      </div>
      <p className="mt-1 text-sm text-stone-600">
        Modules are the units you are taught. Lecture files go inside a module.
      </p>

      <Card className="mt-8">
        <form onSubmit={create} className="flex flex-col gap-4 sm:flex-row sm:items-end">
          <div className="flex-1">
            <label
              htmlFor="module-name"
              className="mb-1.5 block text-sm font-medium text-stone-700"
            >
              New module
            </label>
            <Input
              id="module-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Databases"
              maxLength={120}
              required
            />
          </div>
          <Button type="submit" loading={saving}>
            <Plus className="size-4" aria-hidden />
            Add
          </Button>
        </form>
      </Card>

      <div className="mt-6">
        {modules.isLoading && <Spinner label="Loading modules" />}

        {modules.error && (
          <ErrorState
            message={
              modules.error instanceof Error
                ? modules.error.message
                : "Could not load modules"
            }
          />
        )}

        {modules.data && modules.data.length === 0 && (
          <EmptyState
            icon={Layers}
            title="No modules yet"
            description="Add a module above, then upload that module's lecture slides."
          />
        )}

        {modules.data && modules.data.length > 0 && (
          <ul className="space-y-3">
            {modules.data.map((module) => (
              <li key={module.id}>
                <Card className="flex items-center gap-4 transition-shadow hover:shadow-sm">
                  <Link href={`/modules/${module.id}`} className="min-w-0 flex-1">
                    <p className="truncate font-medium text-stone-900">
                      {module.name}
                    </p>
                    <div className="mt-1.5 flex flex-wrap items-center gap-2">
                      <Pill>
                        {module.fileCount}{" "}
                        {module.fileCount === 1 ? "lecture" : "lectures"}
                      </Pill>
                      {module.totalBytes > 0 && (
                        <span className="text-xs text-stone-500">
                          {formatBytes(module.totalBytes)}
                        </span>
                      )}
                    </div>
                  </Link>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => remove(module)}
                    aria-label={`Delete ${module.name}`}
                  >
                    <Trash2 className="size-4" aria-hidden />
                  </Button>
                  <Link
                    href={`/modules/${module.id}`}
                    aria-label={`Open ${module.name}`}
                    className="text-stone-400 hover:text-stone-900"
                  >
                    <ChevronRight className="size-5" aria-hidden />
                  </Link>
                </Card>
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}

export default function SubjectPage() {
  // Read from the hook rather than the page props, because this is a client
  // component and params is a promise in the server signature.
  const params = useParams<{ subjectId: string }>();

  return (
    <SessionProvider>
      <AppShell>
        <Modules subjectId={params.subjectId} />
      </AppShell>
    </SessionProvider>
  );
}
