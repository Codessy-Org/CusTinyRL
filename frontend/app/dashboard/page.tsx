"use client";

import { useCallback, useEffect, useState, useSyncExternalStore } from "react";
import { api, getToken, setToken, subscribeToken } from "../../lib/api";
import type { URLItem, URLListResponse } from "../../lib/types";
import { AuthPanel } from "../../components/AuthPanel";
import { ApiKeys } from "../../components/ApiKeys";
import { CreateForm } from "../../components/CreateForm";
import { Header } from "../../components/Header";
import { StatsCards } from "../../components/StatsCards";
import { ToastProvider } from "../../components/Toast";
import { UrlTable } from "../../components/UrlTable";
import { Button, Input } from "../../components/ui";

function Dashboard() {
  // null = server snapshot (pre-hydration); "" = signed out; "<token>" = signed in.
  const token = useSyncExternalStore(
    subscribeToken,
    () => getToken() ?? "",
    () => null,
  );
  const [items, setItems] = useState<URLItem[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [cursor, setCursor] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const [showKeys, setShowKeys] = useState(false);

  const load = useCallback((opts: { cursor?: string | null; q?: string; append?: boolean } = {}) => {
    const params = new URLSearchParams({ limit: "20" });
    if (opts.cursor) params.set("cursor", opts.cursor);
    if (opts.q) params.set("q", opts.q);
    return api<URLListResponse>(`/api/v1/urls?${params}`)
      .then((res) => {
        setItems((prev) => (opts.append ? [...prev, ...res.data] : res.data));
        setCursor(res.next_cursor);
        setHasMore(Boolean(res.next_cursor));
        setError(null);
        setLoaded(true);
      })
      .catch((e) => {
        setError(e instanceof Error ? e.message : "Failed to load links.");
        setLoaded(true);
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (token) {
      void load();
    }
  }, [token, load]);

  function logout() {
    api("/api/v1/auth/logout", { method: "POST" }).catch(() => {});
    setToken(null);
    setItems([]);
  }

  if (token === null) {
    return (
      <>
        <Header />
        <div className="flex min-h-[70vh] items-center justify-center">
          <p className="text-[13px] text-muted">Loading…</p>
        </div>
      </>
    );
  }

  if (!token) {
    return (
      <>
        <Header />
        <AuthPanel onAuthed={() => load()} />
      </>
    );
  }

  return (
    <>
      <Header onLogout={logout} />
      <div className="mx-auto w-full max-w-5xl space-y-4 px-4 py-6 sm:px-6">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Dashboard</h1>
          <p className="mt-0.5 text-[13px] text-muted">
            Create, manage, and measure your short links.
          </p>
        </div>

        <StatsCards items={items} />

        <CreateForm
          onCreated={(item) => {
            setItems((prev) => [item, ...prev]);
            setQuery("");
            setSearch("");
          }}
        />

        <div className="flex items-center gap-2">
          <div className="flex-1">
            <Input
              aria-label="Search links"
              placeholder="Search by slug or destination…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  setLoading(true);
                  setSearch(query);
                  load({ q: query });
                }
              }}
            />
          </div>
          <Button
            variant="secondary"
            onClick={() => {
              setLoading(true);
              setSearch(query);
              load({ q: query });
            }}
          >
            Search
          </Button>
          {search && (
            <Button
              variant="ghost"
              onClick={() => {
                setQuery("");
                setSearch("");
                load();
              }}
            >
              Clear
            </Button>
          )}
        </div>

        <UrlTable
          items={items}
          loading={loading || !loaded}
          error={error}
          onChanged={(updated) =>
            setItems((prev) => prev.map((i) => (i.id === updated.id ? updated : i)))
          }
          onDeleted={(id) => setItems((prev) => prev.filter((i) => i.id !== id))}
        />

        {hasMore && !search && (
          <div className="flex justify-center">
            <Button
              variant="secondary"
              disabled={loading}
              onClick={() => {
                setLoading(true);
                load({ cursor, append: true });
              }}
            >
              Load more
            </Button>
          </div>
        )}

        <div className="pt-4">
          <Button variant="ghost" size="sm" onClick={() => setShowKeys((v) => !v)}>
            {showKeys ? "Hide" : "Show"} API keys &amp; CLI
          </Button>
          {showKeys && (
            <div className="mt-3">
              <ApiKeys />
            </div>
          )}
        </div>
      </div>
    </>
  );
}

export default function Page() {
  return (
    <ToastProvider>
      <main className="flex min-h-screen flex-col">
        <Dashboard />
      </main>
      <footer className="border-t border-border py-6">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 text-[12px] text-muted sm:px-6">
          <span>cut.sh — production-ready URL shortener</span>
          <span className="hidden sm:inline">Next.js · Go · PostgreSQL · Redis</span>
        </div>
      </footer>
    </ToastProvider>
  );
}
