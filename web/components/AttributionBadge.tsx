"use client";

import { motion } from "framer-motion";

const AttributionBadge = () => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="inline-flex items-center px-4 py-2 bg-gradient-to-r from-green-500 to-amber-500 rounded-full text-white shadow-md"
    >
      <span className="text-sm font-medium">Made by</span>
      <a
        href="https://www.linkedin.com/in/rakibulb"
        target="_blank"
        rel="noopener noreferrer"
        className="ml-1 text-sm font-bold hover:underline"
      >
        Rakibul Bhuiyan
      </a>
    </motion.div>
  );
};

export default AttributionBadge;
