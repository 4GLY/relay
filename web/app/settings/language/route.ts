import { NextResponse } from "next/server";

import { RELAY_LOCALE_COOKIE, type Locale, isLocale } from "@/i18n/routing";

function parseLocale(value: FormDataEntryValue | string | null): Locale | null {
  if (typeof value !== "string") return null;
  return isLocale(value) ? value : null;
}

function parseRedirectTo(value: FormDataEntryValue | string | null | undefined): string {
  if (typeof value !== "string") return "/";
  return value.startsWith("/") && !value.startsWith("//") ? value : "/";
}

export async function POST(request: Request) {
  const contentType = request.headers.get("content-type") ?? "";
  let locale: Locale | null = null;
  let redirectTo = "/";

  if (contentType.includes("application/json")) {
    const body = (await request.json()) as { locale?: string; redirectTo?: string };
    locale = parseLocale(body.locale ?? null);
    redirectTo = parseRedirectTo(body.redirectTo);
  } else {
    const formData = await request.formData();
    locale = parseLocale(formData.get("locale"));
    redirectTo = parseRedirectTo(formData.get("redirectTo"));
  }

  if (!locale) {
    return NextResponse.json(
      {
        ok: false,
        error: {
          code: "INVALID_LOCALE",
          message: "Unsupported locale.",
        },
      },
      { status: 400 },
    );
  }

  const response = contentType.includes("application/json")
    ? NextResponse.json({ ok: true, locale })
    : NextResponse.redirect(new URL(redirectTo, request.url));

  response.cookies.set(RELAY_LOCALE_COOKIE, locale, {
    httpOnly: false,
    maxAge: 60 * 60 * 24 * 365,
    path: "/",
    sameSite: "lax",
  });

  return response;
}
