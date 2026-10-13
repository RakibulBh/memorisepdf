"use client";

import { Upload } from "lucide-react";
import { useRef, useState } from "react";
import { toast } from "sonner";

import { api, formatBytes } from "@/lib/api";
import { Button, cn } from "./ui";

/** Lets somebody add lecture files to a module, by dropping or choosing them. */
export function FileDropzone({
  moduleId,
  accept,
  maxFileSize,
  onUploaded,
}: {
  moduleId: string;
  accept: string[];
  maxFileSize: number;
  onUploaded: () => void | Promise<unknown>;
}) {
  const [dragging, setDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const send = async (files: File[]) => {
    if (files.length === 0) return;

    setUploading(true);
    try {
      const result = await api.uploadFiles(moduleId, files);

      if (result.stored > 0) {
        toast.success(
          `Added ${result.stored} ${result.stored === 1 ? "lecture" : "lectures"}`,
        );
      }

      // Each file gets its own verdict, so a mixed batch is reported honestly
      // rather than as one overall success or failure.
      for (const outcome of result.results) {
        if (outcome.status === "skipped") {
          toast.info(`${outcome.name}: ${outcome.reason}`);
        } else if (outcome.status === "rejected") {
          toast.error(`${outcome.name}: ${outcome.reason}`);
        }
      }

      await onUploaded();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setUploading(false);
      // Cleared so choosing the same file again still fires a change event.
      if (inputRef.current) inputRef.current.value = "";
    }
  };

  return (
    <div
      onDragOver={(e) => {
        e.preventDefault();
        setDragging(true);
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={(e) => {
        e.preventDefault();
        setDragging(false);
        void send(Array.from(e.dataTransfer.files));
      }}
      className={cn(
        "rounded-xl border-2 border-dashed px-6 py-10 text-center transition-colors",
        dragging
          ? "border-stone-900 bg-stone-100"
          : "border-stone-300 bg-white/50",
      )}
    >
      <Upload className="mx-auto size-7 text-stone-400" aria-hidden />
      <p className="mt-4 text-sm font-medium text-stone-900">
        Drop lecture files here
      </p>
      <p className="mx-auto mt-1 max-w-md text-xs text-stone-500">
        {accept.join(", ")} · up to {formatBytes(maxFileSize)} each
      </p>

      <input
        ref={inputRef}
        type="file"
        multiple
        accept={accept.join(",")}
        className="hidden"
        onChange={(e) => void send(Array.from(e.target.files ?? []))}
      />

      <Button
        variant="secondary"
        size="sm"
        className="mt-5"
        loading={uploading}
        onClick={() => inputRef.current?.click()}
      >
        Choose files
      </Button>
    </div>
  );
}
