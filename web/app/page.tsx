"use client";

import Hero from "@/components/Hero";
import { motion } from "framer-motion";
import { Analytics } from "@vercel/analytics/react";

export default function Home() {
  return (
    <>
      <Analytics />
      <motion.main
        className="relative"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.5 }}
      >
        <Hero />
      </motion.main>
    </>
  );
}
