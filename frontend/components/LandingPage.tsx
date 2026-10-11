"use client";

import Link from "next/link";
import { useState } from "react";
import { CountUp } from "./CountUp";
import { LandingShortener } from "./LandingShortener";
import { ParallaxShowcase } from "./ParallaxShowcase";
import { Reveal } from "./Reveal";
import { Badge, Card, buttonClass, cn } from "./ui";

/* ------------------------------ nav ------------------------------ */

const navLinks = [
  { label: "Features", href: "#features" },
  { label: "How it works", href: "#how" },
  { label: "Use cases", href: "#use-cases" },
  { label: "Compare", href: "#compare" },
  { label: "Self-host", href: "#self-host" },
];

function Logo({ compact }: { compact?: boolean }) {
  return (
    <Link href="/" className="group flex items-center gap-2.5">
      <span className="relative flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-accent via-accent-hover to-[#8b5cf6] text-sm font-bold text-white shadow-[0_0_18px_rgba(50,145,255,0.35)] transition-shadow group-hover:shadow-[0_0_24px_rgba(50,145,255,0.55)]">
        ✂
      </span>
      {!compact && (
        <span className="text-[15px] font-semibold tracking-tight">
          cut<span className="text-muted">.sh</span>
        </span>
      )}
    </Link>
  );
}

function Nav() {
  const [open, setOpen] = useState(false);

  return (
    <div className="sticky top-0 z-50 px-4 pt-4 sm:px-6">
      <nav className="glass relative mx-auto flex h-14 max-w-4xl items-center justify-between rounded-2xl border border-border-strong/60 bg-background/60 pl-4 pr-2 shadow-[0_8px_32px_rgba(0,0,0,0.35)]">
        {/* top highlight line */}
        <span
          aria-hidden
          className="pointer-events-none absolute inset-x-6 -top-px h-px bg-gradient-to-r from-transparent via-accent/50 to-transparent"
        />
        <Logo />
        <div className="hidden items-center gap-1 text-[13px] md:flex">
          {navLinks.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="rounded-full px-3 py-1.5 text-muted transition-colors hover:bg-card-hover hover:text-foreground"
            >
              {l.label}
            </a>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <Link
            href="/dashboard"
            className={cn(buttonClass("ghost", "sm"), "hidden sm:inline-flex")}
          >
            Sign in
          </Link>
          <Link href="/dashboard" className={buttonClass("primary", "sm")}>
            Dashboard
          </Link>
          <button
            type="button"
            aria-label={open ? "Close menu" : "Open menu"}
            aria-expanded={open}
            onClick={() => setOpen((v) => !v)}
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted transition-colors hover:bg-card-hover hover:text-foreground md:hidden"
          >
            <svg className="h-4.5 w-4.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden>
              {open ? (
                <path d="M18 6 6 18M6 6l12 12" />
              ) : (
                <path d="M4 7h16M4 12h16M4 17h16" />
              )}
            </svg>
          </button>
        </div>

        {/* mobile dropdown */}
        {open && (
          <div className="glass animate-in absolute inset-x-0 top-[calc(100%+8px)] rounded-2xl border border-border-strong/60 bg-background/85 p-2 shadow-[0_16px_48px_rgba(0,0,0,0.45)] md:hidden">
            {navLinks.map((l) => (
              <a
                key={l.href}
                href={l.href}
                onClick={() => setOpen(false)}
                className="block rounded-lg px-3 py-2.5 text-[14px] text-muted transition-colors hover:bg-card-hover hover:text-foreground"
              >
                {l.label}
              </a>
            ))}
            <div className="mt-1 border-t border-border pt-2 sm:hidden">
              <Link
                href="/dashboard"
                onClick={() => setOpen(false)}
                className="block rounded-lg px-3 py-2.5 text-[14px] text-accent transition-colors hover:bg-card-hover"
              >
                Sign in
              </Link>
            </div>
          </div>
        )}
      </nav>
    </div>
  );
}

/* ------------------------------ hero ------------------------------ */

function Hero() {
  return (
    <section className="relative overflow-hidden">
      <div className="relative mx-auto max-w-5xl px-4 pt-20 pb-16 text-center sm:px-6 sm:pt-24">
        <div className="hero-in" style={{ animationDelay: "0ms" }}>
          <Badge>
            Open source stack · Go + Chi · PostgreSQL · Redis · Next.js
          </Badge>
        </div>
        <h1
          className="hero-in mx-auto mt-6 max-w-3xl text-4xl font-semibold tracking-tight text-balance sm:text-6xl"
          style={{ animationDelay: "80ms" }}
        >
          Short links,
          <span className="bg-gradient-to-r from-accent via-foreground to-foreground bg-clip-text text-transparent">
            {" "}
            full control
          </span>
        </h1>
        <p
          className="hero-in mx-auto mt-5 max-w-xl text-[15px] leading-relaxed text-muted sm:text-base"
          style={{ animationDelay: "160ms" }}
        >
          Shorten right here — <span className="text-foreground">no account
            needed</span>. Sign in for custom slugs, click analytics, editing, and
          link management. Self-hostable, Redis-cached, privacy-conscious.
        </p>
        <div
          className="hero-in mt-10"
          style={{ animationDelay: "300ms" }}
        >
          <LandingShortener />
        </div>
        <div
          className="hero-in mt-8 flex items-center justify-center gap-3"
          style={{ animationDelay: "400ms" }}
        >
          <Link href="/dashboard" className={cn(buttonClass("secondary", "md"), "h-10 px-5")}>
            Open dashboard — manage &amp; analytics
          </Link>
          <a href="#self-host" className={cn(buttonClass("ghost", "md"), "h-10 px-5")}>
            Self-host in one command
          </a>
        </div>
      </div>
    </section>
  );
}

/* --------------------------- stats strip --------------------------- */

function Stats() {
  return (
    <section className="border-y border-border">
      <div className="mx-auto grid max-w-5xl grid-cols-1 divide-y divide-border px-4 sm:grid-cols-3 sm:divide-x sm:divide-y-0 sm:px-6">
        {[
          { value: 27, suffix: "µs", label: "cache-hit redirect resolution", note: "benchmarked on Apple M1 — run the suite yourself" },
          { value: 100, suffix: "%", label: "of redirect logic in Go", note: "no Node in the hot path — Next.js serves the dashboard" },
          { value: 0, suffix: "", label: "IP addresses stored", note: "aggregate click analytics only — privacy by design" },
        ].map((s) => (
          <div key={s.label} className="px-2 py-8 text-center">
            <p className="text-3xl font-semibold tracking-tight sm:text-4xl">
              <CountUp value={s.value} suffix={s.suffix} />
            </p>
            <p className="mt-1 text-[13px] font-medium">{s.label}</p>
            <p className="mx-auto mt-1 max-w-[16rem] text-[12px] text-muted">{s.note}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

/* ---------------------------- ticker ---------------------------- */

function Ticker() {
  const items = [
    "POST /api/v1/urls",
    "302 Found",
    "Redis Lua rate limits",
    "Base62 · crypto/rand",
    "pgxpool",
    "409 slug_taken",
    "negative caching",
    "graceful shutdown",
    "Chi router",
    "batched analytics",
    "/health/ready",
    "advisory-lock migrations",
  ];
  const row = [...items, ...items];
  return (
    <section className="overflow-hidden border-b border-border py-3">
      <div className="animate-ticker flex w-max gap-8 font-mono text-[12px] text-muted">
        {row.map((item, i) => (
          <span key={i} className="flex items-center gap-2 whitespace-nowrap">
            <span className="h-1 w-1 rounded-full bg-accent" />
            {item}
          </span>
        ))}
      </div>
    </section>
  );
}

/* --------------------------- features --------------------------- */

const features = [
  {
    title: "Exact custom slugs",
    body: "Claim gauravresumev1 and get exactly that — never gauravresumev1-x7Kp9. Uniqueness enforced by the database, not a SELECT-then-insert race.",
    icon: (
      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
    ),
  },
  {
    title: "µs-scale redirects",
    body: "Redis caches slug → destination with singleflight-coalesced misses. Benchmarked cache-hit resolution: 27µs. Misses fall back to Postgres, never a blank page.",
    icon: <path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z" />,
  },
  {
    title: "Distributed rate limiting",
    body: "Atomic Lua fixed-windows in Redis — correct limits across every replica. Separate policies for creation, auth, analytics, and redirects.",
    icon: (
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
    ),
  },
  {
    title: "Analytics without surveillance",
    body: "Click counts, 30-day series, referrers, and device classes — batched async. Bots are filtered out of click counts. Raw IPs are never stored.",
    icon: (
      <path d="M3 3v18h18M7 14l4-4 4 4 6-6" />
    ),
  },
  {
    title: "QR codes, API keys & CLI",
    body: "Download a QR for any link, issue scoped API keys (create/read/manage), and script everything from the cutsh CLI. Built for automation, not just clicking.",
    icon: (
      <path d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h2v2h-2zM18 14h2v2h-2zM16 16h2v2h-2zM18 18h2v2h-2zM14 18h2v2h-2z" />
    ),
  },
  {
    title: "Production plumbing included",
    body: "SQL migrations with advisory locks, pgxpool, structured JSON logs, request IDs, timeouts, graceful shutdown, /health and /metrics endpoints.",
    icon: (
      <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
    ),
  },
  {
    title: "Yours to run",
    body: "One docker compose up — Postgres, Redis, API, dashboard. Or run the Go binary and Next.js app on your own metal. No vendor can raise your prices.",
    icon: (
      <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z M3.27 6.96 12 12.01l8.73-5.05M12 22.08V12" />
    ),
  },
];

function Features() {
  return (
    <section id="features" className="mx-auto max-w-5xl scroll-mt-16 px-4 py-20 sm:px-6">
      <Reveal>
        <p className="text-[13px] font-medium text-accent">Features</p>
        <h2 className="mt-2 max-w-2xl text-3xl font-semibold tracking-tight text-balance">
          Everything you&apos;d build yourself — already built
        </h2>
        <p className="mt-3 max-w-xl text-[14px] text-muted">
          No mystery middleware, no per-click billing. Just the components a
          senior team would ship for a URL shortener, done properly.
        </p>
      </Reveal>
      <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {features.map((f, i) => (
          <Reveal key={f.title} delay={i * 70}>
            <Card className="h-full p-5 transition-colors hover:border-border-strong">
              <div className="mb-3 flex h-9 w-9 items-center justify-center rounded-md border border-border bg-background text-accent">
                <svg
                  className="h-4 w-4"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  aria-hidden
                >
                  {f.icon}
                </svg>
              </div>
              <h3 className="text-[14px] font-semibold">{f.title}</h3>
              <p className="mt-1.5 text-[13px] leading-relaxed text-muted">{f.body}</p>
            </Card>
          </Reveal>
        ))}
      </div>
    </section>
  );
}

/* ---------------------------- how it works ---------------------------- */

const steps = [
  {
    n: "01",
    title: "Paste any URL",
    body: "http(s) only — javascript: and data: URLs are rejected before they reach the database.",
  },
  {
    n: "02",
    title: "Choose your slug",
    body: "Type an exact custom slug, or let crypto/rand generate unbiased Base62 — with an optional prefix like resume-.",
  },
  {
    n: "03",
    title: "Share and measure",
    body: "302 instantly from the Redis cache. Watch clicks, referrers, and devices in the dashboard — no IPs collected.",
  },
];

function HowItWorks() {
  return (
    <section id="how" className="border-y border-border">
      <div className="mx-auto max-w-5xl scroll-mt-16 px-4 py-20 sm:px-6">
        <Reveal>
          <p className="text-[13px] font-medium text-accent">How it works</p>
          <h2 className="mt-2 text-3xl font-semibold tracking-tight">Three steps, zero surprises</h2>
        </Reveal>
        <div className="mt-10 grid gap-6 sm:grid-cols-3">
          {steps.map((s, i) => (
            <Reveal key={s.n} delay={i * 90}>
              <div className="relative">
                <span className="font-mono text-[12px] text-accent">{s.n}</span>
                <h3 className="mt-2 text-[15px] font-semibold">{s.title}</h3>
                <p className="mt-1.5 text-[13px] leading-relaxed text-muted">{s.body}</p>
              </div>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ---------------------------- use cases ---------------------------- */

const useCases = [
  {
    title: "Job applications & referrals",
    body: "Applying via a referral? Use the job id or role as the slug — cut.sh/google-pm-referral, cut.sh/nvidia-swe-2026. Paste it in forms and messages; sign in later to see which applications people actually open.",
    example: "cut.sh/google-pm-referral",
  },
  {
    title: "One resume, many slugs",
    body: "cut.sh/gaurav-resume-pm vs cut.sh/gaurav-resume-swe — same PDF, tailored per role. Print the slug on the resume itself; analytics attributes every view. Sign in and you can even swap the destination later without reprinting.",
    example: "cut.sh/gaurav-resume-pm",
  },
  {
    title: "Posts, launches & bio links",
    body: "cut.sh/hn-launch, cut.sh/ig-bio — names people can speak on a podcast, type from a video, or scan from a QR code. No random gibberish to mistype.",
    example: "cut.sh/hn-launch",
  },
  {
    title: "Per-channel attribution",
    body: "Give each channel its own meaningful slug. Signed-in analytics break down clicks by slug, referrer, and device — so you know whether the referral, the résumé blast, or the bio link worked.",
    example: "cut.sh/newsletter-cta",
  },
];

function UseCases() {
  return (
    <section id="use-cases" className="mx-auto max-w-5xl scroll-mt-16 px-4 py-20 sm:px-6">
      <Reveal>
        <p className="text-[13px] font-medium text-accent">Why custom slugs</p>
        <h2 className="mt-2 max-w-2xl text-3xl font-semibold tracking-tight text-balance">
          A slug is a label — make it mean something
        </h2>
        <p className="mt-3 max-w-xl text-[14px] text-muted">
          Random strings are forgettable. Meaningful slugs work like real
          names: speakable, printable, and trackable. Create them anonymously —
          sign in when you want the numbers behind them.
        </p>
      </Reveal>
      <div className="mt-10 grid gap-4 sm:grid-cols-2">
        {useCases.map((c, i) => (
          <Reveal key={c.title} delay={i * 80}>
            <Card className="h-full p-5">
              <span className="font-mono text-[12px] font-medium text-accent">
                {c.example}
              </span>
              <h3 className="mt-2 text-[14px] font-semibold">{c.title}</h3>
              <p className="mt-1.5 text-[13px] leading-relaxed text-muted">{c.body}</p>
            </Card>
          </Reveal>
        ))}
      </div>
      <Reveal delay={200}>
        <p className="mt-8 text-center text-[13px] text-muted">
          Try it above — no login to create.{" "}
          <Link href="/dashboard" className="font-medium text-accent hover:underline">
            Sign in
          </Link>{" "}
          for custom slugs, click analytics, edit &amp; delete.
        </p>
      </Reveal>
    </section>
  );
}

/* ---------------------------- comparison ---------------------------- */

const compareRows: { label: string; us: string; them: string }[] = [
  { label: "Exact custom slugs", us: "Free, verbatim", them: "Usually a paid tier" },
  { label: "Where redirect data lives", us: "Your Postgres", them: "Vendor's servers" },
  { label: "Rate limits", us: "You configure every policy", them: "Vendor-set, opaque" },
  { label: "Redirect-page ads", us: "Never", them: "Common on free plans" },
  { label: "Analytics privacy", us: "Aggregates only, no IPs", them: "Often visitor-level tracking" },
  { label: "Full REST API", us: "Included, unlimited by us", them: "Plan-gated, rate-capped" },
  { label: "Vendor lock-in", us: "None — it's your stack", them: "Export at best" },
  { label: "Cost", us: "Your infrastructure only", them: "Per-seat + per-link tiers" },
];

function Compare() {
  return (
    <section id="compare" className="mx-auto max-w-5xl scroll-mt-16 px-4 py-20 sm:px-6">
      <Reveal>
        <p className="text-[13px] font-medium text-accent">Compare</p>
        <h2 className="mt-2 max-w-2xl text-3xl font-semibold tracking-tight text-balance">
          Built for developers who outgrew hosted shorteners
        </h2>
        <p className="mt-3 max-w-xl text-[14px] text-muted">
          A generalized comparison against typical hosted services (Bitly-style).
          Your mileage may vary — the difference is ownership.
        </p>
      </Reveal>
      <Reveal delay={100}>
        <Card className="mt-10 overflow-hidden">
          <table className="w-full text-left text-[13px]">
            <thead>
              <tr className="border-b border-border">
                <th className="px-4 py-3 font-medium text-muted sm:px-5">Capability</th>
                <th className="px-4 py-3 font-medium sm:px-5">
                  cut<span className="text-muted">.sh</span>{" "}
                  <span className="ml-1 rounded-full bg-accent/10 px-2 py-0.5 text-[11px] font-medium text-accent">
                    self-hosted
                  </span>
                </th>
                <th className="px-4 py-3 font-medium text-muted sm:px-5">Hosted shorteners</th>
              </tr>
            </thead>
            <tbody>
              {compareRows.map((r) => (
                <tr key={r.label} className="border-b border-border last:border-0">
                  <td className="px-4 py-3 text-muted sm:px-5">{r.label}</td>
                  <td className="px-4 py-3 font-medium sm:px-5">
                    <span className="mr-1.5 text-success" aria-hidden>
                      ✓
                    </span>
                    {r.us}
                  </td>
                  <td className="px-4 py-3 text-muted sm:px-5">{r.them}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      </Reveal>
    </section>
  );
}

/* ------------------------------ self host ------------------------------ */

function SelfHost() {
  return (
    <section id="self-host" className="border-t border-border">
      <div className="mx-auto max-w-5xl scroll-mt-16 px-4 py-20 sm:px-6">
        <div className="grid items-center gap-10 lg:grid-cols-2">
          <Reveal>
            <p className="text-[13px] font-medium text-accent">Self-host</p>
            <h2 className="mt-2 text-3xl font-semibold tracking-tight">
              One command to production-shaped infra
            </h2>
            <p className="mt-3 text-[14px] leading-relaxed text-muted">
              docker compose up starts Postgres, Redis, the Go API, and the
              dashboard — health-checked, migration-gated, secrets via env.
              Prefer native? <code className="rounded bg-code-bg px-1.5 py-0.5 font-mono text-[12px]">go run ./cmd/api</code>{" "}
              and <code className="rounded bg-code-bg px-1.5 py-0.5 font-mono text-[12px]">npm run dev</code> work too.
            </p>
            <ul className="mt-5 space-y-2 text-[13px] text-muted">
              {[
                "Postgres and Redis never exposed publicly by default",
                "Migrations run from embedded, ordered SQL files",
                "Scale the stateless API horizontally — limits and cache are already shared",
              ].map((t) => (
                <li key={t} className="flex gap-2">
                  <span className="text-success" aria-hidden>
                    ✓
                  </span>
                  {t}
                </li>
              ))}
            </ul>
            <Link href="/dashboard" className={cn(buttonClass("primary"), "mt-7")}>
              Open the dashboard
            </Link>
          </Reveal>
          <Reveal delay={120}>
            <div className="rounded-lg border border-border bg-[#0a0a0a] p-5 font-mono text-[12.5px] leading-relaxed text-[#ededed] shadow-2xl">
              <div className="mb-3 flex gap-1.5" aria-hidden>
                <span className="h-2.5 w-2.5 rounded-full bg-[#333]" />
                <span className="h-2.5 w-2.5 rounded-full bg-[#333]" />
                <span className="h-2.5 w-2.5 rounded-full bg-[#333]" />
              </div>
              <p>
                <span className="text-[#666]">$</span> git clone &lt;repo&gt; && cd url-shortner
              </p>
              <p>
                <span className="text-[#666]">$</span> cp .env.example .env
              </p>
              <p>
                <span className="text-[#666]">$</span> docker compose up -d --build
              </p>
              <p className="mt-2 text-[#666]">✓ db healthy · redis healthy · migrate ok</p>
              <p className="text-[#666]">✓ api healthy · web started</p>
              <p className="mt-2">
                <span className="text-[#666]">$</span> curl -X POST localhost:8080/api/v1/urls \
              </p>
              <p className="pl-8">-H &apos;Content-Type: application/json&apos; \</p>
              <p className="pl-8">
                -d &apos;{`{"original_url":"https://example.com","custom_slug":"launch"}`}&apos;
              </p>
              <p className="mt-2 animate-shimmer rounded bg-gradient-to-r from-[#1a1a1a] via-[#2a2a2a] to-[#1a1a1a] px-2 py-1 text-[#ededed]">
                → 201 Created · cut.sh/launch
              </p>
            </div>
          </Reveal>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------- CTA ------------------------------- */

function CTA() {
  return (
    <section className="relative overflow-hidden border-t border-border">
      <div className="relative mx-auto max-w-3xl px-4 py-24 text-center sm:px-6">
        <Reveal>
          <h2 className="text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            Own your links. Own your data.
          </h2>
          <p className="mx-auto mt-4 max-w-md text-[14px] text-muted">
            Spin up the stack, claim your slugs, and stop renting basic
            infrastructure from ad-supported shorteners.
          </p>
          <div className="mt-8 flex justify-center gap-3">
            <Link
              href="/dashboard"
              className={cn(buttonClass("primary", "md"), "h-10 px-6")}
            >
              Get started — it&apos;s free to run
            </Link>
          </div>
        </Reveal>
      </div>
    </section>
  );
}

/* ------------------------------ footer ------------------------------ */

function Footer() {
  return (
    <footer className="border-t border-border">
      <div className="mx-auto flex max-w-5xl flex-col items-center justify-between gap-3 px-4 py-8 text-[12px] text-muted sm:flex-row sm:px-6">
        <div className="flex items-center gap-2">
          <Logo />
          <span className="ml-2">· production-ready URL shortener</span>
        </div>
        <div className="flex items-center gap-4">
          <a href="#features" className="hover:text-foreground">
            Features
          </a>
          <a href="#compare" className="hover:text-foreground">
            Compare
          </a>
          <Link href="/dashboard" className="hover:text-foreground">
            Dashboard
          </Link>
        </div>
      </div>
    </footer>
  );
}

/* ----------------------------- page ----------------------------- */

export function LandingPage() {
  return (
    <div className="flex min-h-screen flex-col">
      <Nav />
      <main className="flex-1">
        <ParallaxShowcase />
        <Hero />
        <Ticker />
        <Stats />
        <Features />
        <HowItWorks />
        <UseCases />
        <Compare />
        <SelfHost />
        <CTA />
      </main>
      <Footer />
    </div>
  );
}
