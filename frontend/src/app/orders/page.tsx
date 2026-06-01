"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

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

export default function OrdersPage() {
  const router = useRouter();
  const [orders, setOrders] = useState<Order[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
    return <p className="text-sm text-muted-foreground">Loading orders...</p>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">My Orders</h1>
        <Button variant="outline" asChild>
          <Link href="/products">Shop more</Link>
        </Button>
      </div>

      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      {!error && orders.length === 0 ? (
        <Card>
          <CardContent className="py-8">
            <p className="text-sm text-muted-foreground">No orders yet.</p>
          </CardContent>
        </Card>
      ) : null}

      <div className="space-y-4">
        {orders.map((order) => (
          <Card key={order.id}>
            <CardHeader>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <CardTitle className="text-base">Order #{order.id.slice(0, 8)}</CardTitle>
                <Badge variant="secondary">{order.status}</Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Created: {formatDate(order.created_at)}
              </p>
              <p className="font-medium">Total: {formatVND(order.total_amount)}</p>
              <div className="space-y-1 text-sm">
                {order.items.map((item) => (
                  <p key={item.id}>
                    {item.quantity} x {item.product.name} ({formatVND(item.unit_price)})
                  </p>
                ))}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {orders.length > 0 ? (
        <div className="flex items-center justify-between">
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
