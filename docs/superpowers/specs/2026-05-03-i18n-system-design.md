# Relay i18n System Design

Date: 2026-05-03
Status: Draft for user review

## Goal

Provide first-class English and Korean localization across Relay's user-facing product surface:

- the full Next.js web UI
- the public packet snapshot HTML surface served by the Go API

The API and MCP contracts remain language-neutral. Stable error `code` values stay as the contract; UI surfaces translate those codes into user-facing copy.

## Decisions

- Supported locales are `en` and `ko`.
- URL paths stay unchanged. Relay continues to serve `/projects/{id}`, `/style-memory`, `/settings/*`, and `/p/{snapshot}` without locale prefixes.
- Locale resolution order is:
  1. `relay_locale` cookie
  2. `Accept-Language`
  3. `en`
- Users can explicitly change language from the web UI. The selected locale is stored in `relay_locale` for one year.
- User-generated content is not translated. Project names, packet bodies, decisions, notes, traces, and heuristic text render as stored.
- API/MCP response bodies do not become locale-aware.

## Architecture

Relay will adopt `next-intl` for the Next.js web UI while preserving the current URL model.

The web app will add:

- `web/i18n/request.ts` to resolve the request locale and load messages.
- `web/messages/en.json` and `web/messages/ko.json` for localized UI messages.
- a root `NextIntlClientProvider` so Client Components can use `useTranslations`.
- server-side translation access through `getTranslations` for Server Components.

The current `web/lib/i18n` boundary will be narrowed. It can keep shared locale constants, cookie naming, locale parsing, and test helpers, but screen copy should move into `next-intl` messages.

The Go public snapshot page cannot use `next-intl` directly. It will receive a small Go-side locale resolver with the same locale policy and a small message map for the snapshot chrome.

## Message Namespaces

Messages should be organized by product surface so each screen can be migrated and tested independently:

- `Common`
- `Errors`
- `Shell`
- `Root`
- `Onboarding`
- `Settings.ProviderCredentials`
- `Settings.ApiKeys`
- `ProjectExplorer`
- `StyleMemory`
- `Traces`
- `DecisionGraph`
- `PacketBuilder`
- `PublicSnapshot`

The existing TypeScript dictionary keys should be migrated into these namespaces rather than kept as a single large `Dictionary` object.

## Web UI Data Flow

For each web request:

1. The request locale is resolved from `relay_locale`, then `Accept-Language`, then `en`.
2. `next-intl` loads `messages/{locale}.json`.
3. The root layout wraps the app in `NextIntlClientProvider`.
4. Server Components read copy with `getTranslations`.
5. Client Components read copy with `useTranslations`.

Client components should stop receiving large `copy` objects over props as they are migrated. They should read the specific message namespace they own.

## Language Switch UX

Relay needs visible language switching paths, not only the existing POST route.

### Authenticated App

`RelayTopRail` should expose a compact language selector near the existing Settings/user controls. It should be available from the main authenticated product screens:

- Project Explorer
- Style Memory
- Judgment Traces
- Decision Graph
- Packet Builder
- Settings

The control should submit to `/settings/language` and redirect back to the current path.

### Unauthenticated And Public Surfaces

Home, onboarding sign-in, and public snapshot pages should also include a language switch. Public snapshot recipients must be able to change language without signing in.

The switch should:

- keep the current URL path unchanged
- store `relay_locale`
- work as a normal form submission without requiring client-side JavaScript
- include an accessible label and current-language state

## Public Snapshot Scope

The Go-rendered public snapshot surface should localize:

- `<html lang>`
- OG/Twitter title and description chrome
- snapshot page footer
- Relay call-to-action copy
- unknown/revoked snapshot fallback copy rendered by `/p/{snapshot}` responses

It should not translate:

- `RenderedBody`
- project names
- note, decision, trace, heuristic, or packet content

## Error Handling

API and MCP errors keep stable machine-readable codes. UI surfaces map known codes to localized messages.

Known examples:

- `UNAUTHENTICATED`
- `INVALID_INPUT`
- `API_KEY_NOT_FOUND_BY_ID`

Unknown errors should fall back to `Common.unknownError` or a screen-specific fallback. Raw English server messages should not be the normal user-facing path.

## Migration Plan

1. Install and wire `next-intl`.
2. Add request config, message files, and root provider.
3. Migrate current `en.ts` and `ko.ts` dictionaries into `messages/en.json` and `messages/ko.json`.
4. Convert already-localized Home, Onboarding, Provider Settings, and API Key Settings to `next-intl`.
5. Convert remaining hardcoded UI surfaces:
   - `RelayTopRail` and app shell
   - Project Explorer
   - Style Memory
   - Judgment Traces
   - Decision Graph
   - Packet Builder
6. Add language switch UI to authenticated, unauthenticated, and public surfaces.
7. Add Go-side locale resolver and snapshot message map.
8. Add tests and message completeness checks.

## Testing

Required coverage:

- Locale resolution prefers `relay_locale` over `Accept-Language`.
- Locale resolution falls back from unsupported values to `en`.
- `/settings/language` stores valid locales, rejects invalid locales, and safely redirects only to local paths.
- English and Korean message files have identical key shapes.
- Main web pages render successfully in both `en` and `ko`.
- Client interactions that emit user-facing copy use localized messages.
- Public snapshot HTML renders Korean chrome for Korean locale requests.
- Public snapshot user content remains unchanged.

## Non-Goals

- No locale prefixes in URLs for this phase.
- No database-backed user language preference for this phase.
- No machine translation of user-generated content.
- No locale-aware API/MCP response contract.
- No SEO `hreflang` expansion until Relay has a public SEO surface that needs it.

## Risks

`next-intl` without locale-prefixed routing depends on request-time locale resolution and cookies. Any page or layout that reads locale from cookies or headers must opt out of static rendering through the repo's existing dynamic rendering pattern.

Large copy migration can accidentally translate code-like labels or user-generated content. The migration should keep domain identifiers and stored content outside message files.
