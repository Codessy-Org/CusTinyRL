import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/link", () => ({
  default: ({ href, children, ...props }: React.PropsWithChildren<{ href: string }>) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

import { ApiKeys } from "../components/ApiKeys";
import { api } from "../lib/api";

vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return { ...actual, api: vi.fn() };
});

const apiMock = vi.mocked(api);

describe("ApiKeys", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("lists existing keys with prefix and scopes", async () => {
    apiMock.mockResolvedValueOnce({
      data: [
        {
          id: 3,
          name: "ci-pipeline",
          key_prefix: "ck_abc12345",
          scopes: ["create", "read"],
          created_at: "2026-10-01T00:00:00Z",
          last_used_at: null,
          revoked_at: null,
        },
      ],
    });

    render(<ApiKeys />);
    await waitFor(() => expect(screen.getByText("ci-pipeline")).toBeInTheDocument());
    expect(screen.getByText(/ck_abc12345/)).toBeInTheDocument();
    expect(screen.getByText("create, read")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /revoke/i })).toBeInTheDocument();
  });

  it("creates a key and shows the raw secret exactly once", async () => {
    apiMock
      .mockResolvedValueOnce({ data: [] }) // initial list
      .mockResolvedValueOnce({
        key: "ck_SUPERSECRET",
        data: {
          id: 9,
          name: "new-key",
          key_prefix: "ck_SUPERSE",
          scopes: ["create"],
          created_at: "2026-10-11T00:00:00Z",
        },
      })
      .mockResolvedValueOnce({ data: [] }); // reload

    render(<ApiKeys />);
    await waitFor(() => expect(apiMock).toHaveBeenCalledTimes(1));

    const input = screen.getByLabelText("Name");
    // jsdom does not wire fireEvent change for controlled inputs automatically
    input.setAttribute("value", "new-key");
    fireEvent.change(input, { target: { value: "new-key" } });
    fireEvent.click(screen.getByRole("button", { name: /create key/i }));

    await waitFor(() =>
      expect(screen.getByText(/copy this key now/i)).toBeInTheDocument(),
    );
    expect(screen.getByText("ck_SUPERSECRET")).toBeInTheDocument();
  });

  it("shows an error when listing fails", async () => {
    apiMock.mockRejectedValue(new Error("boom"));
    render(<ApiKeys />);
    expect(await screen.findByText(/failed to load api keys/i)).toBeInTheDocument();
  });
});
