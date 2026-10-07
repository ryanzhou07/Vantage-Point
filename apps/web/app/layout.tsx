import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Vantage-Point",
  description: "Personal finance and shared expense management",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
