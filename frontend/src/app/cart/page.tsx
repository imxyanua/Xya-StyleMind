"use client";

import { useEffect, useMemo, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { MapPin, ShieldCheck, Truck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { checkoutOrder, fetchCart, removeCartItem, updateCartItem } from "@/lib/api";
import { getToken } from "@/lib/auth";
import { PRODUCT_IMAGE_BLUR } from "@/lib/images";
import { ApiError } from "@/types/api";
import type { Cart } from "@/types/cart";

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

export default function CartPage() {
  const router = useRouter();
  const [cart, setCart] = useState<Cart | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [checkoutLoading, setCheckoutLoading] = useState(false);
  const [itemBusyId, setItemBusyId] = useState<string | null>(null);
  const [quantityInputs, setQuantityInputs] = useState<Record<string, number>>({});

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login?redirect=/cart");
      return;
    }

    let cancelled = false;
    async function loadCart() {
      setLoading(true);
      setError(null);
      try {
        const res = await fetchCart();
        if (!cancelled) {
          const nextCart = res.data ?? null;
          setCart(nextCart);
          const seed: Record<string, number> = {};
          nextCart?.items.forEach((item) => {
            seed[item.id] = item.quantity;
          });
          setQuantityInputs(seed);
        }
      } catch (err) {
        if (!cancelled) {
          if (err instanceof ApiError && err.status === 401) {
            router.replace("/login?redirect=/cart");
            return;
          }
          setError(err instanceof Error ? err.message : "Failed to fetch cart");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadCart();
    return () => {
      cancelled = true;
    };
  }, [router]);

  const isEmpty = useMemo(() => !cart || cart.items.length === 0, [cart]);

  async function onUpdateItem(itemId: string) {
    const quantity = quantityInputs[itemId];
    if (!quantity || quantity < 1) {
      setError("Quantity must be greater than 0");
      return;
    }

    setItemBusyId(itemId);
    setError(null);
    try {
      const res = await updateCartItem(itemId, quantity);
      setCart(res.data ?? null);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/cart");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to update cart item");
    } finally {
      setItemBusyId(null);
    }
  }

  async function onRemoveItem(itemId: string) {
    setItemBusyId(itemId);
    setError(null);
    try {
      const res = await removeCartItem(itemId);
      setCart(res.data ?? null);
      setQuantityInputs((prev) => {
        const copy = { ...prev };
        delete copy[itemId];
        return copy;
      });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/cart");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to remove cart item");
    } finally {
      setItemBusyId(null);
    }
  }

  async function onCheckout() {
    setCheckoutLoading(true);
    setError(null);
    try {
      await checkoutOrder();
      router.push("/orders?checkout=success");
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/cart");
        return;
      }
      setError(err instanceof Error ? err.message : "Checkout failed");
    } finally {
      setCheckoutLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="grid gap-4">
        {Array.from({ length: 3 }).map((_, index) => (
          <div key={index} className="h-36 animate-pulse rounded-[1.5rem] bg-muted/80" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-7">
      <div className="surface-card flex flex-col gap-4 rounded-[2rem] p-6 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="eyebrow">Checkout path</p>
          <h1 className="mt-2 text-4xl font-semibold">Your Cart</h1>
          <p className="mt-2 max-w-xl text-sm text-muted-foreground">
            Review quantities and confirm your order from the live backend cart.
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href="/products">Continue shopping</Link>
        </Button>
      </div>

      {error ? (
        <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      {isEmpty ? (
        <Card className="surface-card">
          <CardContent className="state-panel">
            <p className="text-xl font-semibold">Your cart is empty.</p>
            <p className="max-w-md text-sm text-muted-foreground">
              Add a piece from product detail to begin the checkout flow.
            </p>
            <Button asChild>
              <Link href="/products">Browse products</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid min-w-0 gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
          <div className="space-y-4">
            {cart?.items.map((item) => (
              <Card key={item.id} className="surface-card overflow-hidden rounded-[1.5rem]">
                <CardContent className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex min-w-0 items-center gap-4">
                    <div className="relative h-28 w-24 shrink-0 overflow-hidden rounded-2xl bg-muted">
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
                    <div className="min-w-0 space-y-1">
                      <p className="break-words font-heading text-xl font-semibold">{item.product.name}</p>
                      <p className="text-sm text-muted-foreground">
                        {item.product.style} / {item.product.color}
                      </p>
                      <p className="text-sm">{formatVND(item.product.price)}</p>
                      <p className="text-sm font-semibold">Subtotal: {formatVND(item.subtotal)}</p>
                    </div>
                  </div>

                  <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">
                    <input
                      type="number"
                      min={1}
                      className="h-10 w-20 rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                      value={quantityInputs[item.id] ?? item.quantity}
                      onChange={(event) =>
                        setQuantityInputs((prev) => ({
                          ...prev,
                          [item.id]: Number(event.target.value),
                        }))
                      }
                    />
                    <Button
                      variant="outline"
                      onClick={() => onUpdateItem(item.id)}
                      disabled={itemBusyId === item.id}
                    >
                      Update
                    </Button>
                    <Button
                      variant="destructive"
                      onClick={() => onRemoveItem(item.id)}
                      disabled={itemBusyId === item.id}
                    >
                      Remove
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          <Card className="surface-card h-fit rounded-[1.5rem] lg:sticky lg:top-28">
            <CardHeader>
              <CardTitle className="text-2xl">Checkout summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-3 rounded-2xl border border-border bg-muted/45 p-4 text-sm">
                <div className="flex items-center gap-2 font-medium">
                  <MapPin className="size-4" />
                  Delivery contact
                </div>
                <p className="leading-6 text-muted-foreground">
                  MVP checkout uses your account email. Address collection and carrier selection are
                  ready for the next backend phase.
                </p>
              </div>
              <div className="grid gap-2 rounded-2xl border border-border bg-card/70 p-4 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Items</span>
                  <span>{cart?.items.length ?? 0}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Shipping</span>
                  <span>Calculated later</span>
                </div>
                <div className="flex items-center justify-between border-t border-border pt-3">
                  <span className="font-medium">Total</span>
                  <span className="font-heading text-2xl font-semibold">{formatVND(cart?.total ?? 0)}</span>
                </div>
              </div>
              <div className="grid gap-2 text-xs text-muted-foreground">
                <p className="inline-flex items-center gap-2">
                  <ShieldCheck className="size-4" /> Stock is validated again during checkout.
                </p>
                <p className="inline-flex items-center gap-2">
                  <Truck className="size-4" /> Order status appears in your order history after success.
                </p>
              </div>
              <div className="flex items-center justify-between rounded-2xl bg-primary p-4 text-primary-foreground">
                <span className="text-sm opacity-80">Payable now</span>
                <span className="font-heading text-2xl font-semibold">
                  {formatVND(cart?.total ?? 0)}
                </span>
              </div>
              <Button className="w-full" onClick={onCheckout} disabled={checkoutLoading || isEmpty}>
                {checkoutLoading ? "Processing..." : "Checkout"}
              </Button>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
