"use client";

import { BookOpen } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { Suspense } from "react";

import { SignInButton } from "@/components/AppShell";
import { Spinner } from "@/components/ui";

/** Explains why a sign-in did not go through, in plain words. */
const reasons: Record<string, string> = {
  expired: "That sign-in took too long. Please try again.",
  failed: "Something went wrong signing you in. Please try again.",
  access_denied: "You cancelled the Google sign-in.",
};

function LoginForm() {
  const params = useSearchParams();
  const error = params.get("error");
  const redirectTo = params.get("redirect") ?? "/dashboard";

  return (
    <div className="w-full max-w-sm">
      <div className="flex items-center justify-center gap-2 font-serif text-xl font-semibold">
        <BookOpen className="size-5" aria-hidden />
        MemorisePDF
      </div>

      <h1 className="mt-8 text-center text-2xl font-semibold text-stone-900">
        Sign in
      </h1>
      <p className="mt-2 text-center text-sm text-stone-600">
        Your subjects, lectures and everything generated from them are tied to
        your Google account.
      </p>

      {error && (
        <p
          role="alert"
          className="mt-6 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-800 ring-1 ring-red-200"
        >
          {reasons[error] ?? reasons.failed}
        </p>
      )}

      <div className="mt-8">
        <SignInButton redirectTo={redirectTo} />
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <div className="flex min-h-dvh items-center justify-center px-4 py-16">
      {/* useSearchParams needs a boundary, because the page is streamed. */}
      <Suspense fallback={<Spinner label="Loading" />}>
        <LoginForm />
      </Suspense>
    </div>
  );
}
