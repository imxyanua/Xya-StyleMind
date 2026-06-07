import Link from "next/link";

import { AdminGuard } from "@/components/admin/admin-guard";

export default function AdminLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <AdminGuard>
      <div className="space-y-7">
        <section className="surface-card rounded-[2rem] p-6">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p className="eyebrow">Commerce operations</p>
              <h1 className="mt-2 text-4xl font-semibold">Admin Dashboard</h1>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
                Manage catalog structure, product inventory, and order status from one focused
                workspace.
              </p>
            </div>
            <nav className="flex flex-wrap gap-2 text-sm">
              <Link
                href="/admin/categories"
                className="rounded-full border border-border bg-card px-4 py-2 font-medium hover:bg-muted"
              >
                Categories
              </Link>
              <Link
                href="/admin/products"
                className="rounded-full border border-border bg-card px-4 py-2 font-medium hover:bg-muted"
              >
                Products
              </Link>
              <Link
                href="/admin/orders"
                className="rounded-full border border-border bg-card px-4 py-2 font-medium hover:bg-muted"
              >
                Orders
              </Link>
              <Link
                href="/admin/users"
                className="rounded-full border border-border bg-card px-4 py-2 font-medium hover:bg-muted"
              >
                Users
              </Link>
              <Link
                href="/admin/activity"
                className="rounded-full border border-border bg-card px-4 py-2 font-medium hover:bg-muted"
              >
                Activity
              </Link>
            </nav>
          </div>
        </section>
        {children}
      </div>
    </AdminGuard>
  );
}
