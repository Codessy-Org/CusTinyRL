import { describe, expect, it } from "vitest";
import { isValidHttpUrl, PREFIX_PATTERN, SLUG_PATTERN } from "../lib/format";

describe("isValidHttpUrl", () => {
  it("accepts http and https URLs", () => {
    expect(isValidHttpUrl("https://example.com")).toBe(true);
    expect(isValidHttpUrl("http://example.com/path?q=1")).toBe(true);
  });

  it("rejects other schemes and malformed values", () => {
    expect(isValidHttpUrl("javascript:alert(1)")).toBe(false);
    expect(isValidHttpUrl("data:text/html,x")).toBe(false);
    expect(isValidHttpUrl("file:///etc/passwd")).toBe(false);
    expect(isValidHttpUrl("not a url")).toBe(false);
    expect(isValidHttpUrl("")).toBe(false);
  });
});

describe("slug patterns", () => {
  it("accepts valid custom slugs", () => {
    for (const s of ["gauravresumev1", "a", "my-link_2", "0"]) {
      expect(SLUG_PATTERN.test(s)).toBe(true);
    }
  });

  it("rejects invalid custom slugs", () => {
    for (const s of ["", "UPPER", "has space", "-lead", "_lead", "a".repeat(65)]) {
      expect(SLUG_PATTERN.test(s)).toBe(false);
    }
  });

  it("validates prefixes", () => {
    expect(PREFIX_PATTERN.test("resume")).toBe(true);
    expect(PREFIX_PATTERN.test("UPPER")).toBe(false);
    expect(PREFIX_PATTERN.test("a".repeat(33))).toBe(false);
  });
});
