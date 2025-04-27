"use client";

import React, { useState, useEffect, useCallback, useRef } from "react";
import {
  Upload,
  Check,
  X,
  Loader2,
  Brain,
  Lightbulb,
  School,
  Blocks,
  BarChart2,
  BookOpen,
} from "lucide-react";
import { toast } from "sonner";
import initiateProcessing from "@/services/requests/parse-presentation";
import { useResultStore } from "@/store/useResultStore";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";

export default function UploadPage() {
  const [file, setFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [sse, setSse] = useState<EventSource | null>(null);
  const [difficulty, setDifficulty] = useState<"easy" | "medium" | "hard">(
    "easy"
  );
  const [serviceType, setServiceType] = useState<"testme" | "teachme">(
    "testme"
  );
  const [teachingStyle, setTeachingStyle] = useState<
    "simple-language" | "analogy-driven" | "scaffolded-learning"
  >("simple-language");
  const [progressMessages, setProgressMessages] = useState<string[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<
    "connecting" | "connected" | "error" | null
  >(null);
  const setResults = useResultStore((state) => state.setResults);
  const setTeachingCards = useResultStore((state) => state.setTeachingCards);
  const router = useRouter();

  // Use refs to avoid circular dependencies
  const handleProgressRef = useRef<(event: MessageEvent) => void>(() => {});
  const handleFinishedRef = useRef<(event: MessageEvent) => void>(() => {});
  const handleErrorRef = useRef<(event: MessageEvent | Event) => void>(
    () => {}
  );

  const cleanup = useCallback(() => {
    if (sse) {
      sse.removeEventListener("progress", handleProgressRef.current);
      sse.removeEventListener("finished", handleFinishedRef.current);
      sse.removeEventListener("error", handleErrorRef.current);
      sse.close();
      setSse(null);
    }
    setUploading(false);
    setConnectionStatus(null);
  }, [sse]);

  // Set up the event handlers with refs to avoid circular deps
  handleProgressRef.current = (event: MessageEvent) => {
    // Don't display the connection established message to the user
    if (event.data === "Connection established") {
      setConnectionStatus("connected");
      return;
    }

    toast.info(event.data);
    setProgressMessages((prev) => [...prev, event.data]);
  };

  handleFinishedRef.current = (event: MessageEvent) => {
    try {
      const decodedData = atob(event.data);
      const resultData = JSON.parse(decodedData);

      if (serviceType === "testme") {
        setResults({
          flashcards: resultData.flashcards || [],
          quizzes: resultData.quizzes || [],
        });
        toast.success("Successfully generated flashcards and quizzes!");
        // Clean up before navigation
        cleanup();
        // Navigate to results page
        router.push("/results");
      } else {
        setTeachingCards(resultData.teaching_cards || []);
        toast.success("Successfully generated teaching content!");
        // Clean up before navigation
        cleanup();
        // Navigate to teaching page
        router.push("/teach");
      }
    } catch (error) {
      console.error("Failed to parse result data:", error);
      toast.error("Failed to process presentation results");
      cleanup();
    }
  };

  handleErrorRef.current = (event: MessageEvent | Event) => {
    setConnectionStatus("error");

    // Handle both standard error events and SSE error events
    if (event instanceof MessageEvent) {
      console.error("SSE message error:", event.data);
      toast.error(event.data || "An error occurred during processing");
    } else {
      console.error("SSE connection error:", event);
      toast.error("Connection error: Please try again");
    }

    cleanup();
  };

  useEffect(() => {
    if (!sse) return;

    // Set connection status to connecting
    setConnectionStatus("connecting");

    // Add event listeners using the current refs
    sse.addEventListener("progress", handleProgressRef.current);
    sse.addEventListener("finished", handleFinishedRef.current);
    sse.addEventListener("error", handleErrorRef.current);

    return () => {
      cleanup();
    };
  }, [sse, cleanup]);

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      setFile(e.dataTransfer.files[0]);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      setFile(e.target.files[0]);
    }
  };

  const handleUpload = async () => {
    if (!file) return;

    // Check file size client-side before uploading
    const maxSizeBytes = 75 * 1024 * 1024; // 75MB (reduced from 500MB)

    if (file.size > maxSizeBytes) {
      toast.error(
        `File too large (${(file.size / (1024 * 1024)).toFixed(
          2
        )}MB). Maximum size is 75MB.`
      );
      return;
    }

    setUploading(true);
    setProgressMessages([]);
    setConnectionStatus(null);

    try {
      toast.info("Uploading file...");

      const formData = new FormData();
      formData.append("file", file);
      formData.append("service_type", serviceType);
      formData.append("difficulty", difficulty);
      formData.append("teaching_style", teachingStyle);

      const response = await initiateProcessing(formData);

      if (response.error) {
        toast.error(response.error);
        setUploading(false);
        return;
      }
      const taskID = response.data.task_id;

      if (sse) {
        cleanup();
      }

      // begin the SSE with the taskID received
      const sseURL = new URL(`${process.env.NEXT_PUBLIC_API_URL}/sse`);
      sseURL.searchParams.append("task_id", taskID);

      // create event source with proper error handling
      const eventSource = new EventSource(sseURL.toString());

      // Add a timeout for initial connection
      const connectionTimeout = setTimeout(() => {
        if (connectionStatus === "connecting") {
          toast.error("Server connection timed out. Please try again.");
          eventSource.close();
          setUploading(false);
          setConnectionStatus("error");
        }
      }, 15000); // 15 seconds timeout

      setSse(eventSource);

      // Clear the timeout once connected
      if (connectionStatus === "connected") {
        clearTimeout(connectionTimeout);
      }

      toast.info("Processing started...");
      setProgressMessages((prev) => [...prev, "Processing started..."]);
    } catch (processingError) {
      // If any error occurs during processing, handle it
      console.error("Processing error:", processingError);
      toast.error(
        processingError instanceof Error
          ? processingError.message
          : "Failed to process file"
      );
      setUploading(false);
      setConnectionStatus("error");
    }
  };

  const resetFile = () => {
    setFile(null);
  };

  const ServiceTypeSelector = () => {
    return (
      <div className="w-full mb-4">
        <h3 className="text-sm font-semibold text-gray-700 mb-2">
          What would you like to do?
        </h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
          <motion.button
            onClick={() => setServiceType("testme")}
            className={`py-3 px-4 rounded-lg border text-sm transition-all flex items-center ${
              serviceType === "testme"
                ? "border-green-800 bg-green-50 text-green-800"
                : "border-gray-300 hover:border-green-300 hover:bg-green-50/50"
            }`}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
          >
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center mr-3 ${
                serviceType === "testme" ? "bg-green-100" : "bg-gray-100"
              }`}
            >
              <Brain
                className={`w-4 h-4 ${
                  serviceType === "testme" ? "text-green-800" : "text-gray-500"
                }`}
              />
            </div>
            <div className="flex-1 text-left">
              <span className="font-medium block">Test Me</span>
              <span className="text-xs text-gray-500 mt-1 block">
                Generate flashcards and quizzes
              </span>
            </div>
            {serviceType === "testme" && (
              <div className="w-3 h-3 rounded-full bg-green-800 ml-2"></div>
            )}
          </motion.button>

          <motion.button
            onClick={() => setServiceType("teachme")}
            className={`py-3 px-4 rounded-lg border text-sm transition-all flex items-center ${
              serviceType === "teachme"
                ? "border-green-800 bg-green-50 text-green-800"
                : "border-gray-300 hover:border-green-300 hover:bg-green-50/50"
            }`}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
          >
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center mr-3 ${
                serviceType === "teachme" ? "bg-green-100" : "bg-gray-100"
              }`}
            >
              <Lightbulb
                className={`w-4 h-4 ${
                  serviceType === "teachme" ? "text-green-800" : "text-gray-500"
                }`}
              />
            </div>
            <div className="flex-1 text-left">
              <span className="font-medium block">Teach Me</span>
              <span className="text-xs text-gray-500 mt-1 block">
                Create step-by-step explanations
              </span>
            </div>
            {serviceType === "teachme" && (
              <div className="w-3 h-3 rounded-full bg-green-800 ml-2"></div>
            )}
          </motion.button>
        </div>
      </div>
    );
  };

  const DifficultySelector = () => {
    const difficultyInfo = {
      easy: "Tests basic recall and explicit information",
      medium: "Requires understanding and basic inference",
      hard: "Needs deeper analysis and complex application",
    };

    return (
      <div className="w-full mb-4">
        <h3 className="text-sm font-semibold text-gray-700 mb-2">
          Select difficulty:
        </h3>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
          {(["easy", "medium", "hard"] as const).map((level) => (
            <motion.button
              key={level}
              onClick={() => setDifficulty(level)}
              className={`py-2 px-3 rounded-lg border text-sm transition-all ${
                difficulty === level
                  ? "border-green-800 bg-green-50 text-green-800"
                  : "border-gray-300 hover:border-green-300 hover:bg-green-50/50"
              }`}
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
            >
              <div className="flex items-center justify-between">
                <span className="capitalize font-medium">{level}</span>
                {difficulty === level && (
                  <div className="w-3 h-3 rounded-full bg-green-800 ml-2"></div>
                )}
              </div>
              <p className="text-xs text-gray-500 mt-1 text-left">
                {difficultyInfo[level]}
              </p>
            </motion.button>
          ))}
        </div>
      </div>
    );
  };

  const TeachingStyleSelector = () => {
    const teachingStyleInfo = {
      "simple-language":
        "Explain Like I'm 5 - Simple terms anyone can understand",
      "analogy-driven":
        "Analogy Driven - Using familiar examples to explain concepts",
      "scaffolded-learning":
        "Scaffolded Learning - Step-by-step progressive learning",
    };

    const teachingStyleIcons = {
      "simple-language": <School className="w-4 h-4" />,
      "analogy-driven": <Blocks className="w-4 h-4" />,
      "scaffolded-learning": <BarChart2 className="w-4 h-4" />,
    };

    return (
      <div className="w-full mb-4">
        <h3 className="text-sm font-semibold text-gray-700 mb-2">
          Select teaching style:
        </h3>
        <div className="grid grid-cols-1 gap-2">
          {(
            [
              "simple-language",
              "analogy-driven",
              "scaffolded-learning",
            ] as const
          ).map((style) => (
            <motion.button
              key={style}
              onClick={() => setTeachingStyle(style)}
              className={`py-2 px-3 rounded-lg border text-sm transition-all ${
                teachingStyle === style
                  ? "border-green-800 bg-green-50 text-green-800"
                  : "border-gray-300 hover:border-green-300 hover:bg-green-50/50"
              }`}
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
            >
              <div className="flex items-center">
                <div
                  className={`w-6 h-6 rounded-full flex items-center justify-center mr-2 ${
                    teachingStyle === style ? "bg-green-100" : "bg-gray-100"
                  }`}
                >
                  {teachingStyleIcons[style]}
                </div>
                <span className="font-medium">
                  {style === "simple-language"
                    ? "Explain Like I'm 5"
                    : style === "analogy-driven"
                    ? "Analogy Driven"
                    : "Scaffolded Learning"}
                </span>
                {teachingStyle === style && (
                  <div className="w-3 h-3 rounded-full bg-green-800 ml-2"></div>
                )}
              </div>
              <p className="text-xs text-gray-500 mt-1 pl-8 text-left">
                {teachingStyleInfo[style]}
              </p>
            </motion.button>
          ))}
        </div>
      </div>
    );
  };

  return (
    <motion.main
      className="min-h-screen bg-amber-50"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.5 }}
    >
      <div className="container mx-auto pt-24 sm:pt-28 md:pt-32 px-4 pb-16 flex items-center justify-center">
        <div className="w-full max-w-xl mx-auto">
          <motion.h1
            className="text-2xl sm:text-3xl md:text-4xl font-bold text-green-800 text-center mb-1 sm:mb-2"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            Upload Your Presentation
          </motion.h1>
          <motion.p
            className="text-sm md:text-md text-gray-500 text-center mb-8 sm:mb-12"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            Transform your slides into interactive learning materials instantly
          </motion.p>

          <motion.div
            className={`mt-4 sm:mt-6 border-2 border-dashed rounded-lg p-6 sm:p-8 md:p-12 flex flex-col items-center justify-center transition-colors ${
              isDragging ? "border-green-800 bg-green-50" : "border-gray-300"
            }`}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.2 }}
          >
            {!file ? (
              <>
                <div className="w-12 sm:w-16 h-12 sm:h-16 bg-amber-100 rounded-full flex items-center justify-center mb-3 sm:mb-4">
                  <Upload className="w-6 sm:w-8 h-6 sm:h-8 text-green-800" />
                </div>
                <h3 className="text-lg sm:text-xl font-semibold text-gray-700 mb-1 sm:mb-2 text-center">
                  Drag & drop your presentation
                </h3>
                <p className="text-sm text-gray-500 mb-3 sm:mb-4 text-center">
                  Support for PowerPoint, PDF, Keynote, Word, Excel and
                  OpenDocument formats
                </p>
                <div className="flex gap-2 items-center">
                  <div className="h-px w-16 sm:w-20 bg-gray-300"></div>
                  <span className="text-gray-500 text-xs sm:text-sm">OR</span>
                  <div className="h-px w-16 sm:w-20 bg-gray-300"></div>
                </div>
                <label className="mt-3 sm:mt-4">
                  <input
                    type="file"
                    className="hidden"
                    accept=".pdf,.pptx,.docx,.xlsx,.odt,.odp,.ods"
                    multiple={false}
                    onChange={handleFileChange}
                  />
                  <motion.span
                    className="px-4 sm:px-5 py-2 bg-green-800 text-white rounded-full text-xs sm:text-sm font-semibold cursor-pointer hover:bg-green-700 transition"
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                  >
                    Browse Files
                  </motion.span>
                </label>
              </>
            ) : (
              <div className="w-full">
                <div className="flex items-center justify-between mb-4 sm:mb-6">
                  <div className="flex items-center gap-2 sm:gap-3">
                    <div className="w-8 sm:w-10 h-8 sm:h-10 bg-green-100 rounded-full flex items-center justify-center">
                      <Check className="w-4 sm:w-5 h-4 sm:h-5 text-green-800" />
                    </div>
                    <div>
                      <p className="font-medium text-sm sm:text-base">
                        {file.name}
                      </p>
                      <p className="text-xs sm:text-sm text-gray-500">
                        {(file.size / (1024 * 1024)).toFixed(2)} MB
                      </p>
                    </div>
                  </div>
                  <motion.button
                    onClick={resetFile}
                    className="p-1.5 sm:p-2 hover:bg-gray-100 rounded-full"
                    whileHover={{ scale: 1.1 }}
                    whileTap={{ scale: 0.9 }}
                    disabled={uploading}
                  >
                    <X className="w-4 sm:w-5 h-4 sm:h-5 text-gray-500" />
                  </motion.button>
                </div>

                {!uploading && (
                  <>
                    <ServiceTypeSelector />
                    {serviceType === "testme" ? (
                      <DifficultySelector />
                    ) : (
                      <TeachingStyleSelector />
                    )}
                  </>
                )}

                {uploading ? (
                  <div className="w-full">
                    <div className="flex flex-col items-center justify-center mb-4">
                      <motion.div
                        className="w-10 h-10 mb-3"
                        animate={{ rotate: 360 }}
                        transition={{
                          duration: 1.5,
                          repeat: Infinity,
                          ease: "linear",
                        }}
                      >
                        <Loader2 className="w-10 h-10 text-green-800" />
                      </motion.div>
                      <p className="text-sm text-green-800 font-medium">
                        {connectionStatus === "connecting"
                          ? "Connecting to server..."
                          : connectionStatus === "error"
                          ? "Connection error"
                          : "Processing your document..."}
                      </p>
                    </div>

                    <div className="mt-4 max-h-40 overflow-y-auto border border-gray-200 rounded-lg bg-white p-3">
                      {progressMessages.length > 0 ? (
                        progressMessages.map((message, index) => (
                          <p key={index} className="text-xs text-gray-600 mb-1">
                            {message}
                          </p>
                        ))
                      ) : (
                        <p className="text-xs text-gray-400 text-center">
                          {connectionStatus === "connecting"
                            ? "Establishing connection..."
                            : "Waiting for processing updates..."}
                        </p>
                      )}
                    </div>
                  </div>
                ) : (
                  <motion.button
                    onClick={handleUpload}
                    className="w-full py-2.5 sm:py-3 rounded-full text-white font-semibold bg-green-800 hover:bg-green-700 transition flex items-center justify-center gap-2"
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                  >
                    <BookOpen className="w-4 h-4 mr-1" />
                    {serviceType === "testme"
                      ? "Generate Learning Materials"
                      : "Create Teaching Content"}
                  </motion.button>
                )}
              </div>
            )}
          </motion.div>
        </div>
      </div>
    </motion.main>
  );
}
