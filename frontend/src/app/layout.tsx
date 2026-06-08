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
    <html lang="en" data-scroll-behavior="smooth">
      <body>
        <header className="sticky top-0 z-40 border-b border-border/70 bg-background/82 backdrop-blur-xl">
          <div className="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-3 sm:py-4 lg:flex-row lg:items-center lg:justify-between">
            <Link href="/" className="group flex min-w-0 items-center gap-3">
              <span className="grid size-10 shrink-0 place-items-center rounded-2xl bg-primary text-sm font-semibold text-primary-foreground shadow-soft">
                XS
              </span>
              <span className="min-w-0 leading-tight">
                <span className="block font-heading text-2xl font-semibold">Xya-StyleMind</span>
                <span className="text-xs uppercase tracking-[0.22em] text-muted-foreground">
                  AI fashion commerce
                </span>
              </span>
            </Link>
            <nav className="-mx-1 flex gap-2 overflow-x-auto px-1 pb-1 text-sm sm:flex-wrap sm:overflow-visible sm:pb-0">
              <Link href="/products" className="rounded-full px-3 py-1.5 hover:bg-muted">
                Products
              </Link>
              <Link href="/cart" className="rounded-full px-3 py-1.5 hover:bg-muted">
                Cart
              </Link>
              <Link href="/wishlist" className="rounded-full px-3 py-1.5 hover:bg-muted">
                Wishlist
              </Link>
              <Link href="/orders" className="rounded-full px-3 py-1.5 hover:bg-muted">
                Orders
              </Link>
              <Link href="/profile" className="rounded-full px-3 py-1.5 hover:bg-muted">
                Profile
              </Link>
              <Link href="/admin" className="rounded-full px-3 py-1.5 hover:bg-muted">
                Admin
              </Link>
              <Link href="/login" className="rounded-full px-3 py-1.5 hover:bg-muted">
                Login
              </Link>
              <Link
                href="/register"
                className="rounded-full bg-primary px-3 py-1.5 text-primary-foreground shadow-soft hover:bg-primary/90"
              >
                Register
              </Link>
            </nav>
          </div>
        </header>
        <main className="mx-auto min-w-0 max-w-7xl overflow-x-hidden px-4 py-8 sm:py-10">{children}</main>
      </body>
    </html>
  );
}
