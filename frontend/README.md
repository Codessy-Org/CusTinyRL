# cut.sh frontend

Next.js 16 + TypeScript + Tailwind CSS 4 dashboard for the URL shortener.
Vercel-inspired UI: dark/light themes, quick-create form, compact link table,
and usage/analytics views.

See the [repository README](../README.md) for full setup instructions.

```bash
cp .env.example .env.local     # NEXT_PUBLIC_API_URL
npm ci
npm run dev                    # http://localhost:3000
npm test                       # Vitest + Testing Library
npm run lint && npm run build
```
