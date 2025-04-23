"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useResultStore } from "@/store/useResultStore";
import {
  ChevronLeft,
  ChevronRight,
  RotateCcw,
  Check,
  X,
  Layers,
  HelpCircle,
} from "lucide-react";
import { motion } from "framer-motion";

export default function ResultsPage() {
  const flashcards = useResultStore((state) => state.flashcards);
  const quizzes = useResultStore((state) => state.quizzes);
  const hasResults = useResultStore((state) => state.hasResults);

  // State for interactive elements
  const [currentFlashcard, setCurrentFlashcard] = useState(0);
  const [isFlipped, setIsFlipped] = useState(false);
  const [currentQuiz, setCurrentQuiz] = useState(0);
  const [selectedAnswer, setSelectedAnswer] = useState<number | null>(null);
  const [showAnswer, setShowAnswer] = useState(false);

  const router = useRouter();

  useEffect(() => {
    // If there are no results, redirect to upload page
    if (!hasResults()) {
      router.push("/upload");
    }
  }, [hasResults, router]);

  // Flashcard navigation
  const nextFlashcard = () => {
    if (currentFlashcard < flashcards.length - 1) {
      setCurrentFlashcard(currentFlashcard + 1);
      setIsFlipped(false);
    }
  };

  const prevFlashcard = () => {
    if (currentFlashcard > 0) {
      setCurrentFlashcard(currentFlashcard - 1);
      setIsFlipped(false);
    }
  };

  // Quiz navigation
  const nextQuiz = () => {
    if (currentQuiz < quizzes.length - 1) {
      setCurrentQuiz(currentQuiz + 1);
      setSelectedAnswer(null);
      setShowAnswer(false);
    }
  };

  const prevQuiz = () => {
    if (currentQuiz > 0) {
      setCurrentQuiz(currentQuiz - 1);
      setSelectedAnswer(null);
      setShowAnswer(false);
    }
  };

  // Answer selection
  const handleAnswerSelect = (index: number) => {
    setSelectedAnswer(index);
    setShowAnswer(true);
  };

  // If no results, show loading until redirect happens
  if (!hasResults()) {
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
      className="h-screen bg-amber-50 flex items-center justify-center"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.5 }}
    >
      <div className="container flex justify-center items-center h-[calc(100vh-200px)] px-4">
        <motion.div
          className="w-full max-w-5xl bg-white rounded-xl shadow-lg overflow-hidden"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
        >
          <div className="grid grid-cols-1 md:grid-cols-2 h-[500px] sm:h-[550px] md:h-[600px]">
            {/* Flashcards Section */}
            <div className="border-b md:border-b-0 md:border-r border-gray-200 p-4 sm:p-5 md:p-6 flex flex-col h-full overflow-hidden">
              <div className="flex items-center justify-between mb-3 md:mb-4 h-[40px]">
                <h2 className="text-lg sm:text-xl font-bold text-green-800 flex items-center">
                  <Layers className="w-4 h-4 sm:w-5 sm:h-5 mr-1.5 sm:mr-2" />
                  Flashcards
                </h2>
                <div className="text-xs sm:text-sm text-gray-500">
                  {currentFlashcard + 1} / {flashcards.length}
                </div>
              </div>

              {flashcards.length > 0 && (
                <div className="flex-grow flex flex-col items-center justify-center h-[calc(100%-110px)]">
                  <motion.div
                    className={`w-full h-48 sm:h-56 md:h-64 relative perspective-1000 cursor-pointer ${
                      isFlipped ? "flashcard-flipped" : ""
                    }`}
                    onClick={() => setIsFlipped(!isFlipped)}
                    whileHover={{ scale: 1.02 }}
                  >
                    <motion.div
                      className={`absolute w-full h-full backface-hidden transition-transform duration-500 ${
                        isFlipped ? "rotate-y-180 invisible" : ""
                      } bg-amber-50 rounded-lg flex items-center justify-center p-4 sm:p-6 border-2 border-amber-200`}
                      animate={{ rotateY: isFlipped ? 180 : 0 }}
                      transition={{ duration: 0.5 }}
                    >
                      <h3 className="text-base sm:text-lg md:text-xl font-semibold text-green-800 text-center">
                        {flashcards[currentFlashcard]?.term}
                      </h3>
                    </motion.div>
                    <motion.div
                      className={`absolute w-full h-full backface-hidden transition-transform duration-500 ${
                        isFlipped ? "" : "rotate-y-180 invisible"
                      } bg-green-50 rounded-lg flex items-center justify-center p-4 sm:p-6 border-2 border-green-200`}
                      animate={{ rotateY: isFlipped ? 0 : 180 }}
                      transition={{ duration: 0.5 }}
                    >
                      <p className="text-sm sm:text-base text-gray-700 text-center overflow-y-auto max-h-full">
                        {flashcards[currentFlashcard]?.definition}
                      </p>
                    </motion.div>
                  </motion.div>
                </div>
              )}

              <div className="mt-auto h-[50px] sm:h-[60px] flex items-center justify-between w-full border-t border-gray-100 pt-2">
                <motion.button
                  onClick={prevFlashcard}
                  disabled={currentFlashcard === 0}
                  className={`p-1.5 sm:p-2 rounded-full ${
                    currentFlashcard === 0
                      ? "text-gray-300"
                      : "text-green-800 hover:bg-green-50"
                  }`}
                  whileHover={currentFlashcard !== 0 ? { scale: 1.1 } : {}}
                  whileTap={currentFlashcard !== 0 ? { scale: 0.9 } : {}}
                >
                  <ChevronLeft className="w-5 h-5 sm:w-6 sm:h-6" />
                </motion.button>

                <motion.button
                  onClick={() => setIsFlipped(!isFlipped)}
                  className="px-3 sm:px-4 py-1.5 sm:py-2 bg-amber-100 text-green-800 rounded-full text-xs sm:text-sm flex items-center hover:bg-amber-200 transition"
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                >
                  <RotateCcw className="w-3 h-3 sm:w-4 sm:h-4 mr-1.5 sm:mr-2" />
                  Flip Card
                </motion.button>

                <motion.button
                  onClick={nextFlashcard}
                  disabled={currentFlashcard === flashcards.length - 1}
                  className={`p-1.5 sm:p-2 rounded-full ${
                    currentFlashcard === flashcards.length - 1
                      ? "text-gray-300"
                      : "text-green-800 hover:bg-green-50"
                  }`}
                  whileHover={
                    currentFlashcard !== flashcards.length - 1
                      ? { scale: 1.1 }
                      : {}
                  }
                  whileTap={
                    currentFlashcard !== flashcards.length - 1
                      ? { scale: 0.9 }
                      : {}
                  }
                >
                  <ChevronRight className="w-5 h-5 sm:w-6 sm:h-6" />
                </motion.button>
              </div>
            </div>

            {/* Quizzes Section */}
            <div className="p-4 sm:p-5 md:p-6 flex flex-col h-full overflow-hidden">
              <div className="flex items-center justify-between mb-3 md:mb-4 h-[40px]">
                <h2 className="text-lg sm:text-xl font-bold text-green-800 flex items-center">
                  <HelpCircle className="w-4 h-4 sm:w-5 sm:h-5 mr-1.5 sm:mr-2" />
                  Quiz
                </h2>
                <div className="text-xs sm:text-sm text-gray-500">
                  {currentQuiz + 1} / {quizzes.length}
                </div>
              </div>

              {quizzes.length > 0 && (
                <>
                  <div className="overflow-y-auto h-[calc(100%-110px)]">
                    <motion.div
                      className="bg-white rounded-lg p-3 sm:p-4 border border-gray-200"
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      key={currentQuiz} // Force animation on quiz change
                      transition={{ duration: 0.3 }}
                    >
                      <h3 className="font-semibold text-base sm:text-lg text-green-800 mb-3 sm:mb-4">
                        {quizzes[currentQuiz]?.question}
                      </h3>
                      <div className="space-y-2 sm:space-y-3">
                        {quizzes[currentQuiz]?.answers.map((answer, aIndex) => (
                          <motion.div
                            key={aIndex}
                            onClick={() => handleAnswerSelect(aIndex)}
                            className={`p-2 sm:p-3 rounded-md cursor-pointer transition-colors ${
                              selectedAnswer === aIndex
                                ? showAnswer
                                  ? answer.correct
                                    ? "bg-green-100 border border-green-300"
                                    : "bg-red-100 border border-red-300"
                                  : "bg-amber-100 border border-amber-300"
                                : "bg-gray-50 border border-gray-200 hover:bg-gray-100"
                            }`}
                            initial={{ opacity: 0, y: 10 }}
                            animate={{ opacity: 1, y: 0 }}
                            transition={{ delay: 0.1 * aIndex, duration: 0.3 }}
                            whileHover={{ scale: 1.02 }}
                          >
                            <div className="flex items-start">
                              <div
                                className={`w-5 h-5 sm:w-6 sm:h-6 rounded-full flex items-center justify-center mr-2 ${
                                  selectedAnswer === aIndex
                                    ? showAnswer
                                      ? answer.correct
                                        ? "bg-green-500 text-white"
                                        : "bg-red-500 text-white"
                                      : "bg-amber-500 text-white"
                                    : "bg-gray-200"
                                }`}
                              >
                                {showAnswer && selectedAnswer === aIndex ? (
                                  answer.correct ? (
                                    <Check className="w-3 h-3 sm:w-4 sm:h-4" />
                                  ) : (
                                    <X className="w-3 h-3 sm:w-4 sm:h-4" />
                                  )
                                ) : (
                                  aIndex + 1
                                )}
                              </div>
                              <div className="flex-1">
                                <p className="font-medium text-sm sm:text-base">
                                  {answer.text}
                                </p>
                                {showAnswer &&
                                  selectedAnswer === aIndex &&
                                  answer.explanation && (
                                    <motion.p
                                      className="text-xs sm:text-sm text-gray-600 mt-1.5 sm:mt-2 p-1.5 sm:p-2 bg-gray-50 rounded"
                                      initial={{ opacity: 0, height: 0 }}
                                      animate={{ opacity: 1, height: "auto" }}
                                      transition={{ duration: 0.3 }}
                                    >
                                      {answer.explanation}
                                    </motion.p>
                                  )}
                              </div>
                            </div>
                          </motion.div>
                        ))}
                      </div>
                    </motion.div>
                  </div>

                  <div className="mt-auto h-[50px] sm:h-[60px] flex items-center justify-between w-full border-t border-gray-100 pt-2">
                    <motion.button
                      onClick={prevQuiz}
                      disabled={currentQuiz === 0}
                      className={`p-1.5 sm:p-2 rounded-full ${
                        currentQuiz === 0
                          ? "text-gray-300"
                          : "text-green-800 hover:bg-green-50"
                      }`}
                      whileHover={currentQuiz !== 0 ? { scale: 1.1 } : {}}
                      whileTap={currentQuiz !== 0 ? { scale: 0.9 } : {}}
                    >
                      <ChevronLeft className="w-5 h-5 sm:w-6 sm:h-6" />
                    </motion.button>

                    {!showAnswer ? (
                      <motion.button
                        onClick={() => setShowAnswer(true)}
                        className="px-3 sm:px-4 py-1.5 sm:py-2 bg-green-100 text-green-800 rounded-full text-xs sm:text-sm hover:bg-green-200 transition"
                        disabled={selectedAnswer === null}
                        whileHover={
                          selectedAnswer !== null ? { scale: 1.05 } : {}
                        }
                        whileTap={
                          selectedAnswer !== null ? { scale: 0.95 } : {}
                        }
                      >
                        Check Answer
                      </motion.button>
                    ) : (
                      <motion.button
                        onClick={nextQuiz}
                        disabled={currentQuiz === quizzes.length - 1}
                        className="px-3 sm:px-4 py-1.5 sm:py-2 bg-green-100 text-green-800 rounded-full text-xs sm:text-sm hover:bg-green-200 transition"
                        whileHover={
                          currentQuiz !== quizzes.length - 1
                            ? { scale: 1.05 }
                            : {}
                        }
                        whileTap={
                          currentQuiz !== quizzes.length - 1
                            ? { scale: 0.95 }
                            : {}
                        }
                      >
                        Next Question
                      </motion.button>
                    )}

                    <motion.button
                      onClick={nextQuiz}
                      disabled={currentQuiz === quizzes.length - 1}
                      className={`p-1.5 sm:p-2 rounded-full ${
                        currentQuiz === quizzes.length - 1
                          ? "text-gray-300"
                          : "text-green-800 hover:bg-green-50"
                      }`}
                      whileHover={
                        currentQuiz !== quizzes.length - 1 ? { scale: 1.1 } : {}
                      }
                      whileTap={
                        currentQuiz !== quizzes.length - 1 ? { scale: 0.9 } : {}
                      }
                    >
                      <ChevronRight className="w-5 h-5 sm:w-6 sm:h-6" />
                    </motion.button>
                  </div>
                </>
              )}
            </div>
          </div>
        </motion.div>
      </div>
    </motion.main>
  );
}
