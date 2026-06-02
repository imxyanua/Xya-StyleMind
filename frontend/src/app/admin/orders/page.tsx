"use client";

import { FormEvent, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { updateOrderStatus } from "@/lib/api";
import type { Order } from "@/types/order";

const statuses: Order["status"][] = ["pending", "paid", "shipping", "completed", "cancelled"];

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
      <Card>
        <CardHeader>
          <CardTitle>Update Order Status</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-1">
              <label htmlFor="orderId" className="text-sm font-medium">
                Order ID
              </label>
              <Input id="orderId" value={orderId} onChange={(event) => setOrderId(event.target.value)} required />
            </div>
            <div className="space-y-1">
              <label htmlFor="status" className="text-sm font-medium">
                Status
              </label>
              <select
                id="status"
                className="h-8 w-full rounded-lg border border-input bg-background px-3 text-sm"
                value={status}
                onChange={(event) => setStatus(event.target.value as Order["status"])}
              >
                {statuses.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </select>
            </div>
            {error ? <p className="text-sm text-red-600">{error}</p> : null}
            <Button type="submit" disabled={saving}>
              {saving ? "Updating..." : "Update status"}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Last Updated Order</CardTitle>
        </CardHeader>
        <CardContent>
          {!order ? (
            <p className="text-sm text-muted-foreground">No order updated yet.</p>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center justify-between gap-3">
                <span className="font-medium">#{order.id}</span>
                <Badge variant="secondary">{order.status}</Badge>
              </div>
              <p className="text-sm">Total: {formatVND(order.total_amount)}</p>
              <div className="space-y-1 text-sm text-muted-foreground">
                {order.items.map((item) => (
                  <p key={item.id}>
                    {item.quantity} x {item.product.name}
                  </p>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
