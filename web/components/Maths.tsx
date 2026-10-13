"use client";

import katex from "katex";
import { useMemo } from "react";

/**
 * Renders text that may contain LaTeX between dollar signs, which the prompts
 * ask the model to use for maths, chemistry and Greek symbols.
 *
 * KaTeX is called directly rather than through a wrapper component, so there is
 * one less dependency to keep current and no untyped package to work around.
 */
export function Maths({ children }: { children: string }) {
  const parts = useMemo(() => split(children), [children]);

  return (
    <>
      {parts.map((part, i) =>
        part.maths ? (
          <span
            key={i}
            // KaTeX escapes its own input and returns markup it generated
            // itself, so there is no caller-supplied HTML in here.
            dangerouslySetInnerHTML={{ __html: render(part.text) }}
          />
        ) : (
          <span key={i}>{part.text}</span>
        ),
      )}
    </>
  );
}

interface Part {
  text: string;
  maths: boolean;
}

/** Splits text into plain runs and the maths between dollar signs. */
function split(text: string): Part[] {
  const chunks = text.split("$");

  // An odd number of delimiters leaves one unmatched, so the text is treated as
  // plain rather than rendering half an equation.
  if (chunks.length < 3 || chunks.length % 2 === 0) {
    return [{ text, maths: false }];
  }

  return chunks
    .map((chunk, i) => ({ text: chunk, maths: i % 2 === 1 }))
    .filter((part) => part.text !== "");
}

/** Turns one LaTeX expression into markup, or gives back the source if it will not parse. */
function render(expression: string): string {
  try {
    return katex.renderToString(expression, {
      throwOnError: false,
      displayMode: false,
      output: "html",
    });
  } catch {
    // A malformed expression should show as the text the model wrote rather
    // than blanking the question it belongs to.
    return escapeHtml(`$${expression}$`);
  }
}

/** Makes text safe to drop into markup. */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}
