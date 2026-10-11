export function formatDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: d.getFullYear() === new Date().getFullYear() ? undefined : "numeric",
  });
}

export function formatRelative(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const diff = d.getTime() - Date.now();
  const abs = Math.abs(diff);
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const minutes = Math.round(abs / 60_000);
  if (minutes < 60) return rtf.format(Math.sign(diff) * minutes, "minute");
  const hours = Math.round(abs / 3_600_000);
  if (hours < 24) return rtf.format(Math.sign(diff) * hours, "hour");
  const days = Math.round(abs / 86_400_000);
  return rtf.format(Math.sign(diff) * days, "day");
}

export function formatCount(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1).replace(/\.0$/, "") + "M";
  if (n >= 1_000) return (n / 1_000).toFixed(1).replace(/\.0$/, "") + "k";
  return String(n);
}

export function shortDisplayUrl(baseUrl: string, slug: string): string {
  try {
    const u = new URL(baseUrl);
    return u.host + "/" + slug;
  } catch {
    return baseUrl + "/" + slug;
  }
}

export function isValidHttpUrl(value: string): boolean {
  try {
    const u = new URL(value);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

export const SLUG_PATTERN = /^[a-z0-9][a-z0-9_-]{0,63}$/;
export const PREFIX_PATTERN = /^[a-z0-9][a-z0-9_-]{0,31}$/;
