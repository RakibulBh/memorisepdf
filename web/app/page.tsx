import { BookOpen, FolderTree, Layers, Sparkles } from "lucide-react";

import { SignInButton } from "@/components/AppShell";
import { SessionProvider } from "@/components/SessionProvider";

const steps = [
  {
    icon: FolderTree,
    title: "Organise by subject",
    body: "Create a subject for each course, and a module inside it for each unit you are taught.",
  },
  {
    icon: Layers,
    title: "Upload your lectures",
    body: "Drop a module's slides in as PDF, PowerPoint or Word. Scanned decks are read too.",
  },
  {
    icon: Sparkles,
    title: "Study it",
    body: "Quiz yourself, review flashcards, or have the material taught back to you.",
  },
];

export default function Home() {
  return (
    <SessionProvider>
      <div className="min-h-dvh">
        <header className="mx-auto flex h-16 max-w-5xl items-center justify-between px-4">
          <span className="flex items-center gap-2 font-serif text-lg font-semibold">
            <BookOpen className="size-5" aria-hidden />
            Unmemorize
          </span>
        </header>

        <main className="mx-auto max-w-5xl px-4">
          <section className="py-16 sm:py-24">
            <h1 className="max-w-2xl font-serif text-4xl font-semibold tracking-tight text-stone-900 sm:text-5xl">
              Your lecture slides, turned into something you can actually revise
              from.
            </h1>
            <p className="mt-5 max-w-xl text-lg text-stone-600">
              Keep your course organised by subject and module, upload each
              module&rsquo;s lectures, and generate quizzes, flashcards and
              teaching cards from them. Everything you generate is saved, so
              revisiting a module costs nothing.
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row">
              <SignInButton redirectTo="/dashboard" />
            </div>
          </section>

          <section className="grid gap-4 pb-20 sm:grid-cols-3">
            {steps.map(({ icon: Icon, title, body }) => (
              <div
                key={title}
                className="rounded-xl bg-white p-6 ring-1 ring-stone-200"
              >
                <Icon className="size-5 text-stone-900" aria-hidden />
                <h2 className="mt-4 font-medium text-stone-900">{title}</h2>
                <p className="mt-2 text-sm leading-relaxed text-stone-600">
                  {body}
                </p>
              </div>
            ))}
          </section>
        </main>

        <footer className="border-t border-stone-200 py-8">
          <p className="mx-auto max-w-5xl px-4 text-sm text-stone-500">
            Unmemorize — self-hosted.
          </p>
        </footer>
      </div>
    </SessionProvider>
  );
}
