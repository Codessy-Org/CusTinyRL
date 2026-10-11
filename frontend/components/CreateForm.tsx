"use client";

import { FormEvent, useState } from "react";
import { api, ApiError } from "../lib/api";
import { isValidHttpUrl, PREFIX_PATTERN, SLUG_PATTERN } from "../lib/format";
import type { CreateURLPayload, URLItem } from "../lib/types";
import { useToast } from "./Toast";
import { Button, Card, Input, Label, Spinner, Switch, cn } from "./ui";

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

export function CopyButton({ value, small }: { value: string; small?: boolean }) {
  const [copied, setCopied] = useState(false);
  const { push } = useToast();

  return (
    <Button
      size={small ? "sm" : "md"}
      variant="ghost"
      aria-label="Copy short URL"
      onClick={async () => {
        const ok = await copyText(value);
        if (ok) {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        } else {
          push("Clipboard access denied", "error");
        }
      }}
    >
      {copied ? (
        <svg className="h-3.5 w-3.5 text-success" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
          <path d="M20 6 9 17l-5-5" />
        </svg>
      ) : (
        <svg className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
          <rect x="9" y="9" width="13" height="13" rx="2" />
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
        </svg>
      )}
      {copied ? "Copied" : "Copy"}
    </Button>
  );
}

export function CreateForm({ onCreated }: { onCreated: (item: URLItem) => void }) {
  const [mode, setMode] = useState<"auto" | "custom">("auto");
  const [url, setUrl] = useState("");
  const [slug, setSlug] = useState("");
  const [prefix, setPrefix] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [conflict, setConflict] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [created, setCreated] = useState<URLItem | null>(null);

  function validate(): boolean {
    const next: Record<string, string> = {};
    if (!url.trim()) {
      next.url = "Paste a URL to shorten.";
    } else if (!isValidHttpUrl(url.trim())) {
      next.url = "Enter a valid http(s) URL, e.g. https://example.com/page";
    }
    if (mode === "custom") {
      const s = slug.trim();
      if (!s) {
        next.slug = "Choose a slug or switch to automatic.";
      } else if (!SLUG_PATTERN.test(s)) {
        next.slug =
          "Lowercase letters, numbers, hyphens, underscores; max 64 chars; must start alphanumeric.";
      }
    }
    if (mode === "auto" && prefix.trim() && !PREFIX_PATTERN.test(prefix.trim())) {
      next.prefix = "Lowercase letters, numbers, hyphens, underscores; max 32 chars.";
    }
    setErrors(next);
    return Object.keys(next).length === 0;
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    setConflict(null);
    setSubmitError(null);
    setCreated(null);
    if (!validate()) return;

    const payload: CreateURLPayload = { original_url: url.trim() };
    if (mode === "custom") payload.custom_slug = slug.trim();
    if (mode === "auto" && prefix.trim()) payload.prefix = prefix.trim();
    if (expiresAt) payload.expires_at = new Date(expiresAt).toISOString();

    setLoading(true);
    try {
      const item = await api<URLItem>("/api/v1/urls", { method: "POST", body: payload });
      setCreated(item);
      setUrl("");
      setSlug("");
      setPrefix("");
      setExpiresAt("");
      onCreated(item);
    } catch (err) {
      if (err instanceof ApiError && err.code === "slug_taken") {
        setConflict(
          `The slug "${slug.trim()}" is already taken. Pick a different one — your input is preserved.`,
        );
        setErrors({ slug: "This slug is unavailable." });
      } else if (err instanceof ApiError) {
        setSubmitError(err.message);
      } else {
        setSubmitError("Something went wrong. Try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card className="p-4 sm:p-5">
      <form onSubmit={submit} noValidate>
        <div className="flex flex-col gap-3 sm:flex-row">
          <div className="flex-1">
            <Label htmlFor="dest-url">Destination URL</Label>
            <Input
              id="dest-url"
              type="url"
              inputMode="url"
              placeholder="https://docs.google.com/document/d/example-document-id"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              aria-invalid={Boolean(errors.url)}
              aria-describedby={errors.url ? "dest-url-error" : undefined}
            />
            {errors.url && (
              <p id="dest-url-error" role="alert" className="mt-1.5 text-[12px] text-danger">
                {errors.url}
              </p>
            )}
          </div>
        </div>

        <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex items-center gap-2 pb-0.5">
            <span className="text-[13px] text-muted">Slug</span>
            <Switch
              checked={mode === "custom"}
              onChange={(v) => {
                setMode(v ? "custom" : "auto");
                setErrors({});
                setConflict(null);
              }}
              label="Toggle custom slug mode"
            />
            <span className="text-[13px] font-medium">
              {mode === "custom" ? "Custom" : "Automatic"}
            </span>
          </div>

          {mode === "custom" ? (
            <div className="flex-1">
              <Input
                aria-label="Custom slug"
                placeholder="gauravresumev1"
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
                className={cn("font-mono", errors.slug && "border-danger")}
                aria-invalid={Boolean(errors.slug)}
              />
              {errors.slug && (
                <p role="alert" className="mt-1.5 text-[12px] text-danger">
                  {errors.slug}
                </p>
              )}
            </div>
          ) : (
            <div className="flex-1">
              <Input
                aria-label="Prefix for generated slug"
                placeholder="Prefix (optional), e.g. resume"
                value={prefix}
                onChange={(e) => setPrefix(e.target.value)}
                className={cn("font-mono", errors.prefix && "border-danger")}
              />
              {errors.prefix && (
                <p role="alert" className="mt-1.5 text-[12px] text-danger">
                  {errors.prefix}
                </p>
              )}
            </div>
          )}

          <div className="sm:w-48">
            <Label htmlFor="expires">Expires (optional)</Label>
            <Input
              id="expires"
              type="datetime-local"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              className="text-[13px]"
            />
          </div>

          <Button type="submit" variant="primary" disabled={loading} className="sm:w-auto">
            {loading && <Spinner className="h-3.5 w-3.5" />}
            Shorten
          </Button>
        </div>

        {conflict && (
          <div
            role="alert"
            className="mt-3 rounded-md border border-danger/40 bg-danger/5 px-3 py-2 text-[13px] text-danger"
          >
            {conflict}
          </div>
        )}
        {submitError && (
          <div role="alert" className="mt-3 text-[13px] text-danger">
            {submitError}
          </div>
        )}
        {created && (
          <div className="mt-3 flex flex-wrap items-center gap-2 rounded-md border border-border bg-background px-3 py-2 text-[13px]">
            <span className="text-muted">Created</span>
            <a
              href={created.short_url}
              target="_blank"
              rel="noreferrer"
              className="font-mono font-medium text-accent hover:underline"
            >
              {created.short_url.replace(/^https?:\/\//, "")}
            </a>
            <CopyButton value={created.short_url} small />
          </div>
        )}
      </form>
    </Card>
  );
}
