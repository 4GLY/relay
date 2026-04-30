import type { Metadata } from "next";
import { cookies, headers } from "next/headers";
import { Fraunces, Nunito, JetBrains_Mono } from "next/font/google";

import { resolveLocale } from "@/lib/i18n";

import "./globals.css";

const fraunces = Fraunces({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-fraunces",
  style: ["normal", "italic"],
  axes: ["opsz", "SOFT", "WONK"],
});

const nunito = Nunito({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-nunito",
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-jetbrains-mono",
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: "Relay",
  description: "A quiet engine that turns chaos into swans.",
  applicationName: "Relay",
  authors: [{ name: "4gly Labs" }],
  openGraph: {
    title: "Relay",
    description: "A quiet engine that turns chaos into swans.",
    type: "website",
    siteName: "Relay",
  },
  twitter: {
    card: "summary",
    title: "Relay",
    description: "A quiet engine that turns chaos into swans.",
  },
};

export default async function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const cookieStore = await cookies();
  const headerStore = await headers();
  const locale = resolveLocale({
    cookie: cookieStore.toString(),
    acceptLanguage: headerStore.get("accept-language") ?? undefined,
  });

  return (
    <html
      lang={locale}
      className={`${fraunces.variable} ${nunito.variable} ${jetbrainsMono.variable}`}
    >
      <body
        style={{
          background: "var(--canvas)",
          color: "var(--ink)",
          fontFamily: "var(--font-sans)",
        }}
      >
        {children}
      </body>
    </html>
  );
}
