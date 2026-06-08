"use client";

import type {
  DashboardOrdersByStatus,
  DashboardRevenueByDay,
  DashboardTopProduct,
} from "@/types/dashboard";

const statusConfig = [
  { key: "pending", label: "Pending", color: "#9a8f7b" },
  { key: "paid", label: "Paid", color: "#587044" },
  { key: "shipping", label: "Shipping", color: "#9c6f43" },
  { key: "completed", label: "Completed", color: "#263323" },
  { key: "cancelled", label: "Cancelled", color: "#a53b3b" },
] as const;

function formatVND(value?: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
    maximumFractionDigits: 0,
  }).format(value ?? 0);
}

function formatCompact(value?: number) {
  return new Intl.NumberFormat("vi-VN", {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value ?? 0);
}

function normalizeRevenueDays(days: DashboardRevenueByDay[]) {
  const items = days.slice(-10);
  const max = Math.max(...items.map((item) => item.revenue ?? 0), 1);
  return items.map((item, index) => {
    const x = items.length === 1 ? 50 : (index / (items.length - 1)) * 100;
    const y = 100 - ((item.revenue ?? 0) / max) * 78 - 10;
    return { ...item, x, y };
  });
}

export function RevenueLineChart({ data }: { data: DashboardRevenueByDay[] }) {
  if (data.length === 0) {
    return (
      <p className="rounded-3xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
        No revenue yet.
      </p>
    );
  }

  const points = normalizeRevenueDays(data);
  const path = points.map((point, index) => `${index === 0 ? "M" : "L"} ${point.x} ${point.y}`).join(" ");
  const area = `${path} L ${points.at(-1)?.x ?? 100} 100 L ${points[0]?.x ?? 0} 100 Z`;
  const total = data.reduce((sum, item) => sum + (item.revenue ?? 0), 0);

  return (
    <div className="space-y-4">
      <div className="rounded-[1.5rem] border border-border bg-[linear-gradient(180deg,hsl(var(--muted)/0.45),transparent)] p-4">
        <svg viewBox="0 0 100 100" className="h-56 w-full overflow-visible" role="img" aria-label="Revenue by day line chart">
          <defs>
            <linearGradient id="revenue-area" x1="0" x2="0" y1="0" y2="1">
              <stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity="0.28" />
              <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity="0.02" />
            </linearGradient>
          </defs>
          {[20, 40, 60, 80].map((y) => (
            <line key={y} x1="0" x2="100" y1={y} y2={y} stroke="hsl(var(--border))" strokeWidth="0.4" />
          ))}
          <path d={area} fill="url(#revenue-area)" />
          <path d={path} fill="none" stroke="hsl(var(--primary))" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.4" />
          {points.map((point) => (
            <circle key={point.date} cx={point.x} cy={point.y} r="2.2" fill="hsl(var(--card))" stroke="hsl(var(--primary))" strokeWidth="1.5">
              <title>{`${point.date}: ${formatVND(point.revenue)}`}</title>
            </circle>
          ))}
        </svg>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
        <span className="text-muted-foreground">Last {points.length} reporting days</span>
        <span className="font-semibold">{formatVND(total)} total</span>
      </div>
    </div>
  );
}

export function OrdersDonutChart({ data }: { data: DashboardOrdersByStatus }) {
  const items = statusConfig.map((item) => ({ ...item, value: data[item.key] ?? 0 }));
  const total = items.reduce((sum, item) => sum + item.value, 0);

  if (total === 0) {
    return (
      <p className="rounded-3xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
        No orders yet.
      </p>
    );
  }

  const radius = 15.9155;
  const circumference = 100;
  const dashes = items.map((item) => (item.value / total) * circumference);
  const segments = items.map((item, index) => ({
    ...item,
    dash: dashes[index],
    offset: 25 - dashes.slice(0, index).reduce((sum, dash) => sum + dash, 0),
  }));

  return (
    <div className="grid gap-5 sm:grid-cols-[190px_1fr] sm:items-center">
      <div className="relative mx-auto size-48">
        <svg viewBox="0 0 42 42" className="size-full -rotate-90" role="img" aria-label="Orders by status donut chart">
          <circle cx="21" cy="21" r={radius} fill="transparent" stroke="hsl(var(--muted))" strokeWidth="7" />
          {segments.map((item) => (
              <circle
                key={item.key}
                cx="21"
                cy="21"
                r={radius}
                fill="transparent"
                stroke={item.color}
                strokeDasharray={`${item.dash} ${circumference - item.dash}`}
                strokeDashoffset={item.offset}
                strokeLinecap="round"
                strokeWidth="7"
              >
                <title>{`${item.label}: ${item.value}`}</title>
              </circle>
          ))}
        </svg>
        <div className="absolute inset-0 grid place-items-center text-center">
          <div>
            <p className="font-heading text-4xl font-semibold">{total}</p>
            <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">orders</p>
          </div>
        </div>
      </div>
      <div className="grid gap-2">
        {items.map((item) => (
          <div key={item.key} className="flex items-center justify-between gap-3 rounded-2xl bg-muted/55 px-3 py-2 text-sm">
            <span className="flex items-center gap-2">
              <span className="size-2.5 rounded-full" style={{ backgroundColor: item.color }} />
              {item.label}
            </span>
            <span className="font-semibold">{item.value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

export function TopProductsBarChart({ data }: { data: DashboardTopProduct[] }) {
  if (data.length === 0) {
    return (
      <p className="rounded-3xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
        No sold products yet.
      </p>
    );
  }

  const items = data.slice(0, 6);
  const maxRevenue = Math.max(...items.map((item) => item.revenue ?? 0), 1);

  return (
    <div className="space-y-3">
      {items.map((product, index) => {
        const width = `${Math.max(((product.revenue ?? 0) / maxRevenue) * 100, 8)}%`;
        return (
          <div key={product.id} className="space-y-2 rounded-2xl bg-muted/55 p-3">
            <div className="flex items-start justify-between gap-3 text-sm">
              <div className="min-w-0">
                <p className="truncate font-medium">{index + 1}. {product.name}</p>
                <p className="text-xs text-muted-foreground">{formatCompact(product.quantity_sold)} sold</p>
              </div>
              <span className="shrink-0 font-semibold">{formatVND(product.revenue)}</span>
            </div>
            <div className="h-3 overflow-hidden rounded-full bg-card">
              <div className="h-full rounded-full bg-primary" style={{ width }} />
            </div>
          </div>
        );
      })}
    </div>
  );
}
