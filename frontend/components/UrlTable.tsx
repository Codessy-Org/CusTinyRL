"use client";

import { useState } from "react";
import { api, ApiError } from "../lib/api";
import { formatDate, formatCount, formatRelative } from "../lib/format";
import type { URLItem } from "../lib/types";
import { AnalyticsModal } from "./AnalyticsModal";
import { CopyButton } from "./CreateForm";
import { EditModal } from "./EditModal";
import { QRModal } from "./QRModal";
import { useToast } from "./Toast";
import { Badge, Button, Card } from "./ui";

function ConfirmDelete({ item, onCancel, onConfirm }: { item: URLItem; onCancel: () => void; onConfirm: () => void }) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-overlay p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Confirm delete"
      onClick={onCancel}
    >
      <Card className="animate-in w-full max-w-sm p-5" onClick={(e) => e.stopPropagation()}>
        <h2 className="text-[15px] font-semibold">Delete short link?</h2>
        <p className="mt-2 text-[13px] text-muted">
          <span className="font-mono">/{item.slug}</span> and its analytics will be permanently
          removed. This cannot be undone.
        </p>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
          <Button variant="danger" onClick={onConfirm}>
            Delete
          </Button>
        </div>
      </Card>
    </div>
  );
}

function StatusBadge({ item }: { item: URLItem }) {
  if (!item.is_active) return <Badge tone="danger">Disabled</Badge>;
  if (item.is_expired) return <Badge tone="warning">Expired</Badge>;
  return <Badge tone="success">Active</Badge>;
}

function Destination({ item }: { item: URLItem }) {
  return (
    <div className="flex max-w-[280px] items-center gap-2">
      {item.favicon_url ? (
        // eslint-disable-next-line @next/next/no-img-element -- remote favicon, unoptimized
        <img
          src={item.favicon_url}
          alt=""
          width={16}
          height={16}
          className="h-4 w-4 shrink-0 rounded-sm"
          loading="lazy"
          onError={(e) => {
            (e.target as HTMLImageElement).style.display = "none";
          }}
        />
      ) : (
        <span aria-hidden className="h-4 w-4 shrink-0 rounded-sm border border-border" />
      )}
      <div className="min-w-0">
        {item.title && (
          <p className="truncate text-[13px] font-medium text-foreground" title={item.original_url}>
            {item.title}
          </p>
        )}
        <p className={`truncate text-[12px] text-muted ${item.title ? "" : "text-[13px]"}`} title={item.original_url}>
          {item.original_url}
        </p>
      </div>
    </div>
  );
}

function Actions({
  item,
  onEdit,
  onAnalytics,
  onQR,
  onDelete,
}: {
  item: URLItem;
  onEdit: () => void;
  onAnalytics: () => void;
  onQR: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="flex items-center justify-end gap-0.5 whitespace-nowrap">
      <CopyButton value={item.short_url} small />
      <Button size="sm" variant="ghost" onClick={onQR} aria-label={`QR code for /${item.slug}`}>
        QR
      </Button>
      <Button size="sm" variant="ghost" onClick={onAnalytics}>
        Stats
      </Button>
      <Button size="sm" variant="ghost" onClick={onEdit}>
        Edit
      </Button>
      <Button
        size="sm"
        variant="ghost"
        onClick={onDelete}
        aria-label={`Delete /${item.slug}`}
        className="text-danger hover:text-danger"
      >
        Delete
      </Button>
    </div>
  );
}

export function UrlTable({
  items,
  loading,
  error,
  onChanged,
  onDeleted,
}: {
  items: URLItem[];
  loading: boolean;
  error: string | null;
  onChanged: (item: URLItem) => void;
  onDeleted: (id: number) => void;
}) {
  const [editing, setEditing] = useState<URLItem | null>(null);
  const [analyzing, setAnalyzing] = useState<URLItem | null>(null);
  const [qrItem, setQrItem] = useState<URLItem | null>(null);
  const [deleting, setDeleting] = useState<URLItem | null>(null);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const { push } = useToast();

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteLoading(true);
    try {
      await api(`/api/v1/urls/${deleting.id}`, { method: "DELETE" });
      onDeleted(deleting.id);
      push("Link deleted", "success");
      setDeleting(null);
    } catch (err) {
      push(err instanceof ApiError ? err.message : "Failed to delete link", "error");
    } finally {
      setDeleteLoading(false);
    }
  }

  if (loading && items.length === 0) {
    return (
      <Card className="flex items-center justify-center py-16">
        <p className="text-[13px] text-muted">Loading links…</p>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="flex flex-col items-center gap-3 py-16">
        <p className="text-[13px] text-danger">{error}</p>
      </Card>
    );
  }

  if (items.length === 0) {
    return (
      <Card className="flex flex-col items-center gap-1 py-16">
        <p className="text-sm font-medium">No links yet</p>
        <p className="text-[13px] text-muted">Create your first short URL above.</p>
      </Card>
    );
  }

  return (
    <>
      {/* Desktop table */}
      <Card className="hidden overflow-x-auto md:block">
        <table className="w-full min-w-[720px] text-left text-[13px]">
          <thead>
            <tr className="border-b border-border text-[12px] uppercase tracking-wide text-muted">
              <th className="whitespace-nowrap px-4 py-2.5 font-medium">Short link</th>
              <th className="px-4 py-2.5 font-medium">Destination</th>
              <th className="whitespace-nowrap px-4 py-2.5 font-medium">Clicks</th>
              <th className="whitespace-nowrap px-4 py-2.5 font-medium">Created</th>
              <th className="whitespace-nowrap px-4 py-2.5 font-medium">Expires</th>
              <th className="whitespace-nowrap px-4 py-2.5 font-medium">Status</th>
              <th className="whitespace-nowrap px-4 py-2.5" />
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id} className="border-b border-border last:border-0 hover:bg-card-hover">
                <td className="px-4 py-2.5">
                  <a
                    href={item.short_url}
                    target="_blank"
                    rel="noreferrer"
                    className="font-mono font-medium text-accent hover:underline"
                  >
                    /{item.slug}
                  </a>
                </td>
                <td className="px-4 py-2.5">
                  <Destination item={item} />
                </td>
                <td className="px-4 py-2.5 tabular-nums">{formatCount(item.click_count)}</td>
                <td className="px-4 py-2.5 text-muted">{formatDate(item.created_at)}</td>
                <td className="px-4 py-2.5 text-muted">{formatRelative(item.expires_at)}</td>
                <td className="px-4 py-2.5">
                  <StatusBadge item={item} />
                </td>
                <td className="px-2 py-1.5">
                  <Actions
                    item={item}
                    onEdit={() => setEditing(item)}
                    onAnalytics={() => setAnalyzing(item)}
                    onQR={() => setQrItem(item)}
                    onDelete={() => setDeleting(item)}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>

      {/* Mobile cards */}
      <div className="space-y-3 md:hidden">
        {items.map((item) => (
          <Card key={item.id} className="p-4">
            <div className="flex items-start justify-between gap-2">
              <a
                href={item.short_url}
                target="_blank"
                rel="noreferrer"
                className="font-mono text-sm font-medium text-accent hover:underline"
              >
                /{item.slug}
              </a>
              <StatusBadge item={item} />
            </div>
            <p className="mt-1" title={item.original_url}>
              <Destination item={item} />
            </p>
            <div className="mt-2 flex items-center gap-3 text-[12px] text-muted">
              <span className="tabular-nums">{formatCount(item.click_count)} clicks</span>
              <span>{formatDate(item.created_at)}</span>
            </div>
            <div className="mt-3 border-t border-border pt-2">
              <Actions
                item={item}
                onEdit={() => setEditing(item)}
                onAnalytics={() => setAnalyzing(item)}
                onQR={() => setQrItem(item)}
                onDelete={() => setDeleting(item)}
              />
            </div>
          </Card>
        ))}
      </div>

      {editing && (
        <EditModal
          item={editing}
          onClose={() => setEditing(null)}
          onSaved={(updated) => {
            onChanged(updated);
          }}
        />
      )}
      {analyzing && (
        <AnalyticsModal urlId={analyzing.id} slug={analyzing.slug} onClose={() => setAnalyzing(null)} />
      )}
      {qrItem && <QRModal url={qrItem.short_url} onClose={() => setQrItem(null)} />}
      {deleting && (
        <ConfirmDelete
          item={deleting}
          onCancel={() => setDeleting(null)}
          onConfirm={confirmDelete}
        />
      )}
      {deleteLoading && null}
    </>
  );
}
