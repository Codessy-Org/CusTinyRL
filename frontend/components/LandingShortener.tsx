"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { api, ApiError } from "../lib/api";
import { isValidHttpUrl } from "../lib/format";
import type { CreateURLPayload, URLItem } from "../lib/types";
import { QRModal } from "./QRModal";
import { Button, Card, Input, Label, Spinner } from "./ui";

function LocalCopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <Button
      size="sm"
      variant="ghost"
      aria-label="Copy short URL"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(value);
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        } catch {
          /* clipboard blocked — link is selectable */
        }
      }}
    >
      {copied ? (
        <svg
          className="h-3.5 w-3.5 text-success"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          aria-hidden
        >
          <path d="M20 6 9 17l-5-5" />
        </svg>
      ) : (
        <svg
          className="h-3.5 w-3.5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          aria-hidden
        >
          <rect x="9" y="9" width="13" height="13" rx="2" />
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
        </svg>
      )}
      {copied ? "Copied" : "Copy"}
    </Button>
  );
}

/**
 * Anonymous shortener embedded in the landing hero.
 * POST /api/v1/urls without auth (ALLOW_ANONYMOUS=true, 5/min per IP).
 * Anonymous creates get an automatic slug only — custom slugs and prefixes
 * require an account (the API returns 403 auth_required otherwise).
 * Analytics & management stay behind login.
 */
export function LandingShortener() {
  const [url, setUrl] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [created, setCreated] = useState<URLItem | null>(null);
  const [showQR, setShowQR] = useState(false);

  function validate(): boolean {
    const next: Record<string, string> = {};
    if (!url.trim()) {
      next.url = "Paste a URL to shorten.";
    } else if (!isValidHttpUrl(url.trim())) {
      next.url = "Enter a valid http(s) URL.";
    }
    setErrors(next);
    return Object.keys(next).length === 0;
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    setSubmitError(null);
    setCreated(null);
    if (!validate()) return;

    const payload: CreateURLPayload = { original_url: url.trim() };

    setLoading(true);
    try {
      const item = await api<URLItem>("/api/v1/urls", { method: "POST", body: payload });
      setCreated(item);
      setUrl("");
      setErrors({});
    } catch (err) {
      if (err instanceof ApiError && err.code === "rate_limited") {
        setSubmitError(
          "Anonymous limit reached (5/min per IP). Wait a minute, or sign in for a higher limit.",
        );
      } else if (err instanceof ApiError) {
        setSubmitError(err.message);
      } else {
        setSubmitError("Something went wrong. Try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  if (created) {
    const display = created.short_url.replace(/^https?:\/\//, "");
    return (
      <Card className="animate-in mx-auto w-full max-w-lg p-5 text-left">
        <div className="flex items-center justify-between">
          <span className="text-[12px] font-medium text-success">
            ✓ Link created — no account required
          </span>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              setCreated(null);
            }}
          >
            Shorten another
          </Button>
        </div>
        <a
          href={created.short_url}
          target="_blank"
          rel="noreferrer"
          className="mt-3 block rounded-md border border-border bg-background px-3 py-2.5 font-mono text-[15px] font-medium text-accent hover:border-border-strong"
        >
          {display}
        </a>
        {created.title && (
          <p className="mt-2 flex items-center gap-1.5 text-[12px] text-muted">
            {created.favicon_url && (
              // eslint-disable-next-line @next/next/no-img-element -- remote favicon
              <img src={created.favicon_url} alt="" width={14} height={14} className="h-3.5 w-3.5 rounded-sm" />
            )}
            <span className="truncate">{created.title}</span>
          </p>
        )}
        <div className="mt-3 flex flex-wrap items-center justify-between gap-2 text-[12px] text-muted">
          <div className="flex items-center gap-1">
            <LocalCopyButton value={created.short_url} />
            <Button size="sm" variant="ghost" onClick={() => setShowQR(true)}>
              QR
            </Button>
          </div>
          <span>
            Want clicks, referrers &amp; device analytics?{" "}
            <Link href="/dashboard" className="font-medium text-accent hover:underline">
              Sign in free
            </Link>
          </span>
        </div>
        {showQR && <QRModal url={created.short_url} onClose={() => setShowQR(false)} />}
      </Card>
    );
  }

  return (
    <Card className="mx-auto w-full max-w-lg p-5 text-left">
      <form onSubmit={submit} noValidate>
        <Label htmlFor="landing-url">Paste a long URL</Label>
        <Input
          id="landing-url"
          type="url"
          inputMode="url"
          placeholder="https://docs.google.com/document/d/example"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          aria-invalid={Boolean(errors.url)}
          aria-describedby={errors.url ? "landing-url-error" : undefined}
        />
        {errors.url && (
          <p id="landing-url-error" role="alert" className="mt-1.5 text-[12px] text-danger">
            {errors.url}
          </p>
        )}

        <Button
          type="submit"
          variant="primary"
          disabled={loading}
          className="mt-4 w-full"
        >
          {loading && <Spinner className="h-3.5 w-3.5" />}
          Shorten it — no login needed
        </Button>

        {submitError && (
          <div
            role="alert"
            className="mt-3 rounded-md border border-danger/40 bg-danger/5 px-3 py-2 text-[13px] text-danger"
          >
            {submitError}
          </div>
        )}

        <p className="mt-3 text-center text-[12px] text-muted">
          Works anonymously (5 links/min).{" "}
          <Link href="/dashboard" className="text-accent hover:underline">
            Sign in
          </Link>{" "}
          for custom slugs, analytics, edit &amp; delete.
        </p>
      </form>
    </Card>
  );
}
