"use client";

import { BookOpen, Upload } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";

const Navbar = () => {
  const [isScrolled, setIsScrolled] = useState(false);
  const pathname = usePathname();

  useEffect(() => {
    const handleScroll = () => {
      if (window.scrollY > 10) {
        setIsScrolled(true);
      } else {
        setIsScrolled(false);
      }
    };

    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const links = [
    {
      name: "Upload",
      href: "/upload",
      icon: <Upload className="w-4 h-4" />,
    },
    {
      name: "Results",
      href: "/results",
      icon: <BookOpen className="w-4 h-4" />,
    },
  ];

  return (
    <motion.nav
      initial={{ y: -100 }}
      animate={{ y: 0 }}
      transition={{ type: "spring", stiffness: 300, damping: 30 }}
      className={`w-full flex px-6 py-4 justify-center items-center fixed top-0 left-0 z-50 transition-all duration-300 ${
        isScrolled
          ? "bg-amber-50/95 backdrop-blur-sm shadow-sm"
          : "bg-amber-50/80 backdrop-blur-sm"
      }`}
    >
      <div className="max-w-6xl w-full flex justify-between items-center">
        <Link href="/" className="flex items-center">
          <motion.span
            className="font-bold text-xl text-green-800"
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
          >
            MemorisePDF
          </motion.span>
        </Link>

        <motion.div
          className="flex items-center gap-1 sm:gap-2"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.2, duration: 0.5 }}
        >
          {links.map((link, index) => (
            <Navlink
              key={link.name}
              href={link.href}
              name={link.name}
              icon={link.icon}
              isActive={pathname === link.href}
              index={index}
            />
          ))}
        </motion.div>
      </div>
    </motion.nav>
  );
};

const Navlink = ({
  name,
  href,
  icon,
  isActive,
  index,
}: {
  name: string;
  href: string;
  icon: React.ReactNode;
  isActive: boolean;
  index: number;
}) => (
  <motion.div
    initial={{ opacity: 0, y: -20 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ delay: 0.3 + index * 0.1, duration: 0.5 }}
  >
    <Link
      href={href}
      className={`flex items-center gap-1.5 px-3 py-2 rounded-full transition-colors ${
        isActive ? "bg-green-100 text-green-800" : "hover:bg-amber-100"
      }`}
    >
      <motion.div
        whileHover={{ rotate: [0, -10, 10, -10, 0] }}
        transition={{ duration: 0.5 }}
      >
        {icon}
      </motion.div>
      <motion.span
        className="text-sm font-medium hidden sm:inline"
        whileHover={{ scale: 1.05 }}
      >
        {name}
      </motion.span>
    </Link>
  </motion.div>
);

export default Navbar;
