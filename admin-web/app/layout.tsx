import type { Metadata } from "next";
import "./globals.css";
export const metadata: Metadata = {
  title: "Doodhee Admin",
  description: "Local milk, clearly managed.",
};
export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
