"use client";

import { ChevronRight, FolderTree, Plus, Trash2 } from "lucide-react";
import Link from "next/link";
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
import { api } from "@/lib/api";
import type { Subject } from "@/lib/types";

/** The colours a subject can be tagged with, to tell them apart at a glance. */
const colours = [
  "#2563eb",
  "#059669",
  "#d97706",
  "#dc2626",
  "#7c3aed",
  "#0891b2",
];

function Subjects() {
  const { data, error, isLoading, mutate } = useSWR("subjects", api.subjects);
  const [name, setName] = useState("");
  const [colour, setColour] = useState(colours[0]!);
  const [saving, setSaving] = useState(false);

  const create = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!name.trim()) return;

    setSaving(true);
    try {
      await api.createSubject(name.trim(), colour);
      setName("");
      await mutate();
      toast.success("Subject created");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not create it");
    } finally {
      setSaving(false);
    }
  };

  const remove = async (subject: Subject) => {
    const confirmed = window.confirm(
      `Delete "${subject.name}"? Its modules, lecture files and everything generated from them are deleted too. This cannot be undone.`,
    );
    if (!confirmed) return;

    try {
      await api.deleteSubject(subject.id);
      await mutate();
      toast.success(`Deleted ${subject.name}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not delete it");
    }
  };

  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="font-serif text-3xl font-semibold text-stone-900">
            Your subjects
          </h1>
          <p className="mt-1 text-sm text-stone-600">
            A subject is a course. Modules live inside it.
          </p>
        </div>
      </div>

      <Card className="mt-8">
        <form onSubmit={create} className="flex flex-col gap-4 sm:flex-row sm:items-end">
          <div className="flex-1">
            <label
              htmlFor="subject-name"
              className="mb-1.5 block text-sm font-medium text-stone-700"
            >
              New subject
            </label>
            <Input
              id="subject-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Computer Science"
              maxLength={120}
              required
            />
          </div>

          <fieldset className="flex items-end gap-2">
            <legend className="sr-only">Colour</legend>
            {colours.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setColour(c)}
                aria-label={`Use colour ${c}`}
                aria-pressed={colour === c}
                style={{ backgroundColor: c }}
                className={`size-7 rounded-full transition-transform ${
                  colour === c
                    ? "ring-2 ring-stone-900 ring-offset-2"
                    : "hover:scale-110"
                }`}
              />
            ))}
          </fieldset>

          <Button type="submit" loading={saving}>
            <Plus className="size-4" aria-hidden />
            Add
          </Button>
        </form>
      </Card>

      <div className="mt-6">
        {isLoading && <Spinner label="Loading your subjects" />}

        {error && (
          <ErrorState
            message={
              error instanceof Error ? error.message : "Could not load subjects"
            }
          />
        )}

        {data && data.length === 0 && (
          <EmptyState
            icon={FolderTree}
            title="No subjects yet"
            description="Create your first subject above, then add the modules you are taught in it."
          />
        )}

        {data && data.length > 0 && (
          <ul className="grid gap-3 sm:grid-cols-2">
            {data.map((subject) => (
              <li key={subject.id}>
                <Card className="flex items-center gap-4 transition-shadow hover:shadow-sm">
                  <span
                    aria-hidden
                    style={{ backgroundColor: subject.colour || "#78716c" }}
                    className="size-10 shrink-0 rounded-lg"
                  />

                  <Link
                    href={`/subjects/${subject.id}`}
                    className="min-w-0 flex-1"
                  >
                    <p className="truncate font-medium text-stone-900">
                      {subject.name}
                    </p>
                    <p className="mt-1 text-sm text-stone-500">
                      <Pill>
                        {subject.moduleCount}{" "}
                        {subject.moduleCount === 1 ? "module" : "modules"}
                      </Pill>
                    </p>
                  </Link>

                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => remove(subject)}
                    aria-label={`Delete ${subject.name}`}
                  >
                    <Trash2 className="size-4" aria-hidden />
                  </Button>
                  <Link
                    href={`/subjects/${subject.id}`}
                    aria-label={`Open ${subject.name}`}
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

export default function DashboardPage() {
  return (
    <SessionProvider>
      <AppShell>
        <Subjects />
      </AppShell>
    </SessionProvider>
  );
}
