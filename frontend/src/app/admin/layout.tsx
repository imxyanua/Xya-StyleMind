import Link from "next/link";

import { AdminGuard } from "@/components/admin/admin-guard";

export default function AdminLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <AdminGuard>
      <div className="space-y-6">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h1 className="text-2xl font-semibold">Admin</h1>
          <nav className="flex gap-4 text-sm">
            <Link href="/admin/categories" className="hover:underline">
              Categories
            </Link>
            <Link href="/admin/products" className="hover:underline">
              Products
            </Link>
            <Link href="/admin/orders" className="hover:underline">
              Orders
            </Link>
          </nav>
        </div>
        {children}
      </div>
    </AdminGuard>
  );
}
