import type { Metadata } from "next";
import { Inter, Lora } from "next/font/google";
import { Toaster } from "sonner";

import "./globals.css";
import "katex/dist/katex.min.css";

const inter = Inter({ variable: "--font-inter", subsets: ["latin"] });
const lora = Lora({ variable: "--font-lora", subsets: ["latin"] });

export const metadata: Metadata = {
  title: {
    default: "MemorisePDF — turn lecture slides into quizzes and flashcards",
    template: "%s · MemorisePDF",
  },
  description:
    "Organise your lectures into subjects and modules, then turn each module's slides into quizzes, flashcards and teaching cards.",
  keywords: ["flashcards", "quizzes", "revision", "lecture notes", "study tools"],
  openGraph: {
    title: "MemorisePDF",
    description:
      "Turn your lecture slides into quizzes, flashcards and teaching cards.",
    type: "website",
    locale: "en_GB",
  },
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en-GB">
      <body className={`${inter.variable} ${lora.variable} font-sans`}>
        <Toaster position="top-center" richColors />
        {children}
      </body>
    </html>
  );
}
