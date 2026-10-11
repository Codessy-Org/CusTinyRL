"use client";

import { formatCount } from "../lib/format";
import type { URLItem } from "../lib/types";
import { Card } from "./ui";

export function StatsCards({ items }: { items: URLItem[] }) {
  const totalLinks = items.length;
  const activeLinks = items.filter((i) => i.is_active && !i.is_expired).length;
  const totalClicks = items.reduce((sum, i) => sum + i.click_count, 0);

  const stats = [
    { label: "Links", value: formatCount(totalLinks) },
    { label: "Active", value: formatCount(activeLinks) },
    { label: "Total clicks", value: formatCount(totalClicks) },
  ];

  return (
    <div className="grid grid-cols-3 gap-3">
      {stats.map((s) => (
        <Card key={s.label} className="px-4 py-3">
          <p className="text-[12px] text-muted">{s.label}</p>
          <p className="mt-0.5 text-xl font-semibold tabular-nums tracking-tight sm:text-2xl">
            {s.value}
          </p>
        </Card>
      ))}
    </div>
  );
}
