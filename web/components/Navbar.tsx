"use client";

import { BookOpen, Upload } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import React, { useEffect, useState } from "react";

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
    <nav
      className={`w-full flex px-6 py-4 justify-center items-center fixed top-0 left-0 z-50 transition-all duration-300 ${
        isScrolled
          ? "bg-amber-50/95 backdrop-blur-sm shadow-sm"
          : "bg-amber-50/80 backdrop-blur-sm"
      }`}
    >
      <div className="max-w-6xl w-full flex justify-between items-center">
        <Link href="/" className="flex items-center">
          <span className="font-bold text-xl text-green-800">
            Unmemorizable
          </span>
        </Link>

        <div className="flex items-center gap-1 sm:gap-2">
          {links.map((link) => (
            <Navlink
              key={link.name}
              href={link.href}
              name={link.name}
              icon={link.icon}
              isActive={pathname === link.href}
            />
          ))}
        </div>
      </div>
    </nav>
  );
};

const Navlink = ({
  name,
  href,
  icon,
  isActive,
}: {
  name: string;
  href: string;
  icon: React.ReactNode;
  isActive: boolean;
}) => (
  <Link
    href={href}
    className={`flex items-center gap-1.5 px-3 py-2 rounded-full transition-colors ${
      isActive ? "bg-green-100 text-green-800" : "hover:bg-amber-100"
    }`}
  >
    {icon}
    <span className="text-sm font-medium hidden sm:inline">{name}</span>
  </Link>
);

export default Navbar;
