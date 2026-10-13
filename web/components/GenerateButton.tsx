"use client";

import { useRef, useState } from "react";
import { toast } from "sonner";

import { generate } from "@/lib/api";
import type { GenerateResult, Kind, Progress } from "@/lib/types";
import { Button } from "./ui";

/**
 * Runs a generation and shows what it is doing. The point of the progress line
 * is to make the no-duplicate rule visible: it says how many files are being
 * sent to the model and how many are being reused.
 */
export function GenerateButton({
  moduleId,
  kind,
  variant,
  label,
  filesPending,
  disabled,
  onFinished,
}: {
  moduleId: string;
  kind: Kind;
  variant: string;
  label: string;
  filesPending: number;
  disabled?: boolean;
  onFinished: (result: GenerateResult) => void | Promise<unknown>;
}) {
  const [running, setRunning] = useState(false);
  const [progress, setProgress] = useState<Progress | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const run = async () => {
    setRunning(true);
    setProgress(null);

    const controller = new AbortController();
    abortRef.current = controller;

    try {
      const result = await generate(moduleId, kind, variant, {
        onProgress: setProgress,
        signal: controller.signal,
      });

      if (result.generated === 0 && result.reused > 0) {
        toast.success(
          `Nothing new to generate — reusing ${result.reused} analysed ${
            result.reused === 1 ? "file" : "files"
          }`,
        );
      } else if (result.generated > 0) {
        toast.success(
          `Analysed ${result.generated} new ${
            result.generated === 1 ? "file" : "files"
          }${result.reused > 0 ? `, reused ${result.reused}` : ""}`,
        );
      }

      // Partial failures are surfaced rather than hidden behind the success.
      if (result.failed > 0) {
        toast.error(
          `${result.failed} ${result.failed === 1 ? "file" : "files"} failed`,
          { description: result.errors?.[0] },
        );
      }

      await onFinished(result);
    } catch (err) {
      if (controller.signal.aborted) return;
      toast.error(err instanceof Error ? err.message : "Generation failed");
    } finally {
      setRunning(false);
      setProgress(null);
      abortRef.current = null;
    }
  };

  const cancel = () => abortRef.current?.abort();

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Button onClick={run} loading={running} disabled={disabled}>
          {label}
        </Button>
        {running && (
          <Button variant="ghost" size="sm" onClick={cancel}>
            Cancel
          </Button>
        )}
        {!running && filesPending > 0 && (
          <span className="text-xs text-stone-500">
            {filesPending} new {filesPending === 1 ? "file" : "files"} to analyse
          </span>
        )}
        {!running && filesPending === 0 && (
          <span className="text-xs text-stone-500">
            everything is already analysed
          </span>
        )}
      </div>

      {progress && (
        <div
          role="status"
          aria-live="polite"
          className="rounded-lg bg-stone-100 px-3 py-2 text-xs text-stone-700"
        >
          <p>{progress.message}</p>
          {progress.total > 0 && (
            <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-stone-300">
              <div
                className="h-full bg-stone-900 transition-[width] duration-300"
                style={{
                  width: `${Math.round((progress.done / progress.total) * 100)}%`,
                }}
              />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
