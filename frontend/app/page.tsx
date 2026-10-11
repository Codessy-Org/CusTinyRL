import type { Metadata } from "next";
import { LandingPage } from "../components/LandingPage";

export const metadata: Metadata = {
  title: "cut.sh — short links, full control",
  description:
    "A production-grade, self-hostable URL shortener: exact custom slugs, Redis-cached redirects, distributed rate limiting, and privacy-conscious analytics.",
};

export default function HomePage() {
  return <LandingPage />;
}
