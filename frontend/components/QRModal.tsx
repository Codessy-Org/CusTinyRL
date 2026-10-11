"use client";

import QRCode from "qrcode";
import { useEffect, useState } from "react";
import { Button, Spinner } from "./ui";

/**
 * Modal showing a QR code for a short URL, with SVG and PNG downloads.
 * Generated entirely client-side — the URL never leaves the browser.
 */
export function QRModal({ url, onClose }: { url: string; onClose: () => void }) {
  const [svg, setSvg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    QRCode.toString(url, { type: "svg", margin: 1, width: 280, errorCorrectionLevel: "M" })
      .then((s) => {
        if (!cancelled) setSvg(s);
      })
      .catch(() => {
        if (!cancelled) setError("Could not generate QR code.");
      });
    return () => {
      cancelled = true;
    };
  }, [url]);

  async function downloadPNG() {
    const canvas = await QRCode.toCanvas(url, { width: 1024, margin: 2 });
    const a = document.createElement("a");
    a.href = canvas.toDataURL("image/png");
    a.download = `qr-${url.split("/").pop() || "link"}.png`;
    a.click();
  }

  async function downloadSVG() {
    if (!svg) return;
    const blob = new Blob([svg], { type: "image/svg+xml" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `qr-${url.split("/").pop() || "link"}.svg`;
    a.click();
    URL.revokeObjectURL(a.href);
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label="QR code"
      onClick={onClose}
    >
      <div
        className="glass animate-in w-full max-w-sm rounded-lg border border-border bg-card p-5 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <h2 className="text-[14px] font-semibold">QR code</h2>
          <Button variant="ghost" size="sm" onClick={onClose} aria-label="Close">
            ✕
          </Button>
        </div>
        <p className="mt-0.5 truncate font-mono text-[12px] text-muted">
          {url.replace(/^https?:\/\//, "")}
        </p>
        <div className="mt-4 flex justify-center">
          {error && <p className="text-[13px] text-danger">{error}</p>}
          {!error && !svg && <Spinner className="h-6 w-6 text-muted" />}
          {svg && (
            <div
              className="rounded-md border border-border bg-white p-3 [&>svg]:h-56 [&>svg]:w-56"
              // SVG is generated locally by qrcode from the URL string.
              dangerouslySetInnerHTML={{ __html: svg }}
            />
          )}
        </div>
        <div className="mt-4 flex justify-end gap-2">
          <Button variant="secondary" size="sm" onClick={downloadSVG} disabled={!svg}>
            Download SVG
          </Button>
          <Button variant="primary" size="sm" onClick={downloadPNG} disabled={Boolean(error)}>
            Download PNG
          </Button>
        </div>
      </div>
    </div>
  );
}
