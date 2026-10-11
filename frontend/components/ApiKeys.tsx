"use client";

import { useCallback, useEffect, useState } from "react";
import { api } from "../lib/api";
import type { ApiKey, ApiKeyCreateResponse, ApiKeyScope } from "../lib/types";
import { Button, Card, Input, Label, Spinner } from "./ui";

const SCOPE_OPTIONS: { value: ApiKeyScope; label: string; hint: string }[] = [
  { value: "create", label: "create", hint: "POST /urls" },
  { value: "read", label: "read", hint: "GET /urls + analytics" },
  { value: "manage", label: "manage", hint: "PATCH/DELETE /urls" },
];

function fmtDate(iso?: string | null) {
  if (!iso) return "—";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleDateString();
}

/**
 * API key management (session-only). Keys are scoped credentials for CLIs
 * and scripts; the full secret is shown exactly once at creation.
 */
export function ApiKeys() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [name, setName] = useState("");
  const [scopes, setScopes] = useState<ApiKeyScope[]>(["create", "read"]);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newKey, setNewKey] = useState<ApiKeyCreateResponse | null>(null);
  const [copied, setCopied] = useState(false);

  const load = useCallback(() => {
    return api<{ data: ApiKey[] }>("/api/v1/keys")
      .then((res) => {
        setKeys(res.data);
        setError(null);
      })
      .catch(() => setError("Failed to load API keys."))
      .finally(() => setLoaded(true));
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  function toggleScope(s: ApiKeyScope) {
    setScopes((prev) => (prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]));
  }

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    if (!name.trim() || scopes.length === 0) {
      setError("Name and at least one scope are required.");
      return;
    }
    setCreating(true);
    try {
      const res = await api<ApiKeyCreateResponse>("/api/v1/keys", {
        method: "POST",
        body: { name: name.trim(), scopes },
      });
      setNewKey(res);
      setName("");
      setCopied(false);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create key.");
    } finally {
      setCreating(false);
    }
  }

  async function revoke(id: number) {
    setError(null);
    try {
      await api(`/api/v1/keys/${id}`, { method: "DELETE" });
      await load();
    } catch {
      setError("Failed to revoke key.");
    }
  }

  return (
    <Card className="p-4 sm:p-5">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-[14px] font-semibold">API keys</h2>
          <p className="mt-0.5 text-[12px] text-muted">
            Scoped tokens for the CLI and scripts. Session-only management — keys cannot mint keys.
          </p>
        </div>
      </div>

      <form onSubmit={create} className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="flex-1">
          <Label htmlFor="key-name">Name</Label>
          <Input
            id="key-name"
            placeholder="ci-pipeline"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <fieldset className="flex gap-3">
          <legend className="mb-1 text-[13px] font-medium">Scopes</legend>
          {SCOPE_OPTIONS.map((s) => (
            <label key={s.value} className="flex cursor-pointer items-center gap-1.5 text-[13px]">
              <input
                type="checkbox"
                checked={scopes.includes(s.value)}
                onChange={() => toggleScope(s.value)}
                className="accent-[var(--accent)]"
              />
              {s.label}
            </label>
          ))}
        </fieldset>
        <Button type="submit" variant="primary" disabled={creating}>
          {creating && <Spinner className="h-3.5 w-3.5" />}
          Create key
        </Button>
      </form>

      {error && (
        <p role="alert" className="mt-3 text-[13px] text-danger">
          {error}
        </p>
      )}

      {newKey && (
        <div className="animate-in mt-4 rounded-md border border-accent/40 bg-accent/5 p-3">
          <p className="text-[12px] font-medium text-accent">
            Copy this key now — it is never shown again.
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <code className="max-w-full overflow-x-auto rounded bg-background px-2 py-1 font-mono text-[12px]">
              {newKey.key}
            </code>
            <Button
              size="sm"
              variant="secondary"
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(newKey.key);
                  setCopied(true);
                  setTimeout(() => setCopied(false), 1500);
                } catch {
                  /* selectable above */
                }
              }}
            >
              {copied ? "Copied" : "Copy"}
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setNewKey(null)}>
              Dismiss
            </Button>
          </div>
          <p className="mt-2 font-mono text-[11px] text-muted">
            export CUTSH_TOKEN={newKey.key}
          </p>
        </div>
      )}

      {loaded && keys.length === 0 && !error && (
        <p className="mt-4 text-[13px] text-muted">No API keys yet.</p>
      )}
      {keys.length > 0 && (
        <ul className="mt-4 divide-y divide-border">
          {keys.map((k) => {
            const revoked = Boolean(k.revoked_at);
            return (
              <li
                key={k.id}
                className={`flex flex-wrap items-center justify-between gap-2 py-2.5 text-[13px] ${revoked ? "opacity-50" : ""}`}
              >
                <div>
                  <span className="font-medium">{k.name}</span>
                  <span className="ml-2 rounded-full border border-border px-1.5 py-0.5 font-mono text-[11px] text-muted">
                    {k.key_prefix}…
                  </span>
                  <span className="ml-2 text-[12px] text-muted">{k.scopes.join(", ")}</span>
                </div>
                <div className="flex items-center gap-3 text-[12px] text-muted">
                  <span>used {fmtDate(k.last_used_at)}</span>
                  {revoked ? (
                    <span className="text-danger">revoked</span>
                  ) : (
                    <Button size="sm" variant="ghost" onClick={() => void revoke(k.id)}>
                      Revoke
                    </Button>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}

      <p className="mt-4 rounded-md border border-border bg-background px-3 py-2 font-mono text-[11.5px] text-muted">
        cutsh add https://example.com --slug my-slug
      </p>
    </Card>
  );
}
