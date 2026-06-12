"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  fetchAdminReturnRequest,
  fetchAdminReturnRequests,
  updateReturnRequestStatus,
  type AdminReturnRequestListParams,
} from "@/lib/api";
import type { PaginationMeta } from "@/types/api";
import type { ReturnRequest, ReturnRequestStatus } from "@/types/return";

type FilterState = {
  status: "" | ReturnRequestStatus;
  orderId: string;
  userId: string;
  sort: "newest" | "oldest";
};

type AdminActionStatus = "approved" | "rejected" | "cancelled";

const initialFilters: FilterState = {
  status: "",
  orderId: "",
  userId: "",
  sort: "newest",
};

const returnStatuses: ReturnRequestStatus[] = ["requested", "approved", "rejected", "cancelled"];
const adminActionStatuses: AdminActionStatus[] = ["approved", "rejected", "cancelled"];

const statusTone: Record<ReturnRequestStatus, "secondary" | "outline" | "destructive"> = {
  requested: "outline",
  approved: "secondary",
  rejected: "destructive",
  cancelled: "outline",
};

const paymentTone: Record<string, "secondary" | "outline" | "destructive"> = {
  unpaid: "outline",
  pending: "outline",
  paid: "secondary",
  failed: "destructive",
  refunded: "destructive",
};

function buildParams(filters: FilterState, page: number): AdminReturnRequestListParams {
  return {
    page,
    limit: 10,
    status: filters.status || undefined,
    order_id: filters.orderId || undefined,
    user_id: filters.userId || undefined,
    sort: filters.sort,
  };
}

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
    return "-";
  }
  return value.length > 18 ? `${value.slice(0, 8)}...${value.slice(-6)}` : value;
}

function paymentLabel(status?: string) {
  return status ? status.replaceAll("_", " ") : "unknown";
}

export default function AdminReturnsPage() {
  const [filters, setFilters] = useState<FilterState>(initialFilters);
  const [appliedFilters, setAppliedFilters] = useState<FilterState>(initialFilters);
  const [page, setPage] = useState(1);
  const [requests, setRequests] = useState<ReturnRequest[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | undefined>();
  const [selectedRequest, setSelectedRequest] = useState<ReturnRequest | null>(null);
  const [status, setStatus] = useState<AdminActionStatus>("approved");
  const [adminNote, setAdminNote] = useState("");
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function loadRequests(nextPage = page, nextFilters = appliedFilters, selectFirst = false) {
    setLoading(true);
    setError(null);
    try {
      const response = await fetchAdminReturnRequests(buildParams(nextFilters, nextPage));
      const nextRequests = response.data ?? [];
      setRequests(nextRequests);
      setMeta(response.meta);
      if (selectFirst && nextRequests[0]?.id) {
        await loadRequestDetail(nextRequests[0].id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load return requests");
    } finally {
      setLoading(false);
    }
  }

  async function loadRequestDetail(id: string) {
    setDetailLoading(true);
    setDetailError(null);
    try {
      const response = await fetchAdminReturnRequest(id);
      const request = response.data ?? null;
      setSelectedRequest(request);
      if (request?.status === "approved" || request?.status === "rejected" || request?.status === "cancelled") {
        setStatus(request.status);
      } else {
        setStatus("approved");
      }
      setAdminNote(request?.admin_note ?? "");
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : "Failed to load return detail");
    } finally {
      setDetailLoading(false);
    }
  }

  useEffect(() => {
    let active = true;
    fetchAdminReturnRequests(buildParams(initialFilters, 1))
      .then(async (response) => {
        if (!active) {
          return;
        }
        const nextRequests = response.data ?? [];
        setRequests(nextRequests);
        setMeta(response.meta);
        if (nextRequests[0]?.id) {
          setDetailLoading(true);
          try {
            const detailResponse = await fetchAdminReturnRequest(nextRequests[0].id);
            if (!active) {
              return;
            }
            const request = detailResponse.data ?? null;
            setSelectedRequest(request);
            if (
              request?.status === "approved" ||
              request?.status === "rejected" ||
              request?.status === "cancelled"
            ) {
              setStatus(request.status);
            } else {
              setStatus("approved");
            }
            setAdminNote(request?.admin_note ?? "");
          } catch (err) {
            if (active) {
              setDetailError(err instanceof Error ? err.message : "Failed to load return detail");
            }
          } finally {
            if (active) {
              setDetailLoading(false);
            }
          }
        }
      })
      .catch((err) => {
        if (active) {
          setError(err instanceof Error ? err.message : "Failed to load return requests");
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, []);

  const totalPages = useMemo(() => meta?.total_pages ?? meta?.total_page ?? 1, [meta]);

  function updateFilter<K extends keyof FilterState>(key: K, value: FilterState[K]) {
    setFilters((current) => ({ ...current, [key]: value }));
  }

  async function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSuccess(null);
    setSelectedRequest(null);
    setAppliedFilters(filters);
    setPage(1);
    await loadRequests(1, filters, true);
  }

  async function resetFilters() {
    setFilters(initialFilters);
    setAppliedFilters(initialFilters);
    setPage(1);
    setSuccess(null);
    setSelectedRequest(null);
    await loadRequests(1, initialFilters, true);
  }

  async function goToPage(nextPage: number) {
    setPage(nextPage);
    setSuccess(null);
    await loadRequests(nextPage, appliedFilters, true);
  }

  async function submitStatus(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedRequest?.id) {
      setDetailError("Select a return request first.");
      return;
    }

    if (
      (status === "approved" || status === "rejected") &&
      !window.confirm(`Confirm ${status} return request #${selectedRequest.id.slice(0, 8)}?`)
    ) {
      return;
    }

    setSaving(true);
    setDetailError(null);
    setSuccess(null);
    try {
      const response = await updateReturnRequestStatus(selectedRequest.id, {
        status,
        admin_note: adminNote.trim() || undefined,
      });
      const updated = response.data ?? null;
      setSelectedRequest(updated);
      setSuccess(
        status === "approved"
          ? "Return approved. Payment status is refunded when the order payment was paid."
          : "Return request updated."
      );
      await loadRequests(page, appliedFilters);
      if (updated?.id) {
        await loadRequestDetail(updated.id);
      }
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : "Failed to update return request");
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
                <p className="eyebrow">After-sales operations</p>
                <CardTitle className="mt-2 text-3xl">Return Requests</CardTitle>
              </div>
              <Badge variant="secondary">{meta?.total ?? requests.length} requests</Badge>
            </div>
          </CardHeader>
          <CardContent>
            <form className="grid gap-3 lg:grid-cols-[150px_1fr_1fr_150px]" onSubmit={applyFilters}>
              <div className="space-y-1.5">
                <label htmlFor="return-status-filter" className="text-sm font-medium">
                  Status
                </label>
                <select
                  id="return-status-filter"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm capitalize outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.status}
                  onChange={(event) => updateFilter("status", event.target.value as FilterState["status"])}
                >
                  <option value="">All</option>
                  {returnStatuses.map((item) => (
                    <option key={item} value={item}>
                      {item}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="return-order-filter" className="text-sm font-medium">
                  Order ID
                </label>
                <Input
                  id="return-order-filter"
                  value={filters.orderId}
                  onChange={(event) => updateFilter("orderId", event.target.value)}
                  placeholder="Optional order UUID"
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="return-user-filter" className="text-sm font-medium">
                  User ID
                </label>
                <Input
                  id="return-user-filter"
                  value={filters.userId}
                  onChange={(event) => updateFilter("userId", event.target.value)}
                  placeholder="Optional user UUID"
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="return-sort-filter" className="text-sm font-medium">
                  Sort
                </label>
                <select
                  id="return-sort-filter"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.sort}
                  onChange={(event) => updateFilter("sort", event.target.value as FilterState["sort"])}
                >
                  <option value="newest">Newest</option>
                  <option value="oldest">Oldest</option>
                </select>
              </div>
              <div className="flex gap-2 lg:col-span-4">
                <Button type="submit" disabled={loading}>
                  Apply filters
                </Button>
                <Button type="button" variant="outline" onClick={resetFilters} disabled={loading}>
                  Reset
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow">Customer requests</p>
              <CardTitle className="text-3xl">Review Queue</CardTitle>
            </div>
            {meta ? (
              <p className="text-sm text-muted-foreground">
                Page {meta.page} of {totalPages || 1}
              </p>
            ) : null}
          </CardHeader>
          <CardContent>
            {error ? (
              <div className="state-panel border-destructive/30 bg-destructive/10 text-destructive">
                <p className="text-xl font-semibold">Could not load return requests.</p>
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

            {!loading && !error && requests.length === 0 ? (
              <div className="state-panel">
                <p className="text-xl font-semibold">No return requests found.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  New customer refund requests will appear here after they are submitted from an
                  eligible order.
                </p>
              </div>
            ) : null}

            {!loading && !error && requests.length > 0 ? (
              <div className="overflow-x-auto rounded-2xl border border-border">
                <div className="hidden grid-cols-[1fr_1fr_120px_120px] gap-4 bg-muted/60 px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground lg:grid">
                  <span>Request</span>
                  <span>Order/User</span>
                  <span>Status</span>
                  <span className="text-right">Payment</span>
                </div>
                <div className="divide-y divide-border">
                  {requests.map((request) => (
                    <button
                      key={request.id}
                      type="button"
                      className="grid w-full min-w-0 gap-4 p-4 text-left transition hover:bg-muted/50 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_120px_120px] lg:items-center"
                      onClick={() => loadRequestDetail(request.id)}
                    >
                      <div className="min-w-0">
                        <p className="font-mono text-sm font-semibold">{shortID(request.id)}</p>
                        <p className="mt-1 line-clamp-1 text-xs text-muted-foreground">{request.reason}</p>
                        <p className="mt-1 text-xs text-muted-foreground">{formatDate(request.created_at)}</p>
                      </div>
                      <div className="min-w-0">
                        <p className="font-mono text-sm">{shortID(request.order_id)}</p>
                        <p className="mt-1 truncate text-xs text-muted-foreground">
                          {request.user?.email ?? request.user_id}
                        </p>
                      </div>
                      <Badge variant={statusTone[request.status]} className="w-fit capitalize">
                        {request.status}
                      </Badge>
                      <Badge
                        variant={paymentTone[request.order?.payment_status ?? "unpaid"]}
                        className="w-fit capitalize lg:justify-self-end"
                      >
                        {paymentLabel(request.order?.payment_status)}
                      </Badge>
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
                  Showing {requests.length} of {meta.total}
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
            <p className="eyebrow">Admin decision</p>
            <CardTitle className="text-3xl">Review Return</CardTitle>
          </CardHeader>
          <CardContent>
            {detailLoading ? <div className="h-44 animate-pulse rounded-2xl bg-muted/80" /> : null}
            {detailError ? (
              <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {detailError}
              </p>
            ) : null}
            {success ? <p className="mb-3 text-sm text-primary">{success}</p> : null}
            {!detailLoading && !selectedRequest ? (
              <div className="state-panel">
                <p className="text-xl font-semibold">No request selected.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Choose a return request from the queue to review customer reason and order state.
                </p>
              </div>
            ) : null}
            {!detailLoading && selectedRequest ? (
              <div className="space-y-5">
                <div className="rounded-2xl border border-border bg-card/70 p-4">
                  <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">Return request</p>
                  <p className="mt-2 break-all font-mono text-sm">{selectedRequest.id}</p>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Badge variant={statusTone[selectedRequest.status]} className="capitalize">
                      {selectedRequest.status}
                    </Badge>
                    {selectedRequest.order?.payment_status ? (
                      <Badge
                        variant={paymentTone[selectedRequest.order.payment_status]}
                        className="capitalize"
                      >
                        Payment {paymentLabel(selectedRequest.order.payment_status)}
                      </Badge>
                    ) : null}
                  </div>
                </div>

                <div className="rounded-2xl bg-muted/60 p-4">
                  <p className="text-sm text-muted-foreground">Customer reason</p>
                  <p className="mt-2 whitespace-pre-wrap text-sm leading-6">{selectedRequest.reason}</p>
                </div>

                {selectedRequest.admin_note ? (
                  <div className="rounded-2xl border border-border bg-card/70 p-4">
                    <p className="text-sm text-muted-foreground">Previous admin note</p>
                    <p className="mt-2 whitespace-pre-wrap text-sm leading-6">{selectedRequest.admin_note}</p>
                  </div>
                ) : null}

                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="rounded-2xl border border-border bg-card/70 p-4">
                    <p className="text-sm text-muted-foreground">Order</p>
                    <Link
                      href={`/admin/orders`}
                      className="mt-2 block break-all font-mono text-sm font-medium hover:underline"
                    >
                      {selectedRequest.order_id}
                    </Link>
                    <p className="mt-2 text-sm text-muted-foreground">
                      Status: {selectedRequest.order?.status ?? "unknown"}
                    </p>
                    <p className="mt-1 text-sm text-muted-foreground">
                      Total: {formatVND(selectedRequest.order?.total_amount)}
                    </p>
                  </div>
                  <div className="rounded-2xl border border-border bg-card/70 p-4">
                    <p className="text-sm text-muted-foreground">Customer</p>
                    <p className="mt-2 font-medium">{selectedRequest.user?.full_name ?? "Unknown user"}</p>
                    <p className="mt-1 break-all text-sm text-muted-foreground">
                      {selectedRequest.user?.email ?? selectedRequest.user_id}
                    </p>
                  </div>
                </div>

                <form className="space-y-4" onSubmit={submitStatus}>
                  <div className="space-y-1.5">
                    <label htmlFor="return-decision" className="text-sm font-medium">
                      Decision
                    </label>
                    <select
                      id="return-decision"
                      className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm capitalize outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                      value={status}
                      onChange={(event) => setStatus(event.target.value as AdminActionStatus)}
                    >
                      {adminActionStatuses.map((item) => (
                        <option key={item} value={item}>
                          {item}
                        </option>
                      ))}
                    </select>
                    <p className="text-xs text-muted-foreground">
                      Approving a paid order marks payment as refunded. Rejected/cancelled requests
                      keep payment unchanged.
                    </p>
                  </div>
                  <div className="space-y-1.5">
                    <label htmlFor="admin-note" className="text-sm font-medium">
                      Admin note
                    </label>
                    <textarea
                      id="admin-note"
                      className="min-h-28 w-full rounded-2xl border border-input bg-card px-3 py-2 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                      value={adminNote}
                      onChange={(event) => setAdminNote(event.target.value)}
                      maxLength={1000}
                      placeholder="Optional internal/customer-safe note."
                    />
                    <p className="text-xs text-muted-foreground">{adminNote.length}/1000 characters</p>
                  </div>
                  <Button type="submit" disabled={saving || selectedRequest.status !== "requested"}>
                    {saving ? "Updating..." : "Update return request"}
                  </Button>
                  {selectedRequest.status !== "requested" ? (
                    <p className="text-xs text-muted-foreground">
                      This request has already been reviewed and cannot be changed again.
                    </p>
                  ) : null}
                </form>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
