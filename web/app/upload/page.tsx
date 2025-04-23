"use client";

import React, { useState, useEffect, useCallback, useRef } from "react";
import { Upload, Check, X } from "lucide-react";
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
  const setResults = useResultStore((state) => state.setResults);
  const router = useRouter();

  // Use refs to avoid circular dependencies
  const handleProgressRef = useRef<(event: MessageEvent) => void>(() => {});
  const handleFinishedRef = useRef<(event: MessageEvent) => void>(() => {});
  const handleErrorRef = useRef<(event: Event) => void>(() => {});

  const cleanup = useCallback(() => {
    if (sse) {
      sse.removeEventListener("progress", handleProgressRef.current);
      sse.removeEventListener("finished", handleFinishedRef.current);
      sse.removeEventListener("error", handleErrorRef.current);
      sse.close();
      setSse(null);
    }
    setUploading(false);
  }, [sse]);

  // Set up the event handlers with refs to avoid circular deps
  handleProgressRef.current = (event: MessageEvent) => {
    toast.info(event.data);
  };

  handleFinishedRef.current = (event: MessageEvent) => {
    try {
      const decodedData = atob(event.data);
      const resultData = JSON.parse(decodedData);

      setResults({
        flashcards: resultData.flashcards || [],
        quizzes: resultData.quizzes || [],
      });

      toast.success("Successfully generated content!");

      // Clean up before navigation
      cleanup();

      // Navigate to results page
      router.push("/results");
    } catch (error) {
      console.error("Failed to parse result data:", error);
      toast.error("Failed to process presentation results");
      cleanup();
    }
  };

  handleErrorRef.current = (event: Event) => {
    console.error("SSE error:", event);
    toast.error("An error occurred during processing");
    cleanup();
  };

  useEffect(() => {
    if (!sse) return;

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
    setUploading(true);

    try {
      const formData = new FormData();
      formData.append("file", file);

      // Convert PDF to text
      const pdfResponse = await fetch("/api/parse-file", {
        method: "POST",
        body: formData,
      });

      if (!pdfResponse.ok) {
        throw new Error(`Failed to parse file: ${pdfResponse.statusText}`);
      }

      const pdfData = await pdfResponse.json();

      // Initiate processing
      const response = await initiateProcessing(pdfData.parsedText);
      if (response.error) {
        toast.error(response.error);
        setUploading(false);
        return;
      }
      const taskID = response.data.task_id;

      // Close any existing SSE connection
      if (sse) {
        cleanup();
      }

      // begin the SSE with the taskID received
      const sseURL = new URL(`${process.env.NEXT_PUBLIC_API_URL}/sse`);
      sseURL.searchParams.append("task_id", taskID);

      // create event source with proper error handling
      const eventSource = new EventSource(sseURL.toString());
      setSse(eventSource);

      toast.info("Processing started...");
    } catch (error) {
      console.error("Upload error:", error);
      toast.error(
        error instanceof Error ? error.message : "Failed to upload file"
      );
      setUploading(false);
    }
  };

  const resetFile = () => {
    setFile(null);
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
            Transform your slides into interactive quizzes and flashcards
            instantly
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
                  Support for PowerPoint, PDF, Keynote
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
                    accept=".pdf"
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
                  >
                    <X className="w-4 sm:w-5 h-4 sm:h-5 text-gray-500" />
                  </motion.button>
                </div>

                <motion.button
                  onClick={handleUpload}
                  disabled={uploading}
                  className={`w-full py-2.5 sm:py-3 rounded-full text-white font-semibold ${
                    uploading
                      ? "bg-green-700"
                      : "bg-green-800 hover:bg-green-700"
                  } transition flex items-center justify-center gap-2`}
                  whileHover={!uploading ? { scale: 1.02 } : {}}
                  whileTap={!uploading ? { scale: 0.98 } : {}}
                >
                  {uploading ? "Processing..." : "Convert Presentation"}
                </motion.button>
              </div>
            )}
          </motion.div>
        </div>
      </div>
    </motion.main>
  );
}
