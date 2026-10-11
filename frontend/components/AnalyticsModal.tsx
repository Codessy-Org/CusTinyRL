"use client";

import { useEffect, useState } from "react";
import { api } from "../lib/api";
import type { AnalyticsSummary } from "../lib/types";
import { Button, Card, Spinner } from "./ui";

function BarChart({ series }: { series: { date: string; count: number }[] }) {
  const max = Math.max(1, ...series.map((p) => p.count));
  return (
    <div className="flex h-28 items-end gap-1" role="img" aria-label="Clicks per day">
      {series.map((p) => (
        <div key={p.date} className="group relative flex-1">
          <div
            className="w-full rounded-sm bg-accent/70 transition-colors group-hover:bg-accent"
            style={{ height: `${Math.max(2, (p.count / max) * 100)}%` }}
          />
          <span className="pointer-events-none absolute -top-6 left-1/2 hidden -translate-x-1/2 whitespace-nowrap rounded bg-foreground px-1.5 py-0.5 text-[10px] text-background group-hover:block">
            {p.date}: {p.count}
          </span>
        </div>
      ))}
      {series.length === 0 && (
        <p className="self-center text-[13px] text-muted">No clicks in this period.</p>
      )}
    </div>
  );
}

export function AnalyticsModal({ urlId, slug, onClose }: { urlId: number; slug: string; onClose: () => void }) {
  const [data, setData] = useState<AnalyticsSummary | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let alive = true;
    api<AnalyticsSummary>(`/api/v1/urls/${urlId}/analytics?days=30`)
      .then((s) => {
        if (alive) setData(s);
      })
      .catch((e) => {
        if (alive) setError(e.message ?? "Failed to load analytics");
      });
    return () => {
      alive = false;
    };
  }, [urlId]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-overlay p-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={`Analytics for ${slug}`}
    >
      <Card className="animate-in w-full max-w-lg p-5">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-[15px] font-semibold">Analytics</h2>
            <p className="font-mono text-[12px] text-muted">/{slug}</p>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close analytics">
            ✕
          </Button>
        </div>

        {error && <p className="text-[13px] text-danger">{error}</p>}
        {!data && !error && (
          <div className="flex justify-center py-10">
            <Spinner />
          </div>
        )}
        {data && (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-3">
              <div className="rounded-md border border-border p-3">
                <p className="text-[12px] text-muted">Total clicks</p>
                <p className="text-2xl font-semibold tabular-nums">{data.total_clicks}</p>
              </div>
              <div className="rounded-md border border-border p-3">
                <p className="text-[12px] text-muted">Last 24 hours</p>
                <p className="text-2xl font-semibold tabular-nums">{data.clicks_24h}</p>
              </div>
            </div>

            <div>
              <h3 className="mb-2 text-[13px] font-medium text-muted">Last 30 days</h3>
              <BarChart series={data.series} />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <h3 className="mb-2 text-[13px] font-medium text-muted">Top referrers</h3>
                <ul className="space-y-1 text-[13px]">
                  {data.top_referrers.length === 0 && <li className="text-muted">None yet</li>}
                  {data.top_referrers.map((r) => (
                    <li key={r.label} className="flex justify-between gap-2">
                      <span className="truncate">{r.label}</span>
                      <span className="tabular-nums text-muted">{r.count}</span>
                    </li>
                  ))}
                </ul>
              </div>
              <div>
                <h3 className="mb-2 text-[13px] font-medium text-muted">Devices</h3>
                <ul className="space-y-1 text-[13px]">
                  {data.devices.length === 0 && <li className="text-muted">None yet</li>}
                  {data.devices.map((d) => (
                    <li key={d.label} className="flex justify-between gap-2">
                      <span>{d.label}</span>
                      <span className="tabular-nums text-muted">{d.count}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
