import type { Metadata } from "next";
import { Inter, Lora } from "next/font/google";
import "./globals.css";
import { Toaster } from "sonner";
import Navbar from "@/components/Navbar";
import AnimationProvider from "@/components/AnimationProvider";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const lora = Lora({
  variable: "--font-lora",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "MemorisePDF - Convert Slides to Quizzes & Flashcards",
  description:
    "Transform your presentations and slides into interactive quizzes and flashcards for effective learning and retention",
  keywords: [
    "flashcards",
    "quizzes",
    "learning tools",
    "study aids",
    "slide conversion",
  ],
  openGraph: {
    title: "MemorisePDF - Convert Slides to Quizzes & Flashcards",
    description:
      "Transform your presentations and slides into interactive quizzes and flashcards for effective learning and retention",
    type: "website",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "MemorisePDF - Convert Slides to Quizzes & Flashcards",
    description:
      "Transform your presentations and slides into interactive quizzes and flashcards for effective learning and retention",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${inter.variable} ${lora.variable} antialiased text-gray-900`}
      >
        <Toaster />
        <Navbar />
        <AnimationProvider>{children}</AnimationProvider>
      </body>
    </html>
  );
}
