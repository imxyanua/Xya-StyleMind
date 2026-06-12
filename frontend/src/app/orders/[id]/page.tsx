"use client";

import { useEffect, useMemo, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, CheckCircle2, Clock3, CreditCard, MapPin, PackageCheck, Truck } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchMyOrder } from "@/lib/api";
import { getToken } from "@/lib/auth";
import { PRODUCT_IMAGE_BLUR } from "@/lib/images";
import { ApiError } from "@/types/api";
import type { Order } from "@/types/order";

const timelineSteps = ["pending", "paid", "shipping", "completed"] as const;

type TimelineStatus = (typeof timelineSteps)[number];

const statusTone: Record<Order["status"], "secondary" | "outline" | "destructive"> = {
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

function statusLabel(status: string) {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function Timeline({ status }: { status: Order["status"] }) {
  if (status === "cancelled") {
    return (
      <div className="rounded-3xl border border-destructive/30 bg-destructive/10 p-5 text-sm text-destructive">
        <p className="font-semibold">This order was cancelled.</p>
        <p className="mt-1 text-destructive/80">Fulfillment is stopped and no further timeline steps apply.</p>
      </div>
    );
  }

  const currentIndex = timelineSteps.indexOf(status as TimelineStatus);

  return (
    <div className="grid gap-3 sm:grid-cols-4">
      {timelineSteps.map((step, index) => {
        const reached = currentIndex >= index;
        const current = currentIndex === index;
        return (
          <div
            key={step}
            className={`rounded-3xl border p-4 transition ${
              reached ? "border-primary/35 bg-secondary/60" : "border-border bg-card/70"
            }`}
          >
            <div className="flex items-center gap-2">
              <span
                className={`grid size-8 place-items-center rounded-2xl ${
                  reached ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                }`}
              >
                {reached ? <CheckCircle2 className="size-4" /> : <Clock3 className="size-4" />}
              </span>
              <p className="font-medium capitalize">{step}</p>
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {current ? "Current step" : reached ? "Completed" : "Waiting"}
            </p>
          </div>
        );
      })}
    </div>
  );
}

export default function OrderDetailPage() {
  const router = useRouter();
  const params = useParams();
  const orderId = Array.isArray(params.id) ? params.id[0] : params.id;
  const [order, setOrder] = useState<Order | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (!orderId) {
      return;
    }
    if (!getToken()) {
      router.replace(`/login?redirect=/orders/${orderId}`);
      return;
    }

    const idForRequest = orderId;
    let cancelled = false;
    async function loadOrder() {
      setLoading(true);
      setError(null);
      setNotFound(false);
      try {
        const response = await fetchMyOrder(idForRequest);
        if (!cancelled) {
          setOrder(response.data ?? null);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        if (err instanceof ApiError && err.status === 401) {
          router.replace(`/login?redirect=/orders/${orderId}`);
          return;
        }
        if (err instanceof ApiError && (err.status === 403 || err.status === 404)) {
          setNotFound(true);
          return;
        }
        setError(err instanceof Error ? err.message : "Failed to load order detail");
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadOrder();
    return () => {
      cancelled = true;
    };
  }, [orderId, router]);

  const subtotal = useMemo(() => {
    return order?.items.reduce((sum, item) => sum + (item.subtotal ?? 0), 0) ?? 0;
  }, [order]);

  if (loading) {
    return (
      <div className="grid gap-5">
        <div className="h-48 animate-pulse rounded-[2rem] bg-muted/80" />
        <div className="grid gap-4 lg:grid-cols-[1fr_360px]">
          <div className="h-96 animate-pulse rounded-[1.75rem] bg-muted/80" />
          <div className="h-96 animate-pulse rounded-[1.75rem] bg-muted/80" />
        </div>
      </div>
    );
  }

  if (notFound) {
    return (
      <Card className="surface-card rounded-[2rem]">
        <CardContent className="state-panel">
          <p className="text-xl font-semibold">Order not found.</p>
          <p className="max-w-md text-sm text-muted-foreground">
            This order may not exist, or it may belong to another account.
          </p>
          <div className="flex flex-wrap justify-center gap-2">
            <Button asChild>
              <Link href="/orders">Back to orders</Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="/products">Continue shopping</Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error || !order) {
    return (
      <Card className="surface-card rounded-[2rem]">
        <CardContent className="state-panel">
          <p className="text-xl font-semibold">Could not load order detail.</p>
          <p className="max-w-md text-sm text-muted-foreground">{error ?? "No order data returned."}</p>
          <Button onClick={() => window.location.reload()}>Retry</Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-7">
      <section className="surface-card overflow-hidden rounded-[2rem] p-6 sm:p-8">
        <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <Button asChild variant="ghost" className="mb-4 px-0">
              <Link href="/orders">
                <ArrowLeft className="size-4" /> Back to orders
              </Link>
            </Button>
            <p className="eyebrow">Order detail</p>
            <h1 className="mt-2 break-all text-4xl font-semibold sm:text-5xl">Order #{order.id.slice(0, 8)}</h1>
            <p className="mt-3 text-sm text-muted-foreground">
              Created {formatDate(order.created_at)} · Updated {formatDate(order.updated_at)}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge variant={statusTone[order.status]} className="h-8 px-3 text-sm capitalize">
              {statusLabel(order.status)}
            </Badge>
            <Badge variant="outline" className="h-8 px-3 text-sm">
              {paymentLabel(order.payment_method)}
            </Badge>
            <Badge variant={paymentStatusTone(order.payment_status)} className="h-8 px-3 text-sm capitalize">
              Payment {paymentStatusLabel(order.payment_status)}
            </Badge>
          </div>
        </div>
      </section>

      <section className="space-y-4">
        <div>
          <p className="eyebrow">Timeline</p>
          <h2 className="mt-2 text-3xl font-semibold">Order progress</h2>
        </div>
        <Timeline status={order.status} />
      </section>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
        <div className="space-y-5">
          <Card className="surface-card rounded-[1.75rem]">
            <CardHeader>
              <CardTitle className="text-3xl">Items</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {order.items.map((item) => (
                <div
                  key={item.id}
                  className="grid gap-4 rounded-3xl border border-border bg-card/70 p-4 sm:grid-cols-[96px_minmax(0,1fr)_auto] sm:items-center"
                >
                  <div className="relative h-28 w-24 overflow-hidden rounded-2xl bg-muted">
                    <Image
                      src={item.product.image_url}
                      alt={item.product.name}
                      fill
                      className="object-cover"
                      sizes="120px"
                      placeholder="blur"
                      blurDataURL={PRODUCT_IMAGE_BLUR}
                    />
                  </div>
                  <div className="min-w-0">
                    <Link href={`/products/${item.product_id}`} className="font-heading text-xl font-semibold hover:underline">
                      {item.product.name}
                    </Link>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {item.product.style} / {item.product.color}
                    </p>
                    <p className="mt-2 text-sm text-muted-foreground">
                      {item.quantity} x {formatVND(item.unit_price)}
                    </p>
                  </div>
                  <p className="font-heading text-2xl font-semibold">{formatVND(item.subtotal)}</p>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card className="surface-card rounded-[1.75rem]">
            <CardHeader>
              <CardTitle className="text-3xl">Shipping contact</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-2">
              <div className="rounded-3xl bg-muted/55 p-4">
                <div className="flex items-center gap-2 font-medium">
                  <MapPin className="size-4" /> Ship to
                </div>
                <p className="mt-3 font-semibold">{order.recipient_name || "Recipient not provided"}</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {[order.address_line, order.district, order.city].filter(Boolean).join(", ") || "Address not provided"}
                </p>
                {order.phone ? <p className="mt-1 text-sm text-muted-foreground">{order.phone}</p> : null}
              </div>
              <div className="rounded-3xl bg-muted/55 p-4">
                <div className="flex items-center gap-2 font-medium">
                  <Truck className="size-4" /> Delivery method
                </div>
                <p className="mt-3 font-semibold">{shippingLabel(order.shipping_method)}</p>
                {order.note ? <p className="mt-1 text-sm text-muted-foreground">Note: {order.note}</p> : null}
              </div>
            </CardContent>
          </Card>
        </div>

        <aside className="space-y-5 lg:sticky lg:top-28 lg:h-fit">
          <Card className="surface-card rounded-[1.75rem]">
            <CardHeader>
              <CardTitle className="text-3xl">Order summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Subtotal</span>
                  <span>{formatVND(subtotal)}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Shipping</span>
                  <span>{shippingLabel(order.shipping_method)}</span>
                </div>
                <div className="flex items-center justify-between rounded-2xl bg-primary p-4 text-primary-foreground">
                  <span>Total</span>
                  <span className="font-heading text-2xl font-semibold">{formatVND(order.total_amount)}</span>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="surface-card rounded-[1.75rem]">
            <CardHeader>
              <CardTitle className="text-3xl">Payment</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <p className="inline-flex items-center gap-2 font-medium">
                <CreditCard className="size-4" /> {paymentLabel(order.payment_method)}
              </p>
              <p className="capitalize">Payment status: {paymentStatusLabel(order.payment_status)}</p>
              <p className="leading-6 text-muted-foreground">
                No real card or bank credentials are stored for this demo checkout.
              </p>
            </CardContent>
          </Card>

          <div className="grid gap-2">
            <Button asChild>
              <Link href="/products">
                <PackageCheck className="size-4" /> Continue shopping
              </Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="/orders">Back to orders</Link>
            </Button>
          </div>
        </aside>
      </div>
    </div>
  );
}
