"use client";

import { motion } from "framer-motion";

const Footer = () => {
  const currentYear = new Date().getFullYear();

  return (
    <motion.footer
      className="w-full bg-amber-50 py-4 border-t border-amber-100/50"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5, delay: 0.3 }}
    >
      <div className="container mx-auto px-6 flex flex-row justify-center items-center">
        <div className="text-xs text-gray-400 flex flex-wrap justify-center gap-4 items-center">
          <span>© {currentYear} MemorisePDF</span>
        </div>
      </div>
    </motion.footer>
  );
};

export default Footer;
