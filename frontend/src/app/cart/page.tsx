"use client";

import { useEffect, useMemo, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchCart, checkoutOrder, removeCartItem, updateCartItem } from "@/lib/api";
import { getToken } from "@/lib/auth";
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
          const message = err instanceof Error ? err.message : "Failed to fetch cart";
          setError(message);
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
      const nextCart = res.data ?? null;
      setCart(nextCart);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/cart");
        return;
      }
      const message = err instanceof Error ? err.message : "Failed to update cart item";
      setError(message);
    } finally {
      setItemBusyId(null);
    }
  }

  async function onRemoveItem(itemId: string) {
    setItemBusyId(itemId);
    setError(null);
    try {
      const res = await removeCartItem(itemId);
      const nextCart = res.data ?? null;
      setCart(nextCart);
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
      const message = err instanceof Error ? err.message : "Failed to remove cart item";
      setError(message);
    } finally {
      setItemBusyId(null);
    }
  }

  async function onCheckout() {
    setCheckoutLoading(true);
    setError(null);
    try {
      await checkoutOrder();
      router.push("/orders");
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/cart");
        return;
      }
      const message = err instanceof Error ? err.message : "Checkout failed";
      setError(message);
    } finally {
      setCheckoutLoading(false);
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading cart...</p>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Your Cart</h1>
        <Button variant="outline" asChild>
          <Link href="/products">Continue shopping</Link>
        </Button>
      </div>

      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      {isEmpty ? (
        <Card>
          <CardContent className="py-8">
            <p className="text-sm text-muted-foreground">Your cart is empty.</p>
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="space-y-4">
            {cart?.items.map((item) => (
              <Card key={item.id}>
                <CardContent className="flex flex-col gap-4 py-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex items-center gap-4">
                    <div className="relative h-24 w-20 overflow-hidden rounded-md">
                      <Image
                        src={item.product.image_url}
                        alt={item.product.name}
                        fill
                        className="object-cover"
                        sizes="120px"
                      />
                    </div>
                    <div className="space-y-1">
                      <p className="font-medium">{item.product.name}</p>
                      <p className="text-sm text-muted-foreground">
                        {item.product.style} · {item.product.color}
                      </p>
                      <p className="text-sm">{formatVND(item.product.price)}</p>
                      <p className="text-sm font-medium">Subtotal: {formatVND(item.subtotal)}</p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      min={1}
                      className="h-8 w-20 rounded-md border border-input px-2 text-sm"
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

          <Card>
            <CardHeader>
              <CardTitle>Total: {formatVND(cart?.total ?? 0)}</CardTitle>
            </CardHeader>
            <CardContent>
              <Button onClick={onCheckout} disabled={checkoutLoading || isEmpty}>
                {checkoutLoading ? "Processing..." : "Checkout"}
              </Button>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
