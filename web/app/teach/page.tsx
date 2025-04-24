"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useResultStore } from "@/store/useResultStore";
import { BookOpen, ArrowLeft, ArrowRight } from "lucide-react";
import { motion } from "framer-motion";
import "katex/dist/katex.min.css";
import { InlineMath } from "react-katex";

// Helper function to safely render text that might contain LaTeX
const renderWithLatex = (text: string) => {
  if (!text) return null;

  // Check if text contains any LaTeX notation (enclosed in $ signs)
  if (!text.includes("$")) return text;

  const segments = text.split(/(\$.*?\$)/g);
  return segments.map((segment, index) => {
    // If segment starts and ends with $, it's LaTeX
    if (segment.startsWith("$") && segment.endsWith("$")) {
      const latex = segment.slice(1, -1); // Remove $ signs
      try {
        return <InlineMath key={index} math={latex} />;
      } catch (error) {
        console.error("Error rendering LaTeX:", error);
        return segment; // Fallback to raw text if rendering fails
      }
    }
    // Regular text
    return <span key={index}>{segment}</span>;
  });
};

export default function TeachPage() {
  const teachingCards = useResultStore((state) => state.teachingCards);
  const hasTeachingContent = useResultStore(
    (state) => state.hasTeachingContent
  );
  const contentType = useResultStore((state) => state.contentType);

  // State for interactive elements
  const [currentCard, setCurrentCard] = useState(0);

  const router = useRouter();

  useEffect(() => {
    // If there's no teaching content or the content type isn't "teach", redirect to upload page
    if (!hasTeachingContent() || contentType !== "teach") {
      router.push("/upload");
    }
  }, [hasTeachingContent, contentType, router]);

  // Card navigation
  const nextCard = () => {
    if (currentCard < teachingCards.length - 1) {
      setCurrentCard(currentCard + 1);
    }
  };

  const prevCard = () => {
    if (currentCard > 0) {
      setCurrentCard(currentCard - 1);
    }
  };

  // If no teaching content, show loading until redirect happens
  if (!hasTeachingContent() || contentType !== "teach") {
    return (
      <motion.div
        className="min-h-screen bg-amber-50 flex items-center justify-center"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
      >
        <p>Redirecting to upload page...</p>
      </motion.div>
    );
  }

  return (
    <motion.main
      className="min-h-screen bg-amber-50 flex items-center justify-center py-8 md:py-12"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.5 }}
    >
      <div className="container flex justify-center items-center px-4 max-w-4xl mx-auto">
        <div className="w-full">
          {/* Subtle header with navigation controls */}
          <div className="flex items-center justify-between mb-4 px-2">
            <div className="flex items-center text-sm text-gray-500">
              <BookOpen className="w-4 h-4 mr-1.5 text-green-700 opacity-80" />
              <span className="font-medium">Lesson {currentCard + 1}</span>
              <span className="mx-1.5 text-gray-400">/</span>
              <span>{teachingCards.length}</span>
            </div>
          </div>

          {/* Card Content */}
          <motion.div
            key={currentCard}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.4 }}
            className="mb-5"
          >
            <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">
              <div className="bg-green-800 px-6 py-4 text-white">
                <h2 className="font-medium text-lg">
                  {teachingCards[currentCard].subtopic}
                </h2>
              </div>
              <div className="p-5 md:p-6 lg:p-7">
                <div className="prose prose-green prose-sm sm:prose max-w-none">
                  <p className="text-gray-700">
                    {renderWithLatex(teachingCards[currentCard].teaching)}
                  </p>
                </div>
              </div>
            </div>
          </motion.div>

          {/* Navigation */}
          <div className="flex items-center justify-between mt-6">
            <motion.button
              onClick={prevCard}
              disabled={currentCard === 0}
              className={`group flex items-center rounded-lg px-4 py-2 border ${
                currentCard === 0
                  ? "text-gray-300 border-gray-200 cursor-not-allowed"
                  : "text-green-800 border-green-800 hover:border-green-300 hover:bg-green-50"
              } transition-colors`}
              whileHover={currentCard !== 0 ? { scale: 1.02 } : {}}
              whileTap={currentCard !== 0 ? { scale: 0.98 } : {}}
            >
              <ArrowLeft
                className={`w-4 h-4 mr-2 ${
                  currentCard === 0
                    ? ""
                    : "group-hover:-translate-x-1 transition-transform"
                }`}
              />
              <span className="text-sm font-medium">Previous</span>
            </motion.button>

            <div className="flex space-x-1.5">
              {teachingCards.map((_, index) => (
                <button
                  key={index}
                  onClick={() => setCurrentCard(index)}
                  className={`w-1.5 h-1.5 rounded-full transition-colors ${
                    currentCard === index ? "bg-green-800" : "bg-gray-300"
                  }`}
                  aria-label={`Go to slide ${index + 1}`}
                />
              ))}
            </div>

            <motion.button
              onClick={nextCard}
              disabled={currentCard === teachingCards.length - 1}
              className={`group flex items-center rounded-lg px-4 py-2 border ${
                currentCard === teachingCards.length - 1
                  ? "text-gray-300 border-gray-200 cursor-not-allowed"
                  : "text-green-800 border-green-800 hover:border-green-300 hover:bg-green-50"
              } transition-colors`}
              whileHover={
                currentCard !== teachingCards.length - 1 ? { scale: 1.02 } : {}
              }
              whileTap={
                currentCard !== teachingCards.length - 1 ? { scale: 0.98 } : {}
              }
            >
              <span className="text-sm font-medium">Next</span>
              <ArrowRight
                className={`w-4 h-4 ml-2 ${
                  currentCard === teachingCards.length - 1
                    ? ""
                    : "group-hover:translate-x-1 transition-transform"
                }`}
              />
            </motion.button>
          </div>
        </div>
      </div>
    </motion.main>
  );
}
