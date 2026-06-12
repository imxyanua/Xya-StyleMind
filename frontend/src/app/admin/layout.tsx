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
        <section className="surface-card rounded-[2rem] p-5 sm:p-6">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="min-w-0">
              <p className="eyebrow">Commerce operations</p>
              <h1 className="mt-2 text-3xl font-semibold sm:text-4xl">Admin Dashboard</h1>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
                Manage catalog structure, product inventory, and order status from one focused
                workspace.
              </p>
            </div>
            <nav className="-mx-1 flex gap-2 overflow-x-auto px-1 pb-1 text-sm sm:flex-wrap sm:overflow-visible sm:pb-0">
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
                href="/admin/returns"
                className="rounded-full border border-border bg-card px-4 py-2 font-medium hover:bg-muted"
              >
                Returns
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
