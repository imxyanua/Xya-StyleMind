import type { Metadata } from "next";
import Link from "next/link";

import "./globals.css";

export const metadata: Metadata = {
  title: "Xya-StyleMind",
  description: "AI-powered fashion ecommerce platform",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>
        <header className="border-b">
          <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4">
            <Link href="/" className="text-lg font-semibold">
              Xya-StyleMind
            </Link>
            <nav className="flex items-center gap-4 text-sm">
              <Link href="/products" className="hover:underline">
                Products
              </Link>
              <Link href="/cart" className="hover:underline">
                Cart
              </Link>
              <Link href="/orders" className="hover:underline">
                Orders
              </Link>
              <Link href="/admin" className="hover:underline">
                Admin
              </Link>
              <Link href="/login" className="hover:underline">
                Login
              </Link>
              <Link href="/register" className="hover:underline">
                Register
              </Link>
            </nav>
          </div>
        </header>
        <main className="mx-auto max-w-6xl px-4 py-8">{children}</main>
      </body>
    </html>
  );
}
