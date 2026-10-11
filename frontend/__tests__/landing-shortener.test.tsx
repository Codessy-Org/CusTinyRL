import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/link", () => ({
  default: ({ href, children, ...props }: React.PropsWithChildren<{ href: string }>) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

import { ApiError, api } from "../lib/api";
import { LandingShortener } from "../components/LandingShortener";

vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return { ...actual, api: vi.fn() };
});

const apiMock = vi.mocked(api);

describe("LandingShortener", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders anonymous create form without a custom slug field", () => {
    render(<LandingShortener />);
    expect(screen.getByLabelText(/paste a long url/i)).toBeInTheDocument();
    // Custom slugs are sign-in only — anonymous users get automatic slugs.
    expect(screen.queryByLabelText(/custom slug/i)).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /shorten it — no login needed/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/works anonymously/i)).toBeInTheDocument();
    expect(screen.getByText(/for custom slugs, analytics/i)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute(
      "href",
      "/dashboard",
    );
  });

  it("validates the URL before calling the API", async () => {
    render(<LandingShortener />);
    fireEvent.click(
      screen.getByRole("button", { name: /shorten it/i }),
    );
    expect(
      await screen.findByText(/paste a url to shorten/i),
    ).toBeInTheDocument();
    expect(apiMock).not.toHaveBeenCalled();
  });

  it("creates a link anonymously and shows the short URL", async () => {
    apiMock.mockResolvedValue({
      id: 1,
      slug: "google-pm-referral",
      short_url: "http://localhost:8090/google-pm-referral",
      original_url: "https://example.com/job",
      title: "Example Job Page",
      favicon_url: "https://example.com/favicon.ico",
      is_active: true,
      is_expired: false,
      click_count: 0,
      expires_at: null,
      created_at: "2026-10-11T00:00:00Z",
    });

    render(<LandingShortener />);
    fireEvent.change(screen.getByLabelText(/paste a long url/i), {
      target: { value: "https://example.com/job" },
    });
    fireEvent.click(screen.getByRole("button", { name: /shorten it/i }));

    await waitFor(() =>
      expect(screen.getByText(/localhost:8090\/google-pm-referral/)).toBeInTheDocument(),
    );
    expect(apiMock).toHaveBeenCalledWith("/api/v1/urls", {
      method: "POST",
      body: {
        original_url: "https://example.com/job",
      },
    });
    expect(screen.getByText(/no account required/i)).toBeInTheDocument();
    expect(screen.getByText(/want clicks, referrers/i)).toBeInTheDocument();
  });

  it("explains the anonymous rate limit on 429", async () => {
    apiMock.mockRejectedValue(
      new ApiError(429, "rate_limited", "too many requests"),
    );

    render(<LandingShortener />);
    fireEvent.change(screen.getByLabelText(/paste a long url/i), {
      target: { value: "https://example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /shorten it/i }));

    expect(
      await screen.findByText(/5\/min per IP/i),
    ).toBeInTheDocument();
  });
});
