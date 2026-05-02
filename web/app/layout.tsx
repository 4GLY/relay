import type { Metadata } from "next";
import localFont from "next/font/local";
import { NextIntlClientProvider } from "next-intl";

import { resolveRequestLocale } from "@/i18n/request";

import "./globals.css";

const fraunces = localFont({
  src: [
    {
      path: "./fonts/Fraunces-VariableFont_SOFT_WONK_opsz_wght.ttf",
      style: "normal",
      weight: "100 900",
    },
    {
      path: "./fonts/Fraunces-Italic-VariableFont_SOFT_WONK_opsz_wght.ttf",
      style: "italic",
      weight: "100 900",
    },
  ],
  display: "swap",
  variable: "--font-fraunces",
});

const nunito = localFont({
  src: [
    {
      path: "./fonts/Nunito-VariableFont_wght.ttf",
      style: "normal",
      weight: "200 1000",
    },
    {
      path: "./fonts/Nunito-Italic-VariableFont_wght.ttf",
      style: "italic",
      weight: "200 1000",
    },
  ],
  display: "swap",
  variable: "--font-nunito",
});

const jetbrainsMono = localFont({
  src: [
    {
      path: "./fonts/JetBrainsMono-VariableFont_wght.ttf",
      style: "normal",
      weight: "100 800",
    },
    {
      path: "./fonts/JetBrainsMono-Italic-VariableFont_wght.ttf",
      style: "italic",
      weight: "100 800",
    },
  ],
  display: "swap",
  variable: "--font-jetbrains-mono",
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
  const locale = await resolveRequestLocale();

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
        <NextIntlClientProvider>{children}</NextIntlClientProvider>
      </body>
    </html>
  );
}
