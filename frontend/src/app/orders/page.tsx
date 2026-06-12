"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { CheckCircle2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchMyOrders } from "@/lib/api";
import { getToken } from "@/lib/auth";
import { ApiError, type PaginationMeta } from "@/types/api";
import type { Order } from "@/types/order";

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

function formatDate(date: string) {
  const parsed = new Date(date);
  return parsed.toLocaleString("vi-VN");
}

function shippingLabel(method?: string) {
  return method === "express" ? "Express delivery" : method === "standard" ? "Standard delivery" : "Not provided";
}

function paymentLabel(method?: string) {
  return method === "demo_payment" ? "Demo payment" : method === "cod" ? "COD" : "Not provided";
}

function paymentStatusLabel(status?: string) {
  return status ? status.replaceAll("_", " ") : "unknown";
}

function paymentStatusTone(status?: Order["payment_status"]) {
  switch (status) {
    case "paid":
      return "secondary";
    case "failed":
    case "refunded":
      return "destructive";
    default:
      return "outline";
  }
}

export default function OrdersPage() {
  const router = useRouter();
  const [orders, setOrders] = useState<Order[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [checkoutSuccess] = useState(
    () =>
      typeof window !== "undefined" &&
      new URLSearchParams(window.location.search).get("checkout") === "success"
  );

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login?redirect=/orders");
      return;
    }

    let cancelled = false;
    async function loadOrders() {
      setLoading(true);
      setError(null);
      try {
        const res = await fetchMyOrders({ page, limit: 20 });
        if (!cancelled) {
          setOrders(res.data ?? []);
          setMeta(res.meta ?? null);
        }
      } catch (err) {
        if (!cancelled) {
          if (err instanceof ApiError && err.status === 401) {
            router.replace("/login?redirect=/orders");
            return;
          }
          const message = err instanceof Error ? err.message : "Failed to fetch orders";
          setError(message);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadOrders();
    return () => {
      cancelled = true;
    };
  }, [page, router]);

  const totalPages = useMemo(() => meta?.total_page ?? 1, [meta]);

  if (loading) {
    return (
      <div className="grid gap-4">
        {Array.from({ length: 3 }).map((_, index) => (
          <div key={index} className="h-40 animate-pulse rounded-[1.5rem] bg-muted/80" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-7">
      <div className="surface-card flex flex-col gap-4 rounded-[2rem] p-6 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="eyebrow">Order history</p>
          <h1 className="mt-2 text-4xl font-semibold">My Orders</h1>
          <p className="mt-2 max-w-xl text-sm text-muted-foreground">
            Track checkout results, review purchased products, and inspect order status.
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href="/products">Shop more</Link>
        </Button>
      </div>

      {error ? (
        <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      {checkoutSuccess ? (
        <Card className="surface-card rounded-[1.75rem] border-primary/30 bg-secondary/45">
          <CardContent className="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-start gap-3">
              <span className="grid size-11 shrink-0 place-items-center rounded-2xl bg-primary text-primary-foreground">
                <CheckCircle2 className="size-5" />
              </span>
              <div>
                <p className="text-lg font-semibold">Order placed successfully.</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Your cart was converted into an order. You can track status here or keep shopping.
                </p>
              </div>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button asChild variant="outline">
                <Link href="/products">Continue shopping</Link>
              </Button>
              {orders[0]?.id ? (
                <Button asChild variant="outline">
                  <Link href={`/orders/${orders[0].id}`}>View latest order</Link>
                </Button>
              ) : null}
              <Button asChild>
                <Link href="/orders">Orders</Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      ) : null}

      {!error && orders.length === 0 ? (
        <Card className="surface-card">
          <CardContent className="state-panel">
            <p className="text-xl font-semibold">No orders yet.</p>
            <p className="max-w-md text-sm text-muted-foreground">
              Checkout from your cart to see order history here.
            </p>
            <Button asChild>
              <Link href="/products">Start shopping</Link>
            </Button>
          </CardContent>
        </Card>
      ) : null}

      <div className="space-y-4">
        {orders.map((order) => (
          <Card key={order.id} className="surface-card rounded-[1.5rem]">
            <CardHeader>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <CardTitle className="text-xl">Order #{order.id.slice(0, 8)}</CardTitle>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Created: {formatDate(order.created_at)}
                  </p>
                </div>
                <Badge variant="secondary" className="capitalize">
                  {order.status}
                </Badge>
                <Badge variant={paymentStatusTone(order.payment_status)} className="capitalize">
                  Payment {paymentStatusLabel(order.payment_status)}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-2xl bg-muted/60 p-4">
                <span className="text-sm text-muted-foreground">Order total</span>
                <span className="font-heading text-2xl font-semibold">
                  {formatVND(order.total_amount)}
                </span>
              </div>
              <div className="grid gap-3 rounded-2xl border border-border bg-card/70 p-4 text-sm md:grid-cols-2">
                <div>
                  <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Ship to</p>
                  <p className="mt-2 font-medium">{order.recipient_name || "Recipient not provided"}</p>
                  <p className="mt-1 text-muted-foreground">
                    {[order.address_line, order.district, order.city].filter(Boolean).join(", ") || "Address not provided"}
                  </p>
                  {order.phone ? <p className="mt-1 text-muted-foreground">{order.phone}</p> : null}
                </div>
                <div>
                  <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Fulfillment</p>
                  <p className="mt-2 font-medium">{shippingLabel(order.shipping_method)}</p>
                  <p className="mt-1 text-muted-foreground">Payment: {paymentLabel(order.payment_method)}</p>
                  <p className="mt-1 text-muted-foreground">
                    Payment status: {paymentStatusLabel(order.payment_status)}
                  </p>
                  {order.note ? <p className="mt-1 text-muted-foreground">Note: {order.note}</p> : null}
                </div>
              </div>
              <div className="grid gap-2 text-sm">
                {order.items.map((item) => (
                  <div
                    key={item.id}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border bg-card/70 px-3 py-2"
                  >
                    <span>
                      {item.quantity} x {item.product.name}
                    </span>
                    <span className="font-medium">{formatVND(item.unit_price)}</span>
                  </div>
                ))}
              </div>
              <div className="flex flex-wrap gap-2">
                <Button asChild>
                  <Link href={`/orders/${order.id}`}>View details</Link>
                </Button>
                <Button asChild variant="outline">
                  <Link href="/products">Continue shopping</Link>
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {orders.length > 0 ? (
        <div className="surface-card flex items-center justify-between rounded-[1.5rem] p-4">
          <p className="text-sm text-muted-foreground">
            Page {meta?.page ?? 1} / {totalPages}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => setPage((prev) => Math.max(1, prev - 1))}
              disabled={(meta?.page ?? 1) <= 1}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              onClick={() => setPage((prev) => prev + 1)}
              disabled={(meta?.page ?? 1) >= totalPages}
            >
              Next
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
