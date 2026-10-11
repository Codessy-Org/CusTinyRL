"use client";

import { FormEvent, useEffect, useState } from "react";
import { api, ApiError } from "../lib/api";
import { isValidHttpUrl } from "../lib/format";
import type { URLItem } from "../lib/types";
import { useToast } from "./Toast";
import { Button, Card, Input, Label, Spinner, Switch } from "./ui";

export function EditModal({
  item,
  onClose,
  onSaved,
}: {
  item: URLItem;
  onClose: () => void;
  onSaved: (updated: URLItem) => void;
}) {
  const [url, setUrl] = useState(item.original_url);
  const [active, setActive] = useState(item.is_active);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { push } = useToast();

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (!isValidHttpUrl(url.trim())) {
      setError("Enter a valid http(s) URL.");
      return;
    }
    setLoading(true);
    try {
      const updated = await api<URLItem>(`/api/v1/urls/${item.id}`, {
        method: "PATCH",
        body: { original_url: url.trim(), is_active: active },
      });
      onSaved(updated);
      push("Link updated", "success");
      onClose();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to update link.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-overlay p-4"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={`Edit ${item.slug}`}
    >
      <Card className="animate-in w-full max-w-md p-5" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-[15px] font-semibold">Edit link</h2>
            <p className="font-mono text-[12px] text-muted">/{item.slug}</p>
          </div>
          <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close editor">
            ✕
          </Button>
        </div>

        <form onSubmit={submit} className="space-y-4">
          <div>
            <Label htmlFor="edit-url">Destination URL</Label>
            <Input
              id="edit-url"
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              required
            />
          </div>
          <div className="flex items-center justify-between rounded-md border border-border px-3 py-2">
            <div>
              <p className="text-[13px] font-medium">Active</p>
              <p className="text-[12px] text-muted">Disabled links return 410 Gone.</p>
            </div>
            <Switch checked={active} onChange={setActive} label="Toggle active state" />
          </div>
          {error && (
            <p role="alert" className="text-[13px] text-danger">
              {error}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" disabled={loading}>
              {loading && <Spinner className="h-3.5 w-3.5" />}
              Save
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}
