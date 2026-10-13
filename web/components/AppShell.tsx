"use client";

import { BookOpen, LogOut } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { toast } from "sonner";

import { api, signInUrl } from "@/lib/api";
import { useSession } from "./SessionProvider";
import { Button, ButtonLink, Spinner } from "./ui";

/**
 * The frame every signed-in page sits inside. Anybody not signed in is sent to
 * Google with the path they wanted, so they land back where they were going.
 */
export function AppShell({ children }: { children: React.ReactNode }) {
  const { user, loading, refresh } = useSession();
  const pathname = usePathname();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !user) {
      router.replace(`/login?redirect=${encodeURIComponent(pathname)}`);
    }
  }, [loading, user, pathname, router]);

  if (loading) {
    return <Spinner label="Checking your session" />;
  }
  if (!user) {
    // The redirect above is already on its way.
    return null;
  }

  const signOut = async () => {
    try {
      await api.logout();
      refresh();
      router.push("/");
    } catch {
      toast.error("Could not sign out");
    }
  };

  return (
    <div className="min-h-dvh">
      <header className="sticky top-0 z-10 border-b border-stone-200 bg-stone-50/80 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-4">
          <Link
            href="/dashboard"
            className="flex items-center gap-2 font-serif text-lg font-semibold text-stone-900"
          >
            <BookOpen className="size-5" aria-hidden />
            MemorisePDF
          </Link>

          <div className="flex items-center gap-3">
            <span className="hidden text-sm text-stone-500 sm:inline">
              {user.name || user.email}
            </span>
            <Button variant="ghost" size="sm" onClick={signOut}>
              <LogOut className="size-4" aria-hidden />
              Sign out
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-8">{children}</main>
    </div>
  );
}

/**
 * The sign-in link, used on the landing and login pages. It is a real anchor
 * rather than a button because signing in is a full page navigation to Google,
 * not something fetched in the background.
 */
export function SignInButton({
  redirectTo,
  children = "Continue with Google",
}: {
  redirectTo?: string;
  children?: React.ReactNode;
}) {
  return (
    <ButtonLink href={signInUrl(redirectTo)} size="lg" className="w-full sm:w-auto">
      {children}
    </ButtonLink>
  );
}
