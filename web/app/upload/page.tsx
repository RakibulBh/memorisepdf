"use client";

import React, { useState } from "react";
import { Upload, Check, X } from "lucide-react";
import Navbar from "@/components/Navbar";
import { toast } from "sonner";
import parsePresentation from "@/services/requests/parse-presentation";

export default function UploadPage() {
  const [file, setFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [uploading, setUploading] = useState(false);

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

    const response = await fetch("/api/parse-file", {
      method: "POST",
      body: formData,
    });

    const data = await response.json();

    const parsedPresentation = await parsePresentation(data.parsedText);

    console.log("Parsed presentation", parsedPresentation);

    setUploading(false);
  };

  const resetFile = () => {
    setFile(null);
  };

  return (
    <main className="min-h-screen bg-amber-50">
      <Navbar />
      <div className="container mx-auto pt-32 px-4 pb-16">
        <div className="max-w-2xl mx-auto">
          <h1 className="text-4xl font-bold text-green-800 text-center mb-2">
            Upload Your Presentation
          </h1>
          <p className="text-md text-gray-500 text-center mb-12">
            Transform your slides into interactive quizzes and flashcards
            instantly
          </p>

          <div
            className={`mt-6 border-2 border-dashed rounded-lg p-12 flex flex-col items-center justify-center transition-colors ${
              isDragging ? "border-green-800 bg-green-50" : "border-gray-300"
            }`}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
          >
            {!file ? (
              <>
                <div className="w-16 h-16 bg-amber-100 rounded-full flex items-center justify-center mb-4">
                  <Upload className="w-8 h-8 text-green-800" />
                </div>
                <h3 className="text-xl font-semibold text-gray-700 mb-2">
                  Drag & drop your presentation
                </h3>
                <p className="text-gray-500 mb-4">
                  Support for PowerPoint, PDF, Keynote
                </p>
                <div className="flex gap-2 items-center">
                  <div className="h-px w-20 bg-gray-300"></div>
                  <span className="text-gray-500 text-sm">OR</span>
                  <div className="h-px w-20 bg-gray-300"></div>
                </div>
                <label className="mt-4">
                  <input
                    type="file"
                    className="hidden"
                    accept=".pdf"
                    multiple={false}
                    onChange={handleFileChange}
                  />
                  <span className="px-5 py-2 bg-green-800 text-white rounded-full text-sm font-semibold cursor-pointer hover:bg-green-700 transition">
                    Browse Files
                  </span>
                </label>
              </>
            ) : (
              <div className="w-full">
                <div className="flex items-center justify-between mb-6">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-green-100 rounded-full flex items-center justify-center">
                      <Check className="w-5 h-5 text-green-800" />
                    </div>
                    <div>
                      <p className="font-medium">{file.name}</p>
                      <p className="text-sm text-gray-500">
                        {(file.size / (1024 * 1024)).toFixed(2)} MB
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={resetFile}
                    className="p-2 hover:bg-gray-100 rounded-full"
                  >
                    <X className="w-5 h-5 text-gray-500" />
                  </button>
                </div>

                <button
                  onClick={handleUpload}
                  disabled={uploading}
                  className={`w-full py-3 rounded-full text-white font-semibold ${
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
