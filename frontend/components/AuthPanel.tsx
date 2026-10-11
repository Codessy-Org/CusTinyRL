"use client";

import { FormEvent, useState } from "react";
import { api, ApiError, setToken } from "../lib/api";
import type { AuthResponse } from "../lib/types";
import { Button, Card, Input, Label, Spinner } from "./ui";

export function AuthPanel({ onAuthed }: { onAuthed: () => void }) {
  const [mode, setMode] = useState<"login" | "register">("register");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (!email.includes("@")) {
      setError("Enter a valid email address.");
      return;
    }
    if (password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    setLoading(true);
    try {
      const res = await api<AuthResponse>(`/api/v1/auth/${mode}`, {
        method: "POST",
        body: { email, password },
      });
      setToken(res.token);
      onAuthed();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Something went wrong. Try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-[70vh] items-center justify-center px-4">
      <Card className="w-full max-w-sm p-6 sm:p-8">
        <div className="mb-6">
          <h1 className="text-lg font-semibold tracking-tight">
            {mode === "register" ? "Create your account" : "Welcome back"}
          </h1>
          <p className="mt-1 text-[13px] text-muted">
            {mode === "register"
              ? "Sign up to create and manage short links."
              : "Sign in to manage your short links."}
          </p>
        </div>

        <div className="mb-5 grid grid-cols-2 rounded-md border border-border p-0.5">
          {(["register", "login"] as const).map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => {
                setMode(m);
                setError(null);
              }}
              className={
                "rounded px-3 py-1.5 text-[13px] font-medium transition-colors " +
                (mode === m ? "bg-foreground text-background" : "text-muted hover:text-foreground")
              }
            >
              {m === "register" ? "Sign up" : "Sign in"}
            </button>
          ))}
        </div>

        <form onSubmit={submit} className="space-y-4">
          <div>
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              autoComplete={mode === "register" ? "new-password" : "current-password"}
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>
          {error && (
            <p role="alert" className="text-[13px] text-danger">
              {error}
            </p>
          )}
          <Button type="submit" variant="primary" className="w-full" disabled={loading}>
            {loading && <Spinner className="h-3.5 w-3.5" />}
            {mode === "register" ? "Create account" : "Sign in"}
          </Button>
        </form>
      </Card>
    </div>
  );
}
