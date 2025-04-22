"use client";

import { cn } from "@/lib/utils";
import { useRouter } from "next/navigation";
import React from "react";
import Link from "next/link";

const SquigglyUnderline = () => (
  <svg
    width="100%"
    height="12"
    viewBox="0 0 120 12"
    preserveAspectRatio="none"
    className="mt-0.5 absolute -bottom-3"
  >
    <path
      d="M0,6 
         C5,2 10,10 15,6 
         C20,2 25,10 30,6 
         C35,2 40,10 45,6 
         C50,2 55,10 60,6
         C65,2 70,10 75,6
         C80,2 85,10 90,6
         C95,2 100,10 105,6
         C110,2 115,10 120,6"
      fill="none"
      stroke="#FF8C00"
      strokeWidth="3"
      strokeLinecap="round"
    />
  </svg>
);

const boxes = [
  {
    value: "98%",
    text: "Faster invoice processing",
  },
  {
    value: "3x",
    text: "Increase cash flow speed",
  },
  {
    value: "98%",
    text: "Collected via platfrom",
  },
  {
    value: "0",
    text: "Late payments reported",
  },
];

const Hero = () => {
  const router = useRouter();

  return (
    <section className="h-screen bg-amber-50 flex flex-col items-center justify-center py-20 relative">
      <div className="flex flex-col items-center gap-5 text-center">
        <h1 className="text-5xl font-bold tracking-tight text-green-800">
          <span className="flex items-center gap-2">
            Ace Your Exams in{" "}
            <span className="relative inline-block">
              5 Minutes
              <SquigglyUnderline />
            </span>
          </span>
        </h1>

        <p className="text-md text-gray-500 max-w-2xl">
          Convert your slides into interactive quizzes and flashcards instantly
        </p>

        <div className="flex gap-4 mt-4">
          <Link href="/upload">
            <button className="rounded-full bg-green-800 text-white px-5 py-2 text-sm font-semibold hover:bg-green-700 transition">
              Upload Presentation
            </button>
          </Link>
          <button
            onClick={() => router.push("/upload")}
            className="rounded-full border-2 border-green-800 text-green-800 px-5 py-2 text-sm font-semibold hover:bg-green-50 transition"
          >
            Try Demo
          </button>
        </div>
      </div>

      {/*  */}
      <div className="w-full flex absolute bottom-10 px-16">
        {boxes.map((box, idx) => (
          <StatBox key={box.text} name={box.text} value={box.value} idx={idx} />
        ))}
      </div>

      {/*  */}
    </section>
  );
};

const StatBox = ({
  name,
  value,
  idx,
}: {
  name: string;
  value: string;
  idx: number;
}) => (
  <div
    className={cn(
      "flex-1 flex flex-col items-start justify-center border-y border-black/5 px-2 gap-4 py-4",
      idx != 3 && "border-r"
    )}
  >
    <h1 className="text-green-800 text-4xl font-bold">{value}</h1>
    <p className="text-sm font-semibold text-gray-400">{name}</p>
  </div>
);

export default Hero;
