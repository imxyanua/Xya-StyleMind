"use client";

import { FormEvent, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { updateOrderStatus } from "@/lib/api";
import type { Order } from "@/types/order";

const statuses: Order["status"][] = ["pending", "paid", "shipping", "completed", "cancelled"];

const statusHints: Record<Order["status"], string> = {
  pending: "Order was created and is waiting for payment or review.",
  paid: "Payment has been confirmed and stock has already been reserved.",
  shipping: "Order is being fulfilled or handed to logistics.",
  completed: "Customer has received the order.",
  cancelled: "Order is closed and should not continue fulfillment.",
};

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

export default function AdminOrdersPage() {
  const [orderId, setOrderId] = useState("");
  const [status, setStatus] = useState<Order["status"]>("pending");
  const [order, setOrder] = useState<Order | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError(null);
    setOrder(null);

    try {
      const res = await updateOrderStatus(orderId, status);
      setOrder(res.data ?? null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update order status");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="grid gap-6 lg:grid-cols-[420px_1fr]">
      <Card className="surface-card h-fit rounded-[1.75rem] lg:sticky lg:top-28">
        <CardHeader>
          <p className="eyebrow">Fulfillment control</p>
          <CardTitle className="text-3xl">Update Order Status</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-1.5">
              <label htmlFor="orderId" className="text-sm font-medium">
                Order ID
              </label>
              <Input
                id="orderId"
                value={orderId}
                onChange={(event) => setOrderId(event.target.value)}
                placeholder="Paste full order UUID"
                required
              />
              <p className="text-xs text-muted-foreground">
                Use the exact order ID from customer order history or admin tooling.
              </p>
            </div>
            <div className="space-y-1.5">
              <label htmlFor="status" className="text-sm font-medium">
                Status
              </label>
              <select
                id="status"
                className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm capitalize outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                value={status}
                onChange={(event) => setStatus(event.target.value as Order["status"])}
              >
                {statuses.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </select>
              <p className="text-xs text-muted-foreground">{statusHints[status]}</p>
            </div>
            {error ? (
              <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </p>
            ) : null}
            <Button type="submit" disabled={saving}>
              {saving ? "Updating..." : "Update status"}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card className="surface-card rounded-[1.75rem]">
        <CardHeader>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow">Mutation result</p>
              <CardTitle className="mt-2 text-3xl">Last Updated Order</CardTitle>
            </div>
            {order ? (
              <Badge variant="secondary" className="capitalize">
                {order.status}
              </Badge>
            ) : null}
          </div>
        </CardHeader>
        <CardContent>
          {!order ? (
            <div className="state-panel">
              <p className="text-xl font-semibold">No order updated yet.</p>
              <p className="max-w-md text-sm text-muted-foreground">
                Submit an order ID and status to preview the updated order response here.
              </p>
            </div>
          ) : (
            <div className="space-y-5">
              <div className="rounded-2xl border border-border bg-card/70 p-4">
                <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">Order ID</p>
                <p className="mt-2 break-all font-mono text-sm">{order.id}</p>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-2xl bg-muted/60 p-4">
                  <p className="text-sm text-muted-foreground">Current status</p>
                  <p className="mt-2 text-lg font-semibold capitalize">{order.status}</p>
                </div>
                <div className="rounded-2xl bg-muted/60 p-4">
                  <p className="text-sm text-muted-foreground">Total amount</p>
                  <p className="mt-2 font-heading text-2xl font-semibold">
                    {formatVND(order.total_amount)}
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <p className="text-sm font-medium">Items</p>
                {order.items.map((item) => (
                  <div
                    key={item.id}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border bg-card/70 px-3 py-2 text-sm"
                  >
                    <span>
                      {item.quantity} x {item.product.name}
                    </span>
                    <span className="font-medium">{formatVND(item.subtotal)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
