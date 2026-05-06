# Relay web

The Relay V2 end-user web surface. Public landing, the authenticated **Style Memory** screen,
read-only **Sharable Packet Snapshot** pages, and the **1-click Onboarding** flow live here.

This package contains the initial scaffold plus early authenticated, onboarding,
and shareable packet routes. The next hardening milestones are:

- **S6** — Style Memory authenticated UI with the 900 ms review-to-approved transition.
- **S7** — Sharable Packet Snapshot public URL (`/p/{token}`) is owned by the Go backend; the Next `/p/{snapshotId}` route is a migration placeholder, not the canonical renderer.
- **S8** — 1-click Onboarding (create workspace first; provider keys move to a later settings flow).

## Stack

- Next.js 14 (App Router) · TypeScript strict
- Tailwind CSS v4 (CSS-first via `@import "tailwindcss"` + `@theme`)
- shadcn/ui (New York preset)
- Framer Motion
- `next/font/local` for self-hosted Fraunces, Nunito, JetBrains Mono, and LXGW WenKai KR for Korean UI

All visual tokens come from `DESIGN.md` `Colors` and the token front matter. Do not introduce additional fonts or colors.

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

- Use `DESIGN.md` `Colors` and token front matter names as the canonical vocabulary for new design docs and contracts.
- When editing CSS before the runtime-token migration, verify the concrete variable names in `web/app/globals.css`; some compatibility aliases still exist.
- Use only the declared typefaces: Fraunces, Nunito, JetBrains Mono, and LXGW WenKai KR for Korean UI.
- Accent tokens only appear for focus, review, selected paths, and completion emphasis — never as ambient background.
- Canonical elevation is soft focus-ring emphasis. Some current shared primitives still carry legacy shadows; check `web/app/globals.css` before changing runtime elevation.
- Existing legacy runtime CSS variables are compatibility aliases until the web CSS migration removes them.
