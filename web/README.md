# Relay web

The Relay V2 end-user web surface. Public landing, the authenticated **Style Memory** screen,
read-only **Sharable Packet Snapshot** pages, and the **1-click Onboarding** flow live here.

This package only ships the scaffold (S5). The real screens land in:

- **S6** — Style Memory authenticated UI with the 900 ms duckling→swan signature transform.
- **S7** — Sharable Packet Snapshot URL (`/p/{snapshotId}`); the Go backend serves `/p/{token}` from S3.
- **S8** — 1-click Onboarding (Anthropic key → Relay URL → connect, 60 seconds to first packet).

## Stack

- Next.js 14 (App Router) · TypeScript strict
- Tailwind CSS v4 (CSS-first via `@import "tailwindcss"` + `@theme`)
- shadcn/ui (New York preset)
- Framer Motion
- `next/font` for Fraunces, Nunito, JetBrains Mono (stand-in for Commit Mono until licensed)

All visual tokens come from `DESIGN.md §6`. Do not introduce additional fonts or colors.

## Run

```bash
cd web
cp .env.example .env.local   # adjust if your Go API runs on a different port
npm install
npm run dev
```

Then open <http://localhost:3000>.

## Talking to the Go API

`web/lib/api.ts` exports `relayFetch(path, init?)` which prefixes `NEXT_PUBLIC_RELAY_API_URL`
(defaults to `http://localhost:8080`). Endpoint contracts come from `docs/openapi.yaml` and
`internal/contracts/envelope.go`. S6/S7/S8 add the typed wrappers.

## Scripts

- `npm run dev` — local Next dev server
- `npm run build` — production build
- `npm run start` — serve the production build
- `npm run lint` — `next lint` with `next/core-web-vitals` + `next/typescript`
- `npm run typecheck` — `tsc --noEmit`

## Conventions

- Use only DESIGN.md §6 tokens (`--surface`, `--ink`, `--magic-primary`, …).
- Use only the declared typefaces: Fraunces, Nunito, JetBrains Mono.
- `--magic-primary` and `--magic-accent` only appear at transformation moments — never as
  ambient background.
- Elevation is pastel halo, not drop-shadow.
