import type { Metadata } from "next";
import { Fraunces, Nunito, JetBrains_Mono } from "next/font/google";
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

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      lang="en"
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
