"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Reveal } from "./Reveal";

interface Chip {
  slug: string;
  label: string;
  className: string;
  factor: number;
}

const chips: Chip[] = [
  {
    slug: "gaurav-resume-pm",
    label: "resume → PM role",
    className: "left-[6%] top-[18%]",
    factor: 0.55,
  },
  {
    slug: "google-pm-referral",
    label: "job app → referral",
    className: "right-[8%] top-[14%]",
    factor: 0.7,
  },
  {
    slug: "hn-launch-post",
    label: "launch → HN thread",
    className: "left-[14%] bottom-[24%]",
    factor: 0.85,
  },
  {
    slug: "nvidia-swe-2026",
    label: "job id as slug",
    className: "right-[12%] bottom-[20%]",
    factor: 0.6,
  },
  {
    slug: "ig-bio-portfolio",
    label: "social → portfolio",
    className: "left-[42%] top-[8%]",
    factor: 0.9,
  },
];

/**
 * Parallax band over the shared fixed backdrop: floating slug chips drift at
 * different scroll rates; the site-wide scene shows through. Reduced motion
 * renders a static composition.
 */
export function ParallaxShowcase() {
  const ref = useRef<HTMLElement>(null);
  const [progress, setProgress] = useState(0.5);

  useEffect(() => {
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    let raf = 0;
    const measure = () => {
      raf = 0;
      const el = ref.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      const vh = window.innerHeight || 1;
      const p = (vh - rect.top) / (vh + rect.height);
      setProgress(Math.max(0, Math.min(1, p)));
    };
    const onScroll = () => {
      if (!raf) raf = requestAnimationFrame(measure);
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    raf = requestAnimationFrame(measure);
    return () => {
      window.removeEventListener("scroll", onScroll);
      if (raf) cancelAnimationFrame(raf);
    };
  }, []);

  // progress 0.5 = centered; chips drift ±60px * factor across the band.
  const shift = (factor: number) => (progress - 0.5) * 120 * factor;

  return (
    <section
      ref={ref}
      aria-label="Custom slug examples"
      className="relative isolate h-[400px] overflow-hidden sm:h-[480px]"
    >
      {/* floating slug chips */}
      {chips.map((chip) => (
        <div
          key={chip.slug}
          className={`absolute will-change-transform ${chip.className}`}
          style={{ transform: `translate3d(0, ${shift(chip.factor)}px, 0)` }}
        >
          <Link
            href="/dashboard"
            className="animate-float glass block rounded-lg border border-white/15 bg-black/50 px-3 py-2 shadow-lg transition-colors hover:border-white/40"
          >
            <span className="block font-mono text-[12px] font-medium text-white sm:text-[13px]">
              cut.sh/<span className="text-[#a89cf8]">{chip.slug}</span>
            </span>
            <span className="mt-0.5 block text-[10px] text-white/60 sm:text-[11px]">
              {chip.label}
            </span>
          </Link>
        </div>
      ))}

      {/* center caption */}
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
        <Reveal className="pointer-events-auto glass rounded-xl border border-white/15 bg-black/45 px-5 py-4 text-center sm:px-8 sm:py-5">
          <p className="text-[11px] font-medium tracking-wide text-white/60 uppercase sm:text-[12px]">
            Custom slugs, real meaning
          </p>
          <p className="mt-1.5 max-w-md font-display text-lg font-semibold text-balance text-white sm:text-2xl">
            Job IDs, referral codes, post names — slugs a human can read
          </p>
          <p className="mt-1.5 text-[12px] text-white/70 sm:text-[13px]">
            Each one is a trackable channel in your analytics
          </p>
        </Reveal>
      </div>
    </section>
  );
}
