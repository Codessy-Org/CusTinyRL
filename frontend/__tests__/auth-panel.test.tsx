import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AuthPanel } from "../components/AuthPanel";
import type { AuthResponse } from "../lib/types";
import { getToken } from "../lib/api";

function jsonResponse(status: number, body: unknown): Response {
  return { ok: status >= 200 && status < 300, status, json: async () => body } as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
  window.localStorage.removeItem("cutsh_token");
});

describe("AuthPanel", () => {
  it("validates password length before calling the API", async () => {
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
    const user = userEvent.setup();
    render(<AuthPanel onAuthed={() => {}} />);

    await user.type(screen.getByLabelText("Email"), "you@example.com");
    await user.type(screen.getByLabelText("Password"), "short");
    await user.click(screen.getByRole("button", { name: /create account/i }));

    expect(await screen.findByText(/at least 8 characters/i)).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("stores the token on successful registration", async () => {
    const res: AuthResponse = {
      token: "test-token-123",
      expires_at: new Date(Date.now() + 86_400_000).toISOString(),
      user: { id: 1, email: "you@example.com", created_at: new Date().toISOString() },
    };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(201, res)));
    const onAuthed = vi.fn();
    const user = userEvent.setup();
    render(<AuthPanel onAuthed={onAuthed} />);

    await user.type(screen.getByLabelText("Email"), "you@example.com");
    await user.type(screen.getByLabelText("Password"), "password123");
    await user.click(screen.getByRole("button", { name: /create account/i }));

    await vi.waitFor(() => expect(onAuthed).toHaveBeenCalled());
    expect(getToken()).toBe("test-token-123");
  });

  it("shows server-side error messages", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        jsonResponse(401, {
          error: { code: "invalid_credentials", message: "invalid email or password" },
        }),
      ),
    );
    const user = userEvent.setup();
    render(<AuthPanel onAuthed={() => {}} />);

    await user.click(screen.getByRole("button", { name: /sign in/i }));
    await user.type(screen.getByLabelText("Email"), "you@example.com");
    await user.type(screen.getByLabelText("Password"), "password123");
    const submit = screen
      .getAllByRole("button", { name: /sign in/i })
      .find((b) => b.getAttribute("type") === "submit");
    expect(submit).toBeTruthy();
    await user.click(submit!);

    expect(await screen.findByText(/invalid email or password/i)).toBeInTheDocument();
  });
});
