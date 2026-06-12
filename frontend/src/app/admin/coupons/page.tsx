"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  createCoupon,
  deleteCoupon,
  fetchAdminCoupons,
  updateCoupon,
  type AdminCouponListParams,
  type CouponInput,
} from "@/lib/api";
import type { PaginationMeta } from "@/types/api";
import type { Coupon, CouponType } from "@/types/coupon";

type FilterState = {
  query: string;
  type: "" | CouponType;
  isActive: "" | "true" | "false";
  sort: "newest" | "oldest";
};

type FormState = {
  code: string;
  type: CouponType;
  value: string;
  minOrderAmount: string;
  maxDiscountAmount: string;
  usageLimit: string;
  startsAt: string;
  expiresAt: string;
  isActive: boolean;
};

const initialFilters: FilterState = {
  query: "",
  type: "",
  isActive: "",
  sort: "newest",
};

const initialForm: FormState = {
  code: "",
  type: "percent",
  value: "10",
  minOrderAmount: "0",
  maxDiscountAmount: "",
  usageLimit: "",
  startsAt: "",
  expiresAt: "",
  isActive: true,
};

function buildParams(filters: FilterState, page: number): AdminCouponListParams {
  return {
    page,
    limit: 10,
    q: filters.query || undefined,
    type: filters.type || undefined,
    is_active: filters.isActive ? filters.isActive === "true" : undefined,
    sort: filters.sort,
  };
}

function formatVND(value?: number | null) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value ?? 0);
}

function formatDateTimeLocal(value?: string) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function toIsoOrUndefined(value: string) {
  return value ? new Date(value).toISOString() : undefined;
}

function couponLabel(coupon: Coupon) {
  if (coupon.type === "percent") {
    return `${coupon.value}% off`;
  }
  return `${formatVND(coupon.value)} off`;
}

export default function AdminCouponsPage() {
  const [filters, setFilters] = useState<FilterState>(initialFilters);
  const [appliedFilters, setAppliedFilters] = useState<FilterState>(initialFilters);
  const [coupons, setCoupons] = useState<Coupon[]>([]);
  const [selectedCoupon, setSelectedCoupon] = useState<Coupon | null>(null);
  const [form, setForm] = useState<FormState>(initialForm);
  const [meta, setMeta] = useState<PaginationMeta | undefined>();
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function loadCoupons(nextPage = page, nextFilters = appliedFilters) {
    setLoading(true);
    setError(null);
    try {
      const response = await fetchAdminCoupons(buildParams(nextFilters, nextPage));
      setCoupons(response.data ?? []);
      setMeta(response.meta);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load coupons");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let active = true;
    fetchAdminCoupons(buildParams(initialFilters, 1))
      .then((response) => {
        if (!active) {
          return;
        }
        setCoupons(response.data ?? []);
        setMeta(response.meta);
      })
      .catch((err) => {
        if (active) {
          setError(err instanceof Error ? err.message : "Failed to load coupons");
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

  function updateForm<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function selectCoupon(coupon: Coupon) {
    setSelectedCoupon(coupon);
    setSuccess(null);
    setFormError(null);
    setForm({
      code: coupon.code,
      type: coupon.type,
      value: String(coupon.value),
      minOrderAmount: String(coupon.min_order_amount ?? 0),
      maxDiscountAmount: coupon.max_discount_amount ? String(coupon.max_discount_amount) : "",
      usageLimit: coupon.usage_limit ? String(coupon.usage_limit) : "",
      startsAt: formatDateTimeLocal(coupon.starts_at ?? undefined),
      expiresAt: formatDateTimeLocal(coupon.expires_at ?? undefined),
      isActive: coupon.is_active,
    });
  }

  function resetForm() {
    setSelectedCoupon(null);
    setForm(initialForm);
    setFormError(null);
    setSuccess(null);
  }

  function formToInput(): CouponInput {
    return {
      code: form.code.trim().toUpperCase(),
      type: form.type,
      value: Number(form.value),
      min_order_amount: Number(form.minOrderAmount || 0),
      max_discount_amount: form.maxDiscountAmount ? Number(form.maxDiscountAmount) : undefined,
      usage_limit: form.usageLimit ? Number(form.usageLimit) : undefined,
      starts_at: toIsoOrUndefined(form.startsAt),
      expires_at: toIsoOrUndefined(form.expiresAt),
      is_active: form.isActive,
    };
  }

  async function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAppliedFilters(filters);
    setPage(1);
    await loadCoupons(1, filters);
  }

  async function resetFilters() {
    setFilters(initialFilters);
    setAppliedFilters(initialFilters);
    setPage(1);
    await loadCoupons(1, initialFilters);
  }

  async function goToPage(nextPage: number) {
    setPage(nextPage);
    await loadCoupons(nextPage, appliedFilters);
  }

  async function submitCoupon(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setFormError(null);
    setSuccess(null);
    try {
      const input = formToInput();
      if (!input.code || Number.isNaN(input.value) || input.value <= 0) {
        setFormError("Coupon code and positive value are required.");
        return;
      }
      const response = selectedCoupon
        ? await updateCoupon(selectedCoupon.id, input)
        : await createCoupon(input);
      const saved = response.data;
      setSuccess(selectedCoupon ? "Coupon updated." : "Coupon created.");
      if (saved) {
        selectCoupon(saved);
      }
      await loadCoupons(page, appliedFilters);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to save coupon");
    } finally {
      setSaving(false);
    }
  }

  async function removeCoupon(coupon: Coupon) {
    if (!window.confirm(`Delete coupon ${coupon.code}? Existing orders keep their discount snapshot.`)) {
      return;
    }
    setDeletingId(coupon.id);
    setError(null);
    setSuccess(null);
    try {
      await deleteCoupon(coupon.id);
      if (selectedCoupon?.id === coupon.id) {
        resetForm();
      }
      setSuccess("Coupon deleted.");
      await loadCoupons(page, appliedFilters);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete coupon");
    } finally {
      setDeletingId(null);
    }
  }

  return (
    <div className="grid min-w-0 gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-6">
        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <p className="eyebrow">Pricing operations</p>
                <CardTitle className="mt-2 text-3xl">Admin Coupons</CardTitle>
              </div>
              <Badge variant="secondary">{meta?.total ?? coupons.length} coupons</Badge>
            </div>
          </CardHeader>
          <CardContent>
            <form className="grid gap-3 lg:grid-cols-[1.4fr_150px_150px_150px]" onSubmit={applyFilters}>
              <div className="space-y-1.5">
                <label htmlFor="coupon-search" className="text-sm font-medium">
                  Search code
                </label>
                <Input
                  id="coupon-search"
                  value={filters.query}
                  onChange={(event) => updateFilter("query", event.target.value)}
                  placeholder="SAVE20"
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="coupon-type-filter" className="text-sm font-medium">
                  Type
                </label>
                <select
                  id="coupon-type-filter"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.type}
                  onChange={(event) => updateFilter("type", event.target.value as FilterState["type"])}
                >
                  <option value="">All</option>
                  <option value="percent">Percent</option>
                  <option value="fixed">Fixed</option>
                </select>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="coupon-active-filter" className="text-sm font-medium">
                  Active
                </label>
                <select
                  id="coupon-active-filter"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.isActive}
                  onChange={(event) => updateFilter("isActive", event.target.value as FilterState["isActive"])}
                >
                  <option value="">All</option>
                  <option value="true">Active</option>
                  <option value="false">Inactive</option>
                </select>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="coupon-sort-filter" className="text-sm font-medium">
                  Sort
                </label>
                <select
                  id="coupon-sort-filter"
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
              <p className="eyebrow">Discount rules</p>
              <CardTitle className="text-3xl">Coupon List</CardTitle>
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
                <p className="text-xl font-semibold">Could not load coupons.</p>
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
            {!loading && !error && coupons.length === 0 ? (
              <div className="state-panel">
                <p className="text-xl font-semibold">No coupons found.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Create a coupon to let customers preview and apply discounts at checkout.
                </p>
              </div>
            ) : null}
            {!loading && !error && coupons.length > 0 ? (
              <div className="overflow-x-auto rounded-2xl border border-border">
                <div className="hidden grid-cols-[1.1fr_1fr_110px_130px] gap-4 bg-muted/60 px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground lg:grid">
                  <span>Code</span>
                  <span>Discount</span>
                  <span>Status</span>
                  <span className="text-right">Actions</span>
                </div>
                <div className="divide-y divide-border">
                  {coupons.map((coupon) => (
                    <div
                      key={coupon.id}
                      className="grid min-w-0 gap-4 p-4 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)_110px_130px] lg:items-center"
                    >
                      <button type="button" className="min-w-0 text-left" onClick={() => selectCoupon(coupon)}>
                        <p className="font-heading text-xl font-semibold">{coupon.code}</p>
                        <p className="mt-1 text-xs text-muted-foreground">
                          Used {coupon.used_count}
                          {coupon.usage_limit ? ` / ${coupon.usage_limit}` : ""} times
                        </p>
                      </button>
                      <div>
                        <p className="text-sm font-medium">{couponLabel(coupon)}</p>
                        <p className="mt-1 text-xs text-muted-foreground">
                          Min order {formatVND(coupon.min_order_amount)}
                        </p>
                      </div>
                      <Badge variant={coupon.is_active ? "secondary" : "outline"} className="w-fit">
                        {coupon.is_active ? "active" : "inactive"}
                      </Badge>
                      <div className="flex flex-wrap justify-start gap-2 lg:justify-end">
                        <Button type="button" size="sm" variant="outline" onClick={() => selectCoupon(coupon)}>
                          Edit
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="destructive"
                          disabled={deletingId === coupon.id}
                          onClick={() => removeCoupon(coupon)}
                        >
                          Delete
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
            {meta && totalPages > 1 ? (
              <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
                <Button type="button" variant="outline" disabled={page <= 1 || loading} onClick={() => goToPage(page - 1)}>
                  Previous
                </Button>
                <p className="text-sm text-muted-foreground">
                  Showing {coupons.length} of {meta.total}
                </p>
                <Button type="button" variant="outline" disabled={page >= totalPages || loading} onClick={() => goToPage(page + 1)}>
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
            <p className="eyebrow">{selectedCoupon ? "Edit coupon" : "Create coupon"}</p>
            <CardTitle className="text-3xl">{selectedCoupon ? selectedCoupon.code : "New Coupon"}</CardTitle>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={submitCoupon}>
              <div className="space-y-1.5">
                <label htmlFor="coupon-code" className="text-sm font-medium">
                  Code
                </label>
                <Input
                  id="coupon-code"
                  value={form.code}
                  onChange={(event) => updateForm("code", event.target.value.toUpperCase())}
                  placeholder="SAVE20"
                  required
                />
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <label htmlFor="coupon-type" className="text-sm font-medium">
                    Type
                  </label>
                  <select
                    id="coupon-type"
                    className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                    value={form.type}
                    onChange={(event) => updateForm("type", event.target.value as CouponType)}
                  >
                    <option value="percent">Percent</option>
                    <option value="fixed">Fixed</option>
                  </select>
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="coupon-value" className="text-sm font-medium">
                    Value
                  </label>
                  <Input
                    id="coupon-value"
                    type="number"
                    min="0"
                    step="0.01"
                    value={form.value}
                    onChange={(event) => updateForm("value", event.target.value)}
                    required
                  />
                </div>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <label htmlFor="min-order" className="text-sm font-medium">
                    Min order amount
                  </label>
                  <Input
                    id="min-order"
                    type="number"
                    min="0"
                    step="1000"
                    value={form.minOrderAmount}
                    onChange={(event) => updateForm("minOrderAmount", event.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="max-discount" className="text-sm font-medium">
                    Max discount
                  </label>
                  <Input
                    id="max-discount"
                    type="number"
                    min="0"
                    step="1000"
                    value={form.maxDiscountAmount}
                    onChange={(event) => updateForm("maxDiscountAmount", event.target.value)}
                    placeholder="Optional"
                  />
                </div>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="usage-limit" className="text-sm font-medium">
                  Usage limit
                </label>
                <Input
                  id="usage-limit"
                  type="number"
                  min="1"
                  value={form.usageLimit}
                  onChange={(event) => updateForm("usageLimit", event.target.value)}
                  placeholder="Optional"
                />
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <label htmlFor="starts-at" className="text-sm font-medium">
                    Starts at
                  </label>
                  <Input
                    id="starts-at"
                    type="datetime-local"
                    value={form.startsAt}
                    onChange={(event) => updateForm("startsAt", event.target.value)}
                  />
                </div>
                <div className="space-y-1.5">
                  <label htmlFor="expires-at" className="text-sm font-medium">
                    Expires at
                  </label>
                  <Input
                    id="expires-at"
                    type="datetime-local"
                    value={form.expiresAt}
                    onChange={(event) => updateForm("expiresAt", event.target.value)}
                  />
                </div>
              </div>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.isActive}
                  onChange={(event) => updateForm("isActive", event.target.checked)}
                />
                Active coupon
              </label>
              {formError ? (
                <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {formError}
                </p>
              ) : null}
              {success ? <p className="text-sm text-primary">{success}</p> : null}
              <div className="flex flex-wrap gap-2">
                <Button type="submit" disabled={saving}>
                  {saving ? "Saving..." : selectedCoupon ? "Update coupon" : "Create coupon"}
                </Button>
                <Button type="button" variant="outline" onClick={resetForm}>
                  New coupon
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
