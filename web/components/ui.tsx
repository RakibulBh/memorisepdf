"use client";

import { Loader2, type LucideIcon } from "lucide-react";
import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

/** Joins class names, letting a later Tailwind class win over an earlier one. */
export function cn(...parts: Array<string | false | null | undefined>) {
  return twMerge(clsx(parts));
}

type ButtonProps = React.ComponentProps<"button"> & {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  size?: "sm" | "md" | "lg";
  loading?: boolean;
};

const variants: Record<NonNullable<ButtonProps["variant"]>, string> = {
  primary:
    "bg-stone-900 text-white hover:bg-stone-800 disabled:hover:bg-stone-900",
  secondary:
    "bg-white text-stone-900 ring-1 ring-stone-300 hover:bg-stone-50 disabled:hover:bg-white",
  ghost: "text-stone-600 hover:bg-stone-100 hover:text-stone-900",
  danger:
    "bg-white text-red-700 ring-1 ring-red-200 hover:bg-red-50 disabled:hover:bg-white",
};

const sizes: Record<NonNullable<ButtonProps["size"]>, string> = {
  sm: "h-8 px-3 text-sm gap-1.5",
  md: "h-10 px-4 text-sm gap-2",
  lg: "h-12 px-6 text-base gap-2",
};

export function Button({
  variant = "primary",
  size = "md",
  loading = false,
  className,
  children,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      {...props}
      disabled={disabled || loading}
      className={cn(
        "inline-flex items-center justify-center rounded-lg font-medium transition-colors",
        "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-stone-900",
        "disabled:cursor-not-allowed disabled:opacity-50",
        variants[variant],
        sizes[size],
        className,
      )}
    >
      {loading && <Loader2 className="size-4 animate-spin" aria-hidden />}
      {children}
    </button>
  );
}

type ButtonLinkProps = React.ComponentProps<"a"> & {
  variant?: NonNullable<ButtonProps["variant"]>;
  size?: NonNullable<ButtonProps["size"]>;
};

/** A link that looks like a button, for navigation rather than an action. */
export function ButtonLink({
  variant = "primary",
  size = "md",
  className,
  children,
  ...props
}: ButtonLinkProps) {
  return (
    <a
      {...props}
      className={cn(
        "inline-flex items-center justify-center rounded-lg font-medium transition-colors",
        "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-stone-900",
        variants[variant],
        sizes[size],
        className,
      )}
    >
      {children}
    </a>
  );
}

export function Input({
  className,
  ...props
}: React.ComponentProps<"input">) {
  return (
    <input
      {...props}
      className={cn(
        "h-10 w-full rounded-lg bg-white px-3 text-sm text-stone-900 ring-1 ring-stone-300",
        "placeholder:text-stone-400",
        "focus:outline-2 focus:outline-offset-0 focus:outline-stone-900",
        className,
      )}
    />
  );
}

export function Select({
  className,
  ...props
}: React.ComponentProps<"select">) {
  return (
    <select
      {...props}
      className={cn(
        "h-10 w-full rounded-lg bg-white px-3 text-sm text-stone-900 ring-1 ring-stone-300",
        "focus:outline-2 focus:outline-offset-0 focus:outline-stone-900",
        className,
      )}
    />
  );
}

export function Card({
  className,
  children,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      {...props}
      className={cn(
        "rounded-xl bg-white p-5 ring-1 ring-stone-200",
        className,
      )}
    >
      {children}
    </div>
  );
}

/** A label-and-value pill, for counts shown next to a title. */
export function Pill({
  children,
  tone = "neutral",
}: {
  children: React.ReactNode;
  tone?: "neutral" | "good" | "warn";
}) {
  const tones = {
    neutral: "bg-stone-100 text-stone-600",
    good: "bg-emerald-50 text-emerald-700",
    warn: "bg-amber-50 text-amber-700",
  };
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        tones[tone],
      )}
    >
      {children}
    </span>
  );
}

/** Shown in place of a list that has nothing in it yet. */
export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
}: {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border border-dashed border-stone-300 bg-white/50 px-6 py-14 text-center">
      <Icon className="mx-auto size-8 text-stone-400" aria-hidden />
      <h3 className="mt-4 font-medium text-stone-900">{title}</h3>
      <p className="mx-auto mt-1 max-w-sm text-sm text-stone-500">{description}</p>
      {action && <div className="mt-6 flex justify-center">{action}</div>}
    </div>
  );
}

/** Shown while a page's data is still on its way. */
export function Spinner({ label = "Loading" }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-sm text-stone-500">
      <Loader2 className="size-4 animate-spin" aria-hidden />
      {label}
    </div>
  );
}

/** Shown when a page could not load its data. */
export function ErrorState({ message }: { message: string }) {
  return (
    <div className="rounded-xl bg-red-50 px-5 py-4 text-sm text-red-800 ring-1 ring-red-200">
      {message}
    </div>
  );
}
