"use client";

import { useEffect, useState } from "react";
import { cn } from "./ui";

const slides = ["/bg-aurora.svg", "/bg-dusk.svg", "/bg-teal.svg"];
const SLIDE_MS = 14000;

interface Asteroid {
  left: string;
  delay: string;
  duration: string;
  size: number;
}

// Deterministic asteroid field (SSR-safe: no Math.random at render time).
const asteroids: Asteroid[] = [
  { left: "8%", delay: "0s", duration: "11s", size: 90 },
  { left: "22%", delay: "3.5s", duration: "14s", size: 60 },
  { left: "38%", delay: "7s", duration: "12s", size: 110 },
  { left: "55%", delay: "1.5s", duration: "16s", size: 70 },
  { left: "68%", delay: "9s", duration: "13s", size: 95 },
  { left: "82%", delay: "4.5s", duration: "15s", size: 65 },
  { left: "92%", delay: "11s", duration: "12.5s", size: 80 },
  { left: "15%", delay: "13s", duration: "17s", size: 55 },
];

/**
 * Site-wide fixed backdrop: crossfading night-sky scenes with a slow drift,
 * a falling-asteroid field, a rotating galaxy glow, and twinkling stars.
 * Content scrolls over it; prefers-reduced-motion keeps a static scene.
 */
export function Backdrop() {
  const [active, setActive] = useState(0);
  const [motion, setMotion] = useState(true);

  useEffect(() => {
    let interval = 0;
    const raf = requestAnimationFrame(() => {
      if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
        setMotion(false);
        return;
      }
      interval = window.setInterval(() => {
        setActive((a) => (a + 1) % slides.length);
      }, SLIDE_MS);
    });
    return () => {
      cancelAnimationFrame(raf);
      if (interval) window.clearInterval(interval);
    };
  }, []);

  return (
    <div aria-hidden className="fixed inset-0 -z-10 overflow-hidden bg-[#070b18]">
      {slides.map((src, i) => (
        <div
          key={src}
          className={cn(
            "absolute inset-0 transition-opacity ease-in-out",
            motion ? "duration-[2200ms]" : "duration-0",
            i === active ? "opacity-100" : "opacity-0",
          )}
        >
          {/* eslint-disable-next-line @next/next/no-img-element -- full-bleed decorative bg */}
          <img
            src={src}
            alt=""
            draggable={false}
            className={cn(
              "h-full w-full object-cover",
              motion && "animate-bg-drift",
            )}
            style={motion ? { animationDelay: `${i * -9}s` } : undefined}
          />
        </div>
      ))}

      {/* rotating galaxy glow */}
      <div
        className={cn(
          "absolute -right-[18%] -top-[22%] h-[70vmax] w-[70vmax] rounded-full opacity-[0.18] blur-3xl",
          motion && "animate-galaxy-spin",
        )}
        style={{
          background:
            "conic-gradient(from 0deg, transparent 0%, #7c6cf0 12%, transparent 28%, #3b82f6 45%, transparent 62%, #a78bfa 78%, transparent 100%)",
        }}
      />

      {/* falling asteroids */}
      {motion &&
        asteroids.map((a) => (
          <div
            key={a.left + a.delay}
            className="animate-asteroid absolute top-0 flex flex-col items-center"
            style={{
              left: a.left,
              animationDelay: a.delay,
              animationDuration: a.duration,
            }}
          >
            {/* trail points up/behind the fall direction */}
            <div
              className="w-px bg-gradient-to-t from-white/70 to-transparent"
              style={{ height: a.size }}
            />
            <div
              className="rounded-full bg-white/90 shadow-[0_0_12px_2px_rgba(168,156,248,0.8)]"
              style={{ width: 3, height: 3 }}
            />
          </div>
        ))}

      {/* twinkling star layer */}
      {motion && (
        <div className="absolute inset-0">
          {[
            { top: "12%", left: "18%", delay: "0s", dur: "3.2s" },
            { top: "28%", left: "44%", delay: "0.8s", dur: "2.6s" },
            { top: "8%", left: "71%", delay: "1.4s", dur: "3.8s" },
            { top: "42%", left: "86%", delay: "0.4s", dur: "2.9s" },
            { top: "55%", left: "12%", delay: "1.9s", dur: "3.4s" },
            { top: "64%", left: "58%", delay: "1.1s", dur: "2.4s" },
            { top: "33%", left: "29%", delay: "2.2s", dur: "3.6s" },
            { top: "72%", left: "76%", delay: "0.6s", dur: "3.1s" },
            { top: "18%", left: "93%", delay: "1.7s", dur: "2.8s" },
            { top: "80%", left: "35%", delay: "2.5s", dur: "3.9s" },
          ].map((s) => (
            <span
              key={s.top + s.left}
              className="animate-twinkle absolute h-[2px] w-[2px] rounded-full bg-white"
              style={{
                top: s.top,
                left: s.left,
                animationDelay: s.delay,
                animationDuration: s.dur,
              }}
            />
          ))}
        </div>
      )}

      {/* Theme veil: keeps dark mode moody, washes the art for light-mode text. */}
      <div className="bg-veil absolute inset-0" />
    </div>
  );
}
