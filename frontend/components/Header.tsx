"use client";

import Link from "next/link";
import { Button } from "./ui";

export function Header({ onLogout }: { onLogout?: () => void }) {
  return (
    <div className="sticky top-0 z-50 px-4 pt-4 sm:px-6">
      <header className="glass relative mx-auto flex h-14 max-w-4xl items-center justify-between rounded-2xl border border-border-strong/60 bg-background/60 pl-4 pr-2 shadow-[0_8px_32px_rgba(0,0,0,0.35)]">
        <span
          aria-hidden
          className="pointer-events-none absolute inset-x-6 -top-px h-px bg-gradient-to-r from-transparent via-accent/50 to-transparent"
        />
        <Link href="/" className="group flex items-center gap-2.5">
          <span className="relative flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-accent via-accent-hover to-[#8b5cf6] text-sm font-bold text-white shadow-[0_0_18px_rgba(50,145,255,0.35)] transition-shadow group-hover:shadow-[0_0_24px_rgba(50,145,255,0.55)]">
            ✂
          </span>
          <span className="text-[15px] font-semibold tracking-tight">
            cut<span className="text-muted">.sh</span>
          </span>
          <span className="ml-1 hidden rounded-full border border-border px-2 py-0.5 text-[11px] text-muted sm:inline">
            beta
          </span>
        </Link>
        <div className="flex items-center gap-1">
          {onLogout && (
            <Button variant="ghost" size="sm" onClick={onLogout}>
              Sign out
            </Button>
          )}
        </div>
      </header>
    </div>
  );
}
