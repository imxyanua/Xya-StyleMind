"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  fetchAdminOrder,
  fetchAdminOrders,
  updateOrderStatus,
  type AdminOrderListParams,
} from "@/lib/api";
import type { PaginationMeta } from "@/types/api";
import type { Order } from "@/types/order";

type OrderStatus = NonNullable<Order["status"]>;

const statuses: OrderStatus[] = ["pending", "paid", "shipping", "completed", "cancelled"];

const statusHints: Record<OrderStatus, string> = {
  pending: "Order was created and is waiting for payment or review.",
  paid: "Payment has been confirmed and stock has already been reserved.",
  shipping: "Order is being fulfilled or handed to logistics.",
  completed: "Customer has received the order.",
  cancelled: "Order is closed and should not continue fulfillment.",
};

const statusTone: Record<OrderStatus, "secondary" | "outline" | "destructive"> = {
  pending: "secondary",
  paid: "outline",
  shipping: "outline",
  completed: "secondary",
  cancelled: "destructive",
};

function formatVND(value?: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value ?? 0);
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
    return "";
  }
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

function shippingLabel(method?: string) {
  return method === "express" ? "Express delivery" : method === "standard" ? "Standard delivery" : "Not provided";
}

function paymentLabel(method?: string) {
  return method === "demo_payment" ? "Demo payment" : method === "cod" ? "COD" : "Not provided";
}

function buildParams(
  filters: FilterState,
  page: number
): AdminOrderListParams {
  return {
    page,
    limit: 10,
    q: filters.query || undefined,
    status: filters.status || undefined,
    user_id: filters.userId || undefined,
    from: filters.from || undefined,
    to: filters.to || undefined,
    sort: filters.sort,
  };
}

type FilterState = {
  query: string;
  status: "" | OrderStatus;
  userId: string;
  from: string;
  to: string;
  sort: "newest" | "oldest";
};

const initialFilters: FilterState = {
  query: "",
  status: "",
  userId: "",
  from: "",
  to: "",
  sort: "newest",
};

export default function AdminOrdersPage() {
  const [filters, setFilters] = useState<FilterState>(initialFilters);
  const [appliedFilters, setAppliedFilters] = useState<FilterState>(initialFilters);
  const [page, setPage] = useState(1);
  const [orders, setOrders] = useState<Order[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | undefined>();
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const [orderId, setOrderId] = useState("");
  const [status, setStatus] = useState<OrderStatus>("pending");
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function loadOrders(nextPage = page, nextFilters = appliedFilters, selectFirst = false) {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchAdminOrders(buildParams(nextFilters, nextPage));
      setOrders(res.data ?? []);
      setMeta(res.meta);
      if (selectFirst && res.data?.[0]?.id) {
        await loadOrderDetail(res.data[0].id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load admin orders");
    } finally {
      setLoading(false);
    }
  }

  async function loadOrderDetail(id: string) {
    setDetailLoading(true);
    setDetailError(null);
    try {
      const res = await fetchAdminOrder(id);
      const order = res.data ?? null;
      setSelectedOrder(order);
      if (order?.id) {
        setOrderId(order.id);
      }
      if (order?.status) {
        setStatus(order.status as OrderStatus);
      }
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : "Failed to load order detail");
    } finally {
      setDetailLoading(false);
    }
  }

  useEffect(() => {
    void (async () => {
      await loadOrders(1, initialFilters, true);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const totalPages = meta?.total_pages ?? meta?.total_page ?? 1;

  const selectedUserLabel = useMemo(() => {
    if (!selectedOrder?.user) {
      return selectedOrder?.user_id ?? "Unknown user";
    }
    return `${selectedOrder.user.full_name || "Unnamed user"} · ${selectedOrder.user.email}`;
  }, [selectedOrder]);

  function updateFilter<K extends keyof FilterState>(key: K, value: FilterState[K]) {
    setFilters((prev) => ({ ...prev, [key]: value }));
  }

  async function onFilterSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSuccess(null);
    setSelectedOrder(null);
    setAppliedFilters(filters);
    setPage(1);
    await loadOrders(1, filters, true);
  }

  async function clearFilters() {
    setFilters(initialFilters);
    setAppliedFilters(initialFilters);
    setPage(1);
    setSelectedOrder(null);
    setSuccess(null);
    await loadOrders(1, initialFilters, true);
  }

  async function goToPage(nextPage: number) {
    setPage(nextPage);
    setSuccess(null);
    await loadOrders(nextPage, appliedFilters, true);
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError(null);
    setDetailError(null);
    setSuccess(null);

    try {
      const res = await updateOrderStatus(orderId, status);
      const updated = res.data ?? null;
      setSelectedOrder(updated);
      setSuccess("Order status updated.");
      await loadOrders(page, appliedFilters);
      if (updated?.id) {
        await loadOrderDetail(updated.id);
      }
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : "Failed to update order status");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="grid min-w-0 gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-6">
        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <p className="eyebrow">Fulfillment queue</p>
                <CardTitle className="mt-2 text-3xl">Admin Orders</CardTitle>
              </div>
              <Badge variant="secondary">{meta?.total ?? orders.length} orders</Badge>
            </div>
          </CardHeader>
          <CardContent>
            <form className="grid gap-3 lg:grid-cols-[1.2fr_150px_150px_150px]" onSubmit={onFilterSubmit}>
              <div className="space-y-1.5">
                <label htmlFor="order-search" className="text-sm font-medium">
                  Search order ID
                </label>
                <Input
                  id="order-search"
                  value={filters.query}
                  onChange={(event) => updateFilter("query", event.target.value)}
                  placeholder="Paste full or partial order ID"
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="status-filter" className="text-sm font-medium">
                  Status filter
                </label>
                <select
                  id="status-filter"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm capitalize outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.status}
                  onChange={(event) => updateFilter("status", event.target.value as FilterState["status"])}
                >
                  <option value="">All</option>
                  {statuses.map((item) => (
                    <option key={item} value={item}>
                      {item}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="sort" className="text-sm font-medium">
                  Sort
                </label>
                <select
                  id="sort"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.sort}
                  onChange={(event) => updateFilter("sort", event.target.value as FilterState["sort"])}
                >
                  <option value="newest">Newest</option>
                  <option value="oldest">Oldest</option>
                </select>
              </div>
              <div className="flex items-end gap-2">
                <Button type="submit" className="flex-1">
                  Apply
                </Button>
                <Button type="button" variant="outline" onClick={clearFilters}>
                  Clear
                </Button>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="user-id-filter" className="text-sm font-medium">
                  User ID
                </label>
                <Input
                  id="user-id-filter"
                  value={filters.userId}
                  onChange={(event) => updateFilter("userId", event.target.value)}
                  placeholder="Optional UUID"
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="from-filter" className="text-sm font-medium">
                  From
                </label>
                <Input
                  id="from-filter"
                  type="date"
                  value={filters.from}
                  onChange={(event) => updateFilter("from", event.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="to-filter" className="text-sm font-medium">
                  To
                </label>
                <Input
                  id="to-filter"
                  type="date"
                  value={filters.to}
                  onChange={(event) => updateFilter("to", event.target.value)}
                />
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <p className="eyebrow">Live backend data</p>
                <CardTitle className="mt-2 text-3xl">Order List</CardTitle>
              </div>
              {meta ? (
                <p className="text-sm text-muted-foreground">
                  Page {meta.page} of {totalPages || 1}
                </p>
              ) : null}
            </div>
          </CardHeader>
          <CardContent>
            {error ? (
              <div className="state-panel border-destructive/30 bg-destructive/10 text-destructive">
                <p className="text-xl font-semibold">Could not load orders.</p>
                <p className="text-sm">{error}</p>
              </div>
            ) : null}

            {loading ? (
              <div className="grid gap-3">
                {Array.from({ length: 5 }).map((_, index) => (
                  <div key={index} className="h-24 animate-pulse rounded-2xl bg-muted/80" />
                ))}
              </div>
            ) : null}

            {!loading && !error && orders.length === 0 ? (
              <div className="state-panel">
                <p className="text-xl font-semibold">No orders found.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Try clearing filters or create a checkout flow from the storefront.
                </p>
              </div>
            ) : null}

            {!loading && orders.length > 0 ? (
              <div className="overflow-x-auto rounded-2xl border border-border">
                <div className="hidden grid-cols-[1.2fr_1fr_110px_120px] gap-4 bg-muted/60 px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground lg:grid">
                  <span>Order</span>
                  <span>Customer</span>
                  <span>Status</span>
                  <span className="text-right">Total</span>
                </div>
                <div className="divide-y divide-border">
                  {orders.map((order) => (
                    <button
                      key={order.id}
                      type="button"
                      className="grid w-full min-w-0 gap-4 p-4 text-left transition hover:bg-muted/50 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_110px_120px] lg:items-center"
                      onClick={() => order.id && loadOrderDetail(order.id)}
                    >
                      <div className="min-w-0">
                        <p className="font-mono text-sm font-semibold">{shortID(order.id)}</p>
                        <p className="mt-1 text-xs text-muted-foreground">{formatDate(order.created_at)}</p>
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-medium">
                          {order.user?.full_name || order.user?.email || order.user_id}
                        </p>
                        <p className="mt-1 truncate text-xs text-muted-foreground">{order.user?.email}</p>
                      </div>
                      <Badge variant={statusTone[(order.status as OrderStatus) ?? "pending"]} className="capitalize">
                        {order.status}
                      </Badge>
                      <p className="font-heading text-lg font-semibold lg:text-right">
                        {formatVND(order.total_amount)}
                      </p>
                    </button>
                  ))}
                </div>
              </div>
            ) : null}

            {meta && totalPages > 1 ? (
              <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
                <Button
                  type="button"
                  variant="outline"
                  disabled={page <= 1 || loading}
                  onClick={() => goToPage(page - 1)}
                >
                  Previous
                </Button>
                <p className="text-sm text-muted-foreground">
                  Showing {orders.length} of {meta.total}
                </p>
                <Button
                  type="button"
                  variant="outline"
                  disabled={page >= totalPages || loading}
                  onClick={() => goToPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>

      <div className="space-y-6 xl:sticky xl:top-28 xl:self-start">
        <Card className="surface-card rounded-[1.75rem]">
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
                  Select an order from the list or paste the exact order ID.
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
                  onChange={(event) => setStatus(event.target.value as OrderStatus)}
                >
                  {statuses.map((item) => (
                    <option key={item} value={item}>
                      {item}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-muted-foreground">{statusHints[status]}</p>
              </div>
              {detailError ? (
                <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {detailError}
                </p>
              ) : null}
              {success ? <p className="text-sm text-primary">{success}</p> : null}
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
                <p className="eyebrow">Order detail</p>
                <CardTitle className="mt-2 text-3xl">Last Updated Order</CardTitle>
              </div>
              {selectedOrder?.status ? (
                <Badge variant={statusTone[selectedOrder.status as OrderStatus]} className="capitalize">
                  {selectedOrder.status}
                </Badge>
              ) : null}
            </div>
          </CardHeader>
          <CardContent>
            {detailLoading ? (
              <div className="space-y-3">
                <div className="h-24 animate-pulse rounded-2xl bg-muted/80" />
                <div className="h-36 animate-pulse rounded-2xl bg-muted/80" />
              </div>
            ) : null}

            {!detailLoading && !selectedOrder ? (
              <div className="state-panel">
                <p className="text-xl font-semibold">No order selected yet.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Choose an order from the list to inspect customer, items, status, and totals.
                </p>
              </div>
            ) : null}

            {!detailLoading && selectedOrder ? (
              <div className="space-y-5">
                <div className="rounded-2xl border border-border bg-card/70 p-4">
                  <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">Order ID</p>
                  <p className="mt-2 break-all font-mono text-sm">{selectedOrder.id}</p>
                  <p className="mt-3 text-sm text-muted-foreground">{formatDate(selectedOrder.created_at)}</p>
                </div>

                <div className="rounded-2xl border border-border bg-card/70 p-4">
                  <p className="text-sm text-muted-foreground">Customer</p>
                  <p className="mt-2 font-medium">{selectedUserLabel}</p>
                  <p className="mt-1 break-all font-mono text-xs text-muted-foreground">
                    {selectedOrder.user_id}
                  </p>
                </div>

                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="rounded-2xl bg-muted/60 p-4">
                    <p className="text-sm text-muted-foreground">Current status</p>
                    <p className="mt-2 text-lg font-semibold capitalize">{selectedOrder.status}</p>
                  </div>
                  <div className="rounded-2xl bg-muted/60 p-4">
                    <p className="text-sm text-muted-foreground">Total amount</p>
                    <p className="mt-2 font-heading text-2xl font-semibold">
                      {formatVND(selectedOrder.total_amount)}
                    </p>
                  </div>
                </div>

                <div className="rounded-2xl border border-border bg-card/70 p-4">
                  <p className="text-sm font-medium">Shipping & payment</p>
                  <div className="mt-3 grid gap-3 text-sm md:grid-cols-2">
                    <div>
                      <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Recipient</p>
                      <p className="mt-2 font-medium">{selectedOrder.recipient_name || "Not provided"}</p>
                      <p className="mt-1 text-muted-foreground">
                        {[selectedOrder.address_line, selectedOrder.district, selectedOrder.city]
                          .filter(Boolean)
                          .join(", ") || "Address not provided"}
                      </p>
                      {selectedOrder.phone ? <p className="mt-1 text-muted-foreground">{selectedOrder.phone}</p> : null}
                    </div>
                    <div>
                      <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Method</p>
                      <p className="mt-2 font-medium">{shippingLabel(selectedOrder.shipping_method)}</p>
                      <p className="mt-1 text-muted-foreground">Payment: {paymentLabel(selectedOrder.payment_method)}</p>
                      {selectedOrder.note ? <p className="mt-1 text-muted-foreground">Note: {selectedOrder.note}</p> : null}
                    </div>
                  </div>
                </div>

                <div className="space-y-2">
                  <p className="text-sm font-medium">Items</p>
                  {(selectedOrder.items ?? []).map((item) => (
                    <div
                      key={item.id}
                      className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border bg-card/70 px-3 py-2 text-sm"
                    >
                      <span>
                        {item.quantity} x {item.product?.name ?? item.product_id}
                      </span>
                      <span className="font-medium">{formatVND(item.subtotal)}</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
