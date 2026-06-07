"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { fetchAdminAuditLogs, type AdminAuditLogListParams } from "@/lib/api";
import type { PaginationMeta } from "@/types/api";
import type { AuditLog } from "@/types/audit";

type FilterState = {
  action: string;
  resourceType: "" | "product" | "category" | "order" | "user";
  result: "" | "success" | "failed";
  from: string;
  to: string;
  sort: "newest" | "oldest";
};

const initialFilters: FilterState = {
  action: "",
  resourceType: "",
  result: "",
  from: "",
  to: "",
  sort: "newest",
};

const commonActions = [
  "admin.product.create",
  "admin.product.update",
  "admin.product.delete",
  "admin.category.create",
  "admin.order_status.update",
  "admin.user_role.update",
];

function buildParams(filters: FilterState, page: number): AdminAuditLogListParams {
  return {
    page,
    limit: 20,
    action: filters.action || undefined,
    resource_type: filters.resourceType || undefined,
    result: filters.result || undefined,
    from: filters.from || undefined,
    to: filters.to || undefined,
    sort: filters.sort,
  };
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

function metadataSummary(metadata?: Record<string, unknown>) {
  if (!metadata || Object.keys(metadata).length === 0) {
    return "{}";
  }
  return JSON.stringify(metadata, null, 2);
}

export default function AdminActivityPage() {
  const [filters, setFilters] = useState<FilterState>(initialFilters);
  const [appliedFilters, setAppliedFilters] = useState<FilterState>(initialFilters);
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | undefined>();
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function loadLogs(nextPage = page, nextFilters = appliedFilters) {
    setLoading(true);
    setError(null);
    try {
      const response = await fetchAdminAuditLogs(buildParams(nextFilters, nextPage));
      setLogs(response.data ?? []);
      setMeta(response.meta);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load activity logs");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let active = true;
    async function loadInitialLogs() {
      try {
        const response = await fetchAdminAuditLogs(buildParams(initialFilters, 1));
        if (!active) {
          return;
        }
        setLogs(response.data ?? []);
        setMeta(response.meta);
      } catch (err) {
        if (!active) {
          return;
        }
        setError(err instanceof Error ? err.message : "Failed to load activity logs");
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }
    void loadInitialLogs();
    return () => {
      active = false;
    };
  }, []);

  const totalPages = useMemo(() => meta?.total_pages ?? 1, [meta]);

  function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPage(1);
    setAppliedFilters(filters);
    void loadLogs(1, filters);
  }

  function resetFilters() {
    setFilters(initialFilters);
    setAppliedFilters(initialFilters);
    setPage(1);
    void loadLogs(1, initialFilters);
  }

  function goToPage(nextPage: number) {
    setPage(nextPage);
    void loadLogs(nextPage, appliedFilters);
  }

  return (
    <div className="grid gap-6 xl:grid-cols-[0.8fr_1.6fr]">
      <Card className="surface-card rounded-[1.75rem]">
        <CardHeader>
          <p className="eyebrow">Traceability</p>
          <CardTitle className="text-3xl">Activity Filters</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={applyFilters}>
            <div className="space-y-2">
              <label htmlFor="action" className="text-sm font-medium">Action</label>
              <Input
                id="action"
                list="admin-actions"
                value={filters.action}
                onChange={(event) => setFilters((current) => ({ ...current, action: event.target.value }))}
                placeholder="admin.order_status.update"
              />
              <datalist id="admin-actions">
                {commonActions.map((action) => (
                  <option key={action} value={action} />
                ))}
              </datalist>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <label htmlFor="resource_type" className="text-sm font-medium">Resource</label>
                <select
                  id="resource_type"
                  className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={filters.resourceType}
                  onChange={(event) => setFilters((current) => ({ ...current, resourceType: event.target.value as FilterState["resourceType"] }))}
                >
                  <option value="">All</option>
                  <option value="product">Product</option>
                  <option value="category">Category</option>
                  <option value="order">Order</option>
                  <option value="user">User</option>
                </select>
              </div>
              <div className="space-y-2">
                <label htmlFor="result" className="text-sm font-medium">Result</label>
                <select
                  id="result"
                  className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                  value={filters.result}
                  onChange={(event) => setFilters((current) => ({ ...current, result: event.target.value as FilterState["result"] }))}
                >
                  <option value="">All</option>
                  <option value="success">Success</option>
                  <option value="failed">Failed</option>
                </select>
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <label htmlFor="from" className="text-sm font-medium">From</label>
                <Input id="from" type="date" value={filters.from} onChange={(event) => setFilters((current) => ({ ...current, from: event.target.value }))} />
              </div>
              <div className="space-y-2">
                <label htmlFor="to" className="text-sm font-medium">To</label>
                <Input id="to" type="date" value={filters.to} onChange={(event) => setFilters((current) => ({ ...current, to: event.target.value }))} />
              </div>
            </div>

            <div className="space-y-2">
              <label htmlFor="sort" className="text-sm font-medium">Sort</label>
              <select
                id="sort"
                className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={filters.sort}
                onChange={(event) => setFilters((current) => ({ ...current, sort: event.target.value as FilterState["sort"] }))}
              >
                <option value="newest">Newest first</option>
                <option value="oldest">Oldest first</option>
              </select>
            </div>

            <div className="flex gap-2">
              <Button type="submit" className="flex-1" disabled={loading}>Apply</Button>
              <Button type="button" variant="outline" onClick={resetFilters} disabled={loading}>Reset</Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="surface-card rounded-[1.75rem]">
        <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="eyebrow">Admin audit trail</p>
            <h1 className="text-3xl font-semibold">Activity Log</h1>
            <p className="mt-2 text-sm text-muted-foreground">
              Sensitive admin actions are stored with actor, resource, result, request id, and safe metadata.
            </p>
          </div>
          <Badge variant="secondary">{meta?.total ?? logs.length} events</Badge>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="rounded-3xl border border-dashed border-border p-10 text-center text-sm text-muted-foreground">
              Loading activity logs...
            </div>
          ) : null}

          {error ? (
            <div className="rounded-3xl border border-destructive/30 bg-destructive/10 p-6">
              <p className="font-medium text-destructive">Could not load activity logs.</p>
              <p className="mt-2 text-sm text-muted-foreground">{error}</p>
            </div>
          ) : null}

          {!loading && !error && logs.length === 0 ? (
            <div className="rounded-3xl border border-dashed border-border p-10 text-center">
              <p className="text-xl font-semibold">No activity logs found.</p>
              <p className="mt-2 text-sm text-muted-foreground">Try clearing filters or perform an admin write action.</p>
            </div>
          ) : null}

          {!loading && !error && logs.length > 0 ? (
            <div className="space-y-3">
              {logs.map((log) => (
                <article key={log.id} className="rounded-3xl border border-border bg-card/70 p-4">
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div className="min-w-0 space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={log.result === "success" ? "secondary" : "destructive"}>{log.result}</Badge>
                        <Badge variant="outline">{log.resource_type}</Badge>
                        <span className="font-mono text-xs text-muted-foreground">{shortID(log.resource_id)}</span>
                      </div>
                      <h2 className="break-all text-lg font-semibold">{log.action}</h2>
                      <p className="text-sm text-muted-foreground">
                        Actor {shortID(log.actor_user_id)} ? {log.actor_role} ? {formatDate(log.created_at)}
                      </p>
                      <p className="font-mono text-xs text-muted-foreground">Request {log.request_id || "-"}</p>
                    </div>
                    <pre className="max-h-44 overflow-auto rounded-2xl bg-muted p-3 text-xs leading-5 text-muted-foreground lg:w-80">
                      {metadataSummary(log.metadata)}
                    </pre>
                  </div>
                </article>
              ))}

              {meta ? (
                <div className="flex flex-col gap-3 border-t border-border pt-4 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
                  <span>
                    Showing {logs.length} of {meta.total} events. Page {meta.page} of {meta.total_pages || 1}.
                  </span>
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" disabled={page <= 1 || loading} onClick={() => goToPage(page - 1)}>
                      Previous
                    </Button>
                    <Button variant="outline" size="sm" disabled={page >= totalPages || loading} onClick={() => goToPage(page + 1)}>
                      Next
                    </Button>
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
