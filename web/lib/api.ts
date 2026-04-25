/**
 * Tiny client for the Relay Go API. Reads `NEXT_PUBLIC_RELAY_API_URL`
 * (defaults to `http://localhost:8080`) and prefixes the given path.
 *
 * Concrete endpoints live in S6/S7/S8. This module only ships the base
 * fetcher and the response envelope shape from `internal/contracts/envelope.go`.
 */

export const RELAY_API_URL =
  process.env.NEXT_PUBLIC_RELAY_API_URL ?? "http://localhost:8080";

/**
 * Issue a `fetch` against the Relay Go API. Caller owns headers, body, and
 * error handling. Returns the raw `Response`.
 */
export function relayFetch(path: string, init?: RequestInit): Promise<Response> {
  const url = path.startsWith("http")
    ? path
    : `${RELAY_API_URL}${path.startsWith("/") ? path : `/${path}`}`;
  return fetch(url, init);
}

/** Mirrors `contracts.SuccessEnvelope` from `internal/contracts/envelope.go`. */
export type RelaySuccessEnvelope<T> = {
  ok: true;
  command: string;
  data: T;
  warnings: string[];
};

/** Mirrors `contracts.ErrorPayload`. */
export type RelayErrorPayload = {
  code: string;
  message: string;
  retryable: boolean;
  missing_fields?: string[];
};

/** Mirrors `contracts.ErrorEnvelope`. */
export type RelayErrorEnvelope = {
  ok: false;
  command: string;
  error: RelayErrorPayload;
};

/** Either branch of the Relay envelope contract. */
export type RelayEnvelope<T> = RelaySuccessEnvelope<T> | RelayErrorEnvelope;
