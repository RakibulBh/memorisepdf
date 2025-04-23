"use client";

import { AnimatePresence } from "framer-motion";
import { ReactNode } from "react";

interface AnimationProviderProps {
  children: ReactNode;
  mode?: "sync" | "popLayout" | "wait";
}

export default function AnimationProvider({
  children,
  mode = "wait",
}: AnimationProviderProps) {
  return <AnimatePresence mode={mode}>{children}</AnimatePresence>;
}
