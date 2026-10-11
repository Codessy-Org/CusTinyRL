import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CreateForm } from "../components/CreateForm";
import { ToastProvider } from "../components/Toast";
import type { URLItem } from "../lib/types";

function renderForm(onCreated = vi.fn()) {
  render(
    <ToastProvider>
      <CreateForm onCreated={onCreated} />
    </ToastProvider>,
  );
  return { onCreated };
}

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("CreateForm validation", () => {
  it("rejects an empty destination URL", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderForm();

    await user.click(screen.getByRole("button", { name: /shorten/i }));

    expect(await screen.findByText(/paste a url to shorten/i)).toBeInTheDocument();
    expect(onCreated).not.toHaveBeenCalled();
  });

  it("rejects non-http(s) URLs", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText(/destination url/i), "javascript:alert(1)");
    await user.click(screen.getByRole("button", { name: /shorten/i }));

    expect(await screen.findByText(/valid http\(s\) url/i)).toBeInTheDocument();
  });

  it("does not call the API when client-side validation fails", async () => {
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("button", { name: /shorten/i }));

    expect(await screen.findByText(/paste a url/i)).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });
});

describe("CreateForm slug modes", () => {
  it("shows custom slug input only in custom mode", async () => {
    const user = userEvent.setup();
    renderForm();

    expect(screen.queryByLabelText("Custom slug")).not.toBeInTheDocument();
    expect(screen.getByLabelText(/prefix for generated slug/i)).toBeInTheDocument();

    await user.click(screen.getByRole("switch", { name: /toggle custom slug/i }));

    expect(await screen.findByLabelText("Custom slug")).toBeInTheDocument();
    expect(screen.queryByLabelText(/prefix for generated slug/i)).not.toBeInTheDocument();
  });

  it("rejects invalid custom slugs", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("switch", { name: /toggle custom slug/i }));
    await user.type(screen.getByLabelText(/destination url/i), "https://example.com");
    await user.type(screen.getByLabelText("Custom slug"), "INVALID SLUG");
    await user.click(screen.getByRole("button", { name: /shorten/i }));

    expect(
      await screen.findByText(/lowercase letters, numbers, hyphens/i),
    ).toBeInTheDocument();
  });
});

describe("CreateForm submission", () => {
  it("submits automatic slug mode and shows the created URL", async () => {
    const created: URLItem = {
      id: 1,
      slug: "aBc123xYz9",
      short_url: "http://localhost:8080/aBc123xYz9",
      original_url: "https://example.com",
      title: null,
      is_active: true,
      is_expired: false,
      click_count: 0,
      expires_at: null,
      created_at: new Date().toISOString(),
    };
    const fetchSpy = vi.fn().mockResolvedValue(jsonResponse(201, created));
    vi.stubGlobal("fetch", fetchSpy);

    const user = userEvent.setup();
    const { onCreated } = renderForm();

    await user.type(screen.getByLabelText(/destination url/i), "https://example.com");
    await user.type(screen.getByLabelText(/prefix for generated slug/i), "resume");
    await user.click(screen.getByRole("button", { name: /shorten/i }));

    expect(await screen.findByText(/localhost:8080\/aBc123xYz9/)).toBeInTheDocument();
    expect(onCreated).toHaveBeenCalledWith(created);
    const body = JSON.parse((fetchSpy.mock.calls[0][1] as { body: string }).body);
    expect(body).toEqual({ original_url: "https://example.com", prefix: "resume" });
    expect(body.custom_slug).toBeUndefined();
  });

  it("surfaces slug conflicts with an actionable message and keeps input", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        jsonResponse(409, {
          error: { code: "slug_taken", message: "that slug is already taken" },
        }),
      ),
    );

    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("switch", { name: /toggle custom slug/i }));
    await user.type(screen.getByLabelText(/destination url/i), "https://example.com");
    await user.type(screen.getByLabelText("Custom slug"), "taken-slug");
    await user.click(screen.getByRole("button", { name: /shorten/i }));

    expect(
      await screen.findByText(/slug "taken-slug" is already taken/i),
    ).toBeInTheDocument();
    // User input must be preserved, never silently replaced.
    expect(screen.getByLabelText("Custom slug")).toHaveValue("taken-slug");
  });

  it("copies the short URL to the clipboard", async () => {
    const created: URLItem = {
      id: 2,
      slug: "copied",
      short_url: "http://localhost:8080/copied",
      original_url: "https://example.com",
      title: null,
      is_active: true,
      is_expired: false,
      click_count: 0,
      expires_at: null,
      created_at: new Date().toISOString(),
    };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(201, created)));

    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText(/destination url/i), "https://example.com");
    await user.click(screen.getByRole("button", { name: /shorten/i }));

    const copyBtn = await screen.findByRole("button", { name: /copy/i });
    await user.click(copyBtn);

    await waitFor(async () => {
      expect(await navigator.clipboard.readText()).toBe("http://localhost:8080/copied");
    });
  });

  it("shows a network error when the API is unreachable", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("Failed to fetch")),
    );
    const user = userEvent.setup();
    renderForm();

    await user.type(screen.getByLabelText(/destination url/i), "https://example.com");
    await user.click(screen.getByRole("button", { name: /shorten/i }));

    expect(await screen.findByText(/cannot reach the api/i)).toBeInTheDocument();
  });
});
