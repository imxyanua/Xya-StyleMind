"use client";

import { useEffect, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { addToCart, fetchWishlist, removeWishlistProduct } from "@/lib/api";
import { getToken } from "@/lib/auth";
import { ApiError } from "@/types/api";
import type { WishlistItem } from "@/types/wishlist";

function formatVND(value?: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value ?? 0);
}

export default function WishlistPage() {
  const router = useRouter();
  const [items, setItems] = useState<WishlistItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyProductId, setBusyProductId] = useState<string | null>(null);

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login?redirect=/wishlist");
      return;
    }

    let cancelled = false;
    async function loadWishlist() {
      setLoading(true);
      setError(null);
      try {
        const response = await fetchWishlist({ page: 1, limit: 100 });
        if (!cancelled) {
          setItems(response.data ?? []);
        }
      } catch (err) {
        if (!cancelled) {
          if (err instanceof ApiError && err.status === 401) {
            router.replace("/login?redirect=/wishlist");
            return;
          }
          setError(err instanceof Error ? err.message : "Failed to fetch wishlist");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadWishlist();
    return () => {
      cancelled = true;
    };
  }, [router]);

  async function onRemove(productId?: string) {
    if (!productId) {
      return;
    }
    const previous = items;
    setBusyProductId(productId);
    setError(null);
    setItems((current) => current.filter((item) => item.product_id !== productId));
    try {
      await removeWishlistProduct(productId);
    } catch (err) {
      setItems(previous);
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/wishlist");
        return;
      }
      setError(err instanceof Error ? err.message : "Could not remove wishlist product.");
    } finally {
      setBusyProductId(null);
    }
  }

  async function onAddToCart(productId?: string) {
    if (!productId) {
      return;
    }
    setBusyProductId(productId);
    setError(null);
    try {
      await addToCart(productId, 1);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/wishlist");
        return;
      }
      setError(err instanceof Error ? err.message : "Could not add product to cart.");
    } finally {
      setBusyProductId(null);
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading wishlist...</p>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Wishlist</h1>
          <p className="text-sm text-muted-foreground">
            Save products you want to compare, style, or buy later.
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href="/products">Explore products</Link>
        </Button>
      </div>

      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      {items.length === 0 ? (
        <Card>
          <CardContent className="flex min-h-64 flex-col items-center justify-center gap-3 p-8 text-center">
            <p className="text-lg font-semibold">Your wishlist is empty.</p>
            <p className="max-w-md text-sm text-muted-foreground">
              Tap the heart on product cards or product detail pages to save favorites.
            </p>
            <Button asChild>
              <Link href="/products">Find products</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {items.map((item) => {
            const product = item.product;
            const productId = item.product_id ?? product?.id;
            return (
              <Card key={item.id ?? productId}>
                <CardContent className="flex flex-col gap-4 py-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex items-center gap-4">
                    <div className="relative h-28 w-24 overflow-hidden rounded-xl bg-muted">
                      {product?.image_url ? (
                        <Image
                          src={product.image_url}
                          alt={product.name ?? "Wishlist product"}
                          fill
                          className="object-cover"
                          sizes="120px"
                        />
                      ) : null}
                    </div>
                    <div className="space-y-1">
                      <Link
                        href={productId ? `/products/${productId}` : "/products"}
                        className="font-medium hover:underline"
                      >
                        {product?.name ?? "Unavailable product"}
                      </Link>
                      <p className="text-sm text-muted-foreground">
                        {product?.style ?? "style"} · {product?.color ?? "color"} ·{" "}
                        {product?.stock && product.stock > 0 ? `${product.stock} in stock` : "Sold out"}
                      </p>
                      <p className="text-sm font-medium">{formatVND(product?.price)}</p>
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-2">
                    <Button
                      onClick={() => onAddToCart(productId)}
                      disabled={!productId || busyProductId === productId || !product?.stock}
                    >
                      Add to Cart
                    </Button>
                    <Button
                      variant="outline"
                      onClick={() => onRemove(productId)}
                      disabled={!productId || busyProductId === productId}
                    >
                      Remove
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
