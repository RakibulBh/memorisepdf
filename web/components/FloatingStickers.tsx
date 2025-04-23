"use client";

import { useEffect, useState } from "react";
import Image from "next/image";
import { motion } from "framer-motion";

interface Sticker {
  src: string;
  initialPosition: { x: number; y: number };
  size: number;
  duration: number;
  delay: number;
}

const FloatingStickers = () => {
  const [stickers, setStickers] = useState<Sticker[]>([
    {
      src: "/stickers/check.png",
      initialPosition: { x: 5, y: 20 },
      size: 100,
      duration: 8,
      delay: 0,
    },
    {
      src: "/stickers/test.png",
      initialPosition: { x: 80, y: 15 },
      size: 100,
      duration: 10,
      delay: 1,
    },
    {
      src: "/stickers/gamepad.png",
      initialPosition: { x: 15, y: 55 },
      size: 100,
      duration: 9,
      delay: 2,
    },
    {
      src: "/stickers/grade.png",
      initialPosition: { x: 75, y: 55 },
      size: 100,
      duration: 7,
      delay: 3,
    },
  ]);

  return (
    <div className="absolute inset-0 overflow-hidden pointer-events-none">
      {stickers.map((sticker, index) => (
        <motion.div
          key={index}
          className="absolute"
          style={{
            left: `${sticker.initialPosition.x}%`,
            top: `${sticker.initialPosition.y}%`,
          }}
          animate={{
            y: [0, -20, 0],
            x: [0, 15, 0],
            rotate: [0, 5, -5, 0],
          }}
          transition={{
            duration: sticker.duration,
            repeat: Infinity,
            repeatType: "reverse",
            ease: "easeInOut",
            delay: sticker.delay,
          }}
        >
          <Image
            src={sticker.src}
            alt="Floating sticker"
            width={sticker.size}
            height={sticker.size}
            className="object-contain opacity-80 select-none"
          />
        </motion.div>
      ))}
    </div>
  );
};

export default FloatingStickers;
