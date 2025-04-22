"use client";

import { ChevronDown } from "lucide-react";
import Link from "next/link";
import React, { useEffect, useState } from "react";

const Navbar = () => {
  const [isScrolled, setIsScrolled] = useState(false);

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
      name: "Platform",
      href: "/platfrom",
    },
    {
      name: "Upload",
      href: "/upload",
    },
    {
      name: "Partners",
      href: "/partners",
    },
    {
      name: "About",
      href: "/about",
    },
  ];

  return (
    <nav
      className={`w-full flex px-5 py-4 justify-between items-center fixed top-0 left-0 z-50 transition-all duration-300 ${
        isScrolled ? "bg-amber-50 shadow-sm" : "bg-transparent"
      }`}
    >
      <div className="gap-4 flex items-center">
        {links.map((link) => (
          <Navlink key={link.name} href={link.href} name={link.name} />
        ))}
      </div>
      <div>
        <button className="rounded-full bg-green-800 text-white flex items-center gap-1 px-3 py-2">
          <p className="text-sm font-semibold">Sign up</p>
          <ChevronDown width={20} />
        </button>
      </div>
    </nav>
  );
};

const Navlink = ({ name, href }: { name: string; href: string }) => (
  <Link className="flex items-center gap-1" href={href}>
    <p className="text-black text-sm font-medium">{name}</p>
    <ChevronDown size={18} />
  </Link>
);

export default Navbar;
