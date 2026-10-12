import type {
  Difficulty,
  FlashcardPool,
  GenerateResult,
  Kind,
  LectureFile,
  Meta,
  Module,
  Progress,
  QuizPool,
  Subject,
  TeachingPool,
  TeachingStyle,
  UploadResult,
  Usage,
  User,
} from "./types";

/** Thrown for any non-2xx reply, carrying the message the API sent. */
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }

  /** True when the problem is simply that nobody is signed in. */
  get isUnauthenticated() {
    return this.status === 401;
  }
}

/** Sends a request and reads the reply, turning an error status into an ApiError. */
async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    // The session lives in a cookie, so it has to travel with every request.
    credentials: "include",
    ...init,
    headers: {
      Accept: "application/json",
      ...init.headers,
    },
  });

  if (!response.ok) {
    throw new ApiError(response.status, await errorMessage(response));
  }

  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

/** Pulls the message out of an error reply, falling back to the status text. */
async function errorMessage(response: Response): Promise<string> {
  try {
    const body = await response.json();
    if (body && typeof body.error === "string") {
      return body.error;
    }
  } catch {
    // A non-JSON error body is nothing to work with.
  }
  return response.statusText || `request failed with status ${response.status}`;
}

/** Sends a JSON body. */
function json(method: string, body: unknown): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

export const api = {
  /** Fetches whoever is signed in, or null when nobody is. */
  async me(): Promise<User | null> {
    try {
      return await request<User>("/api/me");
    } catch (err) {
      if (err instanceof ApiError && err.isUnauthenticated) return null;
      throw err;
    }
  },

  logout: () => request<void>("/auth/logout", { method: "POST" }),

  meta: () => request<Meta>("/api/meta"),
  usage: () => request<Usage>("/api/usage"),

  subjects: () => request<Subject[]>("/api/subjects"),
  subject: (id: string) => request<Subject>(`/api/subjects/${id}`),
  createSubject: (name: string, colour: string) =>
    request<Subject>("/api/subjects", json("POST", { name, colour })),
  updateSubject: (id: string, name: string, colour: string) =>
    request<Subject>(`/api/subjects/${id}`, json("PATCH", { name, colour })),
  deleteSubject: (id: string) =>
    request<{ status: string }>(`/api/subjects/${id}`, { method: "DELETE" }),

  modules: (subjectId: string) =>
    request<Module[]>(`/api/subjects/${subjectId}/modules`),
  module: (id: string) => request<Module>(`/api/modules/${id}`),
  createModule: (subjectId: string, name: string) =>
    request<Module>(`/api/subjects/${subjectId}/modules`, json("POST", { name })),
  updateModule: (id: string, name: string) =>
    request<Module>(`/api/modules/${id}`, json("PATCH", { name })),
  deleteModule: (id: string) =>
    request<{ status: string }>(`/api/modules/${id}`, { method: "DELETE" }),

  files: (moduleId: string) =>
    request<LectureFile[]>(`/api/modules/${moduleId}/files`),
  deleteFile: (id: string) =>
    request<{ status: string }>(`/api/files/${id}`, { method: "DELETE" }),
  downloadUrl: (id: string) => `/api/files/${id}/download`,

  /** Uploads lecture files to a module. */
  uploadFiles: (moduleId: string, files: File[]) => {
    const form = new FormData();
    for (const file of files) {
      form.append("files", file);
    }
    // Content-Type is left alone so the browser sets the multipart boundary.
    return request<UploadResult>(`/api/modules/${moduleId}/files`, {
      method: "POST",
      body: form,
    });
  },

  quiz: (moduleId: string, difficulty: Difficulty) =>
    request<QuizPool>(`/api/modules/${moduleId}/quiz?difficulty=${difficulty}`),
  flashcards: (moduleId: string) =>
    request<FlashcardPool>(`/api/modules/${moduleId}/flashcards`),
  teaching: (moduleId: string, style: TeachingStyle) =>
    request<TeachingPool>(`/api/modules/${moduleId}/teaching?style=${style}`),
};

/** Where to send somebody to sign in, coming back to where they were. */
export function signInUrl(redirectTo?: string) {
  const base = "/auth/google";
  return redirectTo ? `${base}?redirect=${encodeURIComponent(redirectTo)}` : base;
}

interface GenerateHandlers {
  onProgress?: (p: Progress) => void;
  onDone?: (r: GenerateResult) => void;
  signal?: AbortSignal;
}

/**
 * Asks the API to analyse a module's new files, reporting progress as it goes.
 *
 * The API answers a POST with an event stream, which EventSource cannot do
 * because it only issues GETs, so the stream is read off the fetch body.
 */
export async function generate(
  moduleId: string,
  kind: Kind,
  variant: string,
  handlers: GenerateHandlers = {},
): Promise<GenerateResult> {
  const response = await fetch(`/api/modules/${moduleId}/generate`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ kind, variant }),
    signal: handlers.signal,
  });

  if (!response.ok) {
    throw new ApiError(response.status, await errorMessage(response));
  }
  if (!response.body) {
    throw new Error("the browser gave no response body to read");
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let result: GenerateResult | null = null;
  let failure: string | null = null;

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });

    // Events are separated by a blank line, so anything after the last one is
    // a partial event and stays in the buffer until the rest arrives.
    const chunks = buffer.split("\n\n");
    buffer = chunks.pop() ?? "";

    for (const chunk of chunks) {
      const event = parseEvent(chunk);
      if (!event) continue;

      if (event.name === "progress") {
        handlers.onProgress?.(event.data as Progress);
      } else if (event.name === "done") {
        result = event.data as GenerateResult;
      } else if (event.name === "error") {
        failure = (event.data as { error?: string }).error ?? "generation failed";
      }
    }
  }

  if (failure) {
    throw new Error(failure);
  }
  if (!result) {
    throw new Error("the generation stream ended without a result");
  }

  handlers.onDone?.(result);
  return result;
}

/** Reads one "event:"/"data:" pair out of the stream. */
function parseEvent(chunk: string): { name: string; data: unknown } | null {
  let name = "message";
  const dataLines: string[] = [];

  for (const line of chunk.split("\n")) {
    if (line.startsWith("event:")) {
      name = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trim());
    }
  }

  if (dataLines.length === 0) return null;

  try {
    return { name, data: JSON.parse(dataLines.join("\n")) };
  } catch {
    return null;
  }
}

/** Turns a byte count into something readable like "2.4 MB". */
export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;

  const units = ["KB", "MB", "GB"];
  let value = bytes;
  let unit = "KB";

  for (const next of units) {
    value /= 1024;
    unit = next;
    if (value < 1024) break;
  }

  return `${value.toFixed(value < 10 ? 1 : 0)} ${unit}`;
}
