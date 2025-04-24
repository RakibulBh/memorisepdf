"use client";

import { cn } from "@/lib/utils";
import { useRouter } from "next/navigation";
import React, { useState, useEffect } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import FloatingStickers from "./FloatingStickers";
import AttributionBadge from "./AttributionBadge";

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
    text: "Faster study time",
    startValue: 0,
    endValue: 98,
    suffix: "%",
  },
  {
    value: "5x",
    text: "Higher engagement",
    startValue: 0,
    endValue: 5,
    suffix: "x",
  },
  {
    value: "89%",
    text: "Better retention",
    startValue: 0,
    endValue: 89,
    suffix: "%",
  },
  {
    value: "<1min",
    text: "Instant quiz generation",
    startValue: 60,
    endValue: 1,
    prefix: "<",
    suffix: "min",
  },
];

const fadeInUpVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0 },
};

const staggerChildrenVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.2,
    },
  },
};

const Hero = () => {
  const router = useRouter();

  return (
    <section className="min-h-screen bg-amber-50 flex flex-col items-center justify-center py-16 md:py-20 relative px-4 overflow-hidden">
      <FloatingStickers />

      <motion.div
        className="flex flex-col items-center gap-5 text-center z-10"
        initial="hidden"
        animate="visible"
        variants={staggerChildrenVariants}
      >
        {/* Attribution Badge */}
        <motion.div
          variants={fadeInUpVariants}
          transition={{ duration: 0.6, delay: 0.1 }}
        >
          <AttributionBadge />
        </motion.div>

        <motion.h1
          className="text-3xl md:text-4xl lg:text-5xl font-bold tracking-tight text-green-800"
          variants={fadeInUpVariants}
          transition={{ duration: 0.6 }}
        >
          <span className="flex flex-col md:flex-row items-center gap-2">
            Ace Your Exams in{" "}
            <span className="relative inline-block">
              5 Minutes
              <SquigglyUnderline />
            </span>
          </span>
        </motion.h1>

        <motion.p
          className="text-sm md:text-md text-gray-500 max-w-2xl"
          variants={fadeInUpVariants}
          transition={{ duration: 0.6, delay: 0.2 }}
        >
          Convert your slides into interactive quizzes and flashcards instantly
        </motion.p>

        <motion.div
          className="flex flex-col sm:flex-row gap-3 sm:gap-4 mt-4 w-full sm:w-auto"
          variants={fadeInUpVariants}
          transition={{ duration: 0.6, delay: 0.4 }}
        >
          <Link href="/upload" className="w-full sm:w-auto">
            <motion.button
              className="w-full rounded-full bg-green-800 text-white px-5 py-2 text-sm font-semibold hover:bg-green-700 transition"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              Upload Presentation
            </motion.button>
          </Link>
          <motion.button
            onClick={() => router.push("/upload")}
            className="w-full sm:w-auto rounded-full border-2 border-green-800 text-green-800 px-5 py-2 text-sm font-semibold hover:bg-green-50 transition"
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
          >
            Try Demo
          </motion.button>
        </motion.div>
      </motion.div>

      {/* Stats Section */}
      <motion.div
        className="w-full grid grid-cols-2 md:flex absolute bottom-0 sm:bottom-10 px-4 sm:px-8 md:px-16"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.8, delay: 0.6 }}
      >
        {boxes.map((box, idx) => (
          <StatBox
            key={box.text}
            name={box.text}
            startValue={box.startValue}
            endValue={box.endValue}
            prefix={box.prefix}
            suffix={box.suffix}
            idx={idx}
            totalBoxes={boxes.length}
          />
        ))}
      </motion.div>
    </section>
  );
};

const StatBox = ({
  name,
  startValue,
  endValue,
  prefix = "",
  suffix = "",
  idx,
  totalBoxes,
}: {
  name: string;
  startValue: number;
  endValue: number;
  prefix?: string;
  suffix?: string;
  idx: number;
  totalBoxes: number;
}) => {
  const [count, setCount] = useState(startValue);

  useEffect(() => {
    const duration = 2000; // 2 seconds
    const steps = 30;
    const stepTime = duration / steps;

    // Calculate the increment per step
    const increment = (endValue - startValue) / steps;

    let currentStep = 0;
    const timer = setInterval(() => {
      currentStep += 1;
      const newValue = startValue + increment * Math.min(currentStep, steps);

      // If we're decrementing (endValue < startValue), we want to round down
      // If we're incrementing (endValue > startValue), we want to round up
      const roundedValue =
        endValue < startValue ? Math.ceil(newValue) : Math.floor(newValue);

      setCount(roundedValue);

      if (currentStep >= steps) {
        clearInterval(timer);
        setCount(endValue);
      }
    }, stepTime);

    return () => clearInterval(timer);
  }, [startValue, endValue]);

  return (
    <motion.div
      className={cn(
        "flex-1 flex flex-col items-start justify-center border-y border-black/5 px-2 gap-2 sm:gap-4 py-3 sm:py-4",
        idx % 2 !== 1 && "border-r md:border-r",
        idx < totalBoxes - 2 && "md:border-r"
      )}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: 0.7 + idx * 0.1 }}
    >
      <motion.h1
        className="text-2xl sm:text-3xl md:text-4xl text-green-800 font-bold"
        whileHover={{ scale: 1.1, color: "#166534" }}
      >
        {prefix}
        {count}
        {suffix}
      </motion.h1>
      <p className="text-xs sm:text-sm font-semibold text-gray-400">{name}</p>
    </motion.div>
  );
};

export default Hero;
