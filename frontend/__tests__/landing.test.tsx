import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const push = vi.fn();
vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: React.PropsWithChildren<{ href: string }>) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));

import { LandingPage } from "../components/LandingPage";

describe("LandingPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders the hero headline and CTAs", () => {
    render(<LandingPage />);
    expect(screen.getByText(/short links/i)).toBeInTheDocument();
    // Anonymous shortener is embedded right in the hero.
    expect(screen.getByLabelText(/paste a long url/i)).toBeInTheDocument();
    // Custom slugs are sign-in only — no slug field for anonymous visitors.
    expect(
      screen.queryByRole("textbox", { name: /custom slug/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /shorten it — no login needed/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /open dashboard/i }),
    ).toHaveAttribute("href", "/dashboard");
    expect(
      screen.getByRole("link", { name: /self-host in one command/i }),
    ).toHaveAttribute("href", "#self-host");
  });

  it("explains that login is only needed for analytics", () => {
    render(<LandingPage />);
    expect(screen.getByText(/no account needed/i)).toBeInTheDocument();
    expect(screen.getAllByText(/sign in/i).length).toBeGreaterThan(0);
  });

  it("links the nav to dashboard and section anchors", () => {
    render(<LandingPage />);
    const nav = screen.getByRole("navigation");
    expect(within(nav).getByRole("link", { name: "Dashboard" })).toHaveAttribute(
      "href",
      "/dashboard",
    );
    expect(within(nav).getByRole("link", { name: "Features" })).toHaveAttribute(
      "href",
      "#features",
    );
    expect(within(nav).getByRole("link", { name: "Compare" })).toHaveAttribute(
      "href",
      "#compare",
    );
  });

  it("explains the comparison against hosted shorteners", () => {
    render(<LandingPage />);
    const table = screen.getByRole("table");
    expect(within(table).getByText("Exact custom slugs")).toBeInTheDocument();
    expect(within(table).getByText("Free, verbatim")).toBeInTheDocument();
    expect(within(table).getByText("Hosted shorteners")).toBeInTheDocument();
    expect(within(table).getByText("Your Postgres")).toBeInTheDocument();
  });

  it("renders feature cards", () => {
    render(<LandingPage />);
    expect(screen.getByText("µs-scale redirects")).toBeInTheDocument();
    expect(
      screen.getByText("Analytics without surveillance"),
    ).toBeInTheDocument();
    expect(screen.getByText("IP addresses stored")).toBeInTheDocument();
  });
});
