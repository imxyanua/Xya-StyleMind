"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { OrdersDonutChart, RevenueLineChart, TopProductsBarChart } from "@/components/admin/dashboard-charts";
import { fetchAdminDashboardStats } from "@/lib/api";
import type { AdminDashboardStats } from "@/types/dashboard";

const adminLinks = [
  {
    href: "/admin/categories",
    title: "Categories",
    description: "Create product categories for catalog organization.",
  },
  {
    href: "/admin/products",
    title: "Products",
    description: "Create, update, and remove catalog products.",
  },
  {
    href: "/admin/orders",
    title: "Orders",
    description: "Filter orders, inspect detail, and update fulfillment status.",
  },
  {
    href: "/admin/activity",
    title: "Activity",
    description: "Trace sensitive admin actions with actor, resource, result, and request id.",
  },
];

function formatVND(value?: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
    maximumFractionDigits: 0,
  }).format(value ?? 0);
}

function formatNumber(value?: number) {
  return new Intl.NumberFormat("vi-VN").format(value ?? 0);
}

function formatDate(value?: string) {
  if (!value) {
    return "Unknown";
  }
  return new Intl.DateTimeFormat("vi-VN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function shortID(value?: string) {
  if (!value) {
    return "-";
  }
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

function metricCards(stats: AdminDashboardStats) {
  return [
    {
      label: "Revenue",
      value: formatVND(stats.total_revenue),
      hint: "Paid, shipping, and completed orders",
      href: "/admin/orders?status=paid",
    },
    {
      label: "Orders",
      value: formatNumber(stats.total_orders),
      hint: "Orders in selected range",
      href: "/admin/orders",
    },
    {
      label: "Products",
      value: formatNumber(stats.total_products),
      hint: "Catalog records",
      href: "/admin/products",
    },
    {
      label: "Reservations",
      value: formatNumber(stats.active_reservations),
      hint: "Active inventory holds",
      href: "/admin/products",
    },
    {
      label: "Users",
      value: formatNumber(stats.total_users),
      hint: "Registered accounts",
      href: "/admin/activity",
    },
  ];
}

export default function AdminPage() {
  const [stats, setStats] = useState<AdminDashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    async function loadStats() {
      try {
        const response = await fetchAdminDashboardStats();
        if (!active) {
          return;
        }
        setStats(response.data ?? null);
      } catch (err) {
        if (!active) {
          return;
        }
        setError(err instanceof Error ? err.message : "Failed to load dashboard stats");
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }
    void loadStats();
    return () => {
      active = false;
    };
  }, []);

  if (loading) {
    return (
      <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Card key={index} className="surface-card rounded-[1.75rem]">
            <CardHeader>
              <div className="h-4 w-24 animate-pulse rounded-full bg-muted" />
              <div className="h-9 w-32 animate-pulse rounded-full bg-muted" />
            </CardHeader>
            <CardContent>
              <div className="h-4 w-full animate-pulse rounded-full bg-muted" />
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (error || !stats) {
    return (
      <Card className="surface-card rounded-[1.75rem]">
        <CardHeader>
          <CardTitle className="text-2xl text-destructive">Could not load dashboard stats.</CardTitle>
          <CardDescription>{error ?? "No dashboard data returned."}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button onClick={() => window.location.reload()}>Retry</Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
        {metricCards(stats).map((card) => (
          <Card key={card.label} className="surface-card rounded-[1.75rem] transition hover:-translate-y-1">
            <CardHeader>
              <div className="flex items-center justify-between gap-3">
                <p className="eyebrow">{card.label}</p>
                <Button asChild variant="outline" size="sm">
                  <Link href={card.href}>Open</Link>
                </Button>
              </div>
              <CardTitle className="text-3xl">{card.value}</CardTitle>
              <CardDescription>{card.hint}</CardDescription>
            </CardHeader>
          </Card>
        ))}
      </section>

      <section className="grid gap-5 xl:grid-cols-[0.9fr_1.1fr]">
        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <p className="eyebrow">Operations</p>
            <CardTitle className="text-2xl">Orders by Status</CardTitle>
            <CardDescription>Donut view of the live order lifecycle distribution.</CardDescription>
          </CardHeader>
          <CardContent>
            <OrdersDonutChart data={stats.orders_by_status} />
          </CardContent>
        </Card>

        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow">Revenue trend</p>
              <CardTitle className="text-2xl">Revenue by Day</CardTitle>
              <CardDescription>Recent paid, shipping, and completed order revenue.</CardDescription>
            </div>
            <Button asChild variant="outline" size="sm">
              <Link href="/admin/orders">View orders</Link>
            </Button>
          </CardHeader>
          <CardContent>
            <RevenueLineChart data={stats.revenue_by_day} />
          </CardContent>
        </Card>
      </section>

      <section className="grid gap-5 xl:grid-cols-3">
        <Card className="surface-card rounded-[1.75rem] xl:col-span-2">
          <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow">Fulfillment pulse</p>
              <CardTitle className="text-2xl">Recent Orders</CardTitle>
            </div>
            <Button asChild variant="outline" size="sm">
              <Link href="/admin/orders">Manage orders</Link>
            </Button>
          </CardHeader>
          <CardContent className="space-y-3">
            {stats.recent_orders.length === 0 ? (
              <p className="rounded-3xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
                No orders yet.
              </p>
            ) : (
              stats.recent_orders.map((order) => (
                <Link
                  key={order.id}
                  href={`/admin/orders?q=${encodeURIComponent(order.id ?? "")}`}
                  className="flex flex-col gap-2 rounded-3xl border border-border bg-card/70 p-4 transition hover:bg-muted/60 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div>
                    <p className="font-mono text-xs text-muted-foreground">{shortID(order.id)}</p>
                    <p className="font-medium">{order.user_email}</p>
                    <p className="text-xs text-muted-foreground">{formatDate(order.created_at)}</p>
                  </div>
                  <div className="flex items-center gap-3">
                    <Badge variant="outline">{order.status}</Badge>
                    <span className="font-semibold">{formatVND(order.total_amount)}</span>
                  </div>
                </Link>
              ))
            )}
          </CardContent>
        </Card>

        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <p className="eyebrow">Inventory risk</p>
            <CardTitle className="text-2xl">Low-Stock Products</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {stats.low_stock_products.length === 0 ? (
              <p className="rounded-3xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
                No low-stock products.
              </p>
            ) : (
              stats.low_stock_products.map((product) => (
                <Link
                  key={product.id}
                  href="/admin/products"
                  className="flex items-center justify-between gap-3 rounded-2xl bg-muted/60 px-4 py-3 text-sm transition hover:bg-muted"
                >
                  <span className="line-clamp-1 font-medium">{product.name}</span>
                  <div className="flex flex-col items-end gap-1">
                    <Badge variant={(product.available_stock ?? product.stock ?? 0) <= 2 ? "destructive" : "secondary"}>
                      Available {product.available_stock ?? product.stock ?? 0}
                    </Badge>
                    {product.reserved_quantity ? (
                      <span className="text-xs text-muted-foreground">
                        {product.reserved_quantity} reserved / {product.stock ?? 0} stock
                      </span>
                    ) : null}
                  </div>
                </Link>
              ))
            )}
          </CardContent>
        </Card>
      </section>

      <section className="grid gap-5 xl:grid-cols-[1fr_0.8fr]">
        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <p className="eyebrow">Merchandising</p>
            <CardTitle className="text-2xl">Top Products</CardTitle>
            <CardDescription>Revenue-ranked bar chart from checkout data.</CardDescription>
          </CardHeader>
          <CardContent>
            <TopProductsBarChart data={stats.top_products} />
          </CardContent>
        </Card>

        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
          {adminLinks.map((item) => (
            <Card key={item.href} className="surface-card rounded-[1.5rem]">
              <CardHeader>
                <CardTitle className="text-xl">{item.title}</CardTitle>
                <CardDescription>{item.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <Button asChild variant="outline" className="w-full">
                  <Link href={item.href}>Open</Link>
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>
    </div>
  );
}
