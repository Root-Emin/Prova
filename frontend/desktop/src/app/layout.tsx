import type { Metadata } from "next";
import localFont from "next/font/local";
import "./globals.css";

// Fonts are bundled; no CDN is reachable under file://.
const inter = localFont({
  variable: "--font-inter",
  src: [
    { path: "../fonts/inter-latin.woff2", style: "normal" },
    { path: "../fonts/inter-latin-ext.woff2", style: "normal" },
  ],
  display: "block",
});

const sourceSerif = localFont({
  variable: "--font-serif",
  src: [
    { path: "../fonts/source-serif-4-latin.woff2", style: "normal" },
    { path: "../fonts/source-serif-4-latin-ext.woff2", style: "normal" },
  ],
  display: "block",
});

export const metadata: Metadata = {
  title: "Prova",
  description: "Kurumsal sesli rol-oyun eğitim ve yetkinlik sertifikasyonu",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="tr"
      className={`${inter.variable} ${sourceSerif.variable} h-full antialiased`}
    >
      <body className="h-full">{children}</body>
    </html>
  );
}
