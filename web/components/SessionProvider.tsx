"use client";

import { createContext, useContext } from "react";
import useSWR from "swr";

import { api } from "@/lib/api";
import type { User } from "@/lib/types";

interface Session {
  user: User | null;
  loading: boolean;
  /** Re-reads who is signed in, after a sign-out for example. */
  refresh: () => void;
}

const SessionContext = createContext<Session>({
  user: null,
  loading: true,
  refresh: () => {},
});

/** Fetches who is signed in once and shares it with the whole tree below. */
export function SessionProvider({ children }: { children: React.ReactNode }) {
  const { data, isLoading, mutate } = useSWR("session", () => api.me(), {
    // The signed-in account rarely changes mid-visit, so this avoids a
    // request on every window focus.
    revalidateOnFocus: false,
    shouldRetryOnError: false,
  });

  return (
    <SessionContext.Provider
      value={{
        user: data ?? null,
        loading: isLoading,
        refresh: () => void mutate(),
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}

/** Reads the signed-in account from anywhere below the provider. */
export function useSession() {
  return useContext(SessionContext);
}
