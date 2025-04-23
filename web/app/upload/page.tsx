"use client";

import React, { useState, useEffect, useCallback } from "react";
import { Upload, Check, X } from "lucide-react";
import Navbar from "@/components/Navbar";
import { toast } from "sonner";
import initiateProcessing from "@/services/requests/parse-presentation";
import { useResultStore } from "@/store/useResultStore";
import { useRouter } from "next/navigation";

export default function UploadPage() {
  const [file, setFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [sse, setSse] = useState<EventSource | null>(null);
  const setResults = useResultStore((state) => state.setResults);
  const router = useRouter();

  const cleanup = useCallback(() => {
    setUploading(false);
    sse?.close();
    setSse(null);
  }, [sse]);

  useEffect(() => {
    if (!sse) return;

    sse.addEventListener("progress", (event) => {
      toast.info(event.data);
    });

    sse.addEventListener("finished", (event) => {
      try {
        // Parse the result data and store it
        const decodedData = atob(event.data); // If using base64
        const resultData = JSON.parse(decodedData);

        setResults({
          flashcards: resultData.flashcards || [],
          quizzes: resultData.quizzes || [],
        });

        // Navigate to results page
        router.push("/results");
      } catch (error) {
        console.error("Failed to parse result data:", error);
        toast.error("Failed to process presentation results");
      }

      cleanup();
    });

    sse.addEventListener("error", (event) => {
      console.error("SSE error:", event);
      toast.error("An error occurred during processing");
      cleanup();
    });

    sse.onerror = () => {
      toast.error("Connection failed or closed unexpectedly");
      cleanup();
    };

    return () => {
      cleanup();
    };
  }, [sse, cleanup, setResults, router]);

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

    const formData = new FormData();
    formData.append("file", file);

    // Convert PDF to text
    const pdfResponse = await fetch("/api/parse-file", {
      method: "POST",
      body: formData,
    });
    const pdfData = await pdfResponse.json();

    // Initiate processing
    const response = await initiateProcessing(pdfData.parsedText);
    if (response.error) {
      toast.error(response.error);
      setUploading(false);
      return;
    }
    const taskID = response.data.task_id;

    // begin the SSE with the taskID recieved
    const sseURL = new URL(`${process.env.NEXT_PUBLIC_API_URL}/sse`);
    sseURL.searchParams.append("task_id", taskID);

    // create event source
    const sse = new EventSource(sseURL);

    setSse(sse);
  };

  const resetFile = () => {
    setFile(null);
  };

  return (
    <main className="min-h-screen bg-amber-50">
      <Navbar />
      <div className="container mx-auto pt-24 sm:pt-28 md:pt-32 px-4 pb-16 flex items-center justify-center">
        <div className="w-full max-w-xl mx-auto">
          <h1 className="text-2xl sm:text-3xl md:text-4xl font-bold text-green-800 text-center mb-1 sm:mb-2">
            Upload Your Presentation
          </h1>
          <p className="text-sm md:text-md text-gray-500 text-center mb-8 sm:mb-12">
            Transform your slides into interactive quizzes and flashcards
            instantly
          </p>

          <div
            className={`mt-4 sm:mt-6 border-2 border-dashed rounded-lg p-6 sm:p-8 md:p-12 flex flex-col items-center justify-center transition-colors ${
              isDragging ? "border-green-800 bg-green-50" : "border-gray-300"
            }`}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
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
                  <span className="px-4 sm:px-5 py-2 bg-green-800 text-white rounded-full text-xs sm:text-sm font-semibold cursor-pointer hover:bg-green-700 transition">
                    Browse Files
                  </span>
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
                  <button
                    onClick={resetFile}
                    className="p-1.5 sm:p-2 hover:bg-gray-100 rounded-full"
                  >
                    <X className="w-4 sm:w-5 h-4 sm:h-5 text-gray-500" />
                  </button>
                </div>

                <button
                  onClick={handleUpload}
                  disabled={uploading}
                  className={`w-full py-2.5 sm:py-3 rounded-full text-white font-semibold ${
                    uploading
                      ? "bg-green-700"
                      : "bg-green-800 hover:bg-green-700"
                  } transition flex items-center justify-center gap-2`}
                >
                  {uploading ? "Uploading..." : "Convert Presentation"}
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}
