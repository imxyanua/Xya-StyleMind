"use client";

import { useEffect, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Badge } from "@/components/ui/badge";
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
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, index) => (
          <div key={index} className="h-80 animate-pulse rounded-[1.5rem] bg-muted/80" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-7">
      <div className="surface-card flex flex-col gap-4 rounded-[2rem] p-6 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="eyebrow">Saved edit</p>
          <h1 className="mt-2 text-4xl font-semibold">Wishlist</h1>
          <p className="mt-2 max-w-xl text-sm text-muted-foreground">
            Save products you want to compare, style, or buy later.
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href="/products">Explore products</Link>
        </Button>
      </div>

      {error ? (
        <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      {items.length === 0 ? (
        <Card className="surface-card">
          <CardContent className="state-panel">
            <p className="text-xl font-semibold">Your wishlist is empty.</p>
            <p className="max-w-md text-sm text-muted-foreground">
              Tap the heart on product cards or product detail pages to save favorites.
            </p>
            <Button asChild>
              <Link href="/products">Find products</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {items.map((item) => {
            const product = item.product;
            const productId = item.product_id ?? product?.id;
            return (
              <Card key={item.id ?? productId} className="surface-card overflow-hidden rounded-[1.5rem]">
                <div className="relative h-72 bg-muted">
                  {product?.image_url ? (
                    <Image
                      src={product.image_url}
                      alt={product.name ?? "Wishlist product"}
                      fill
                      className="object-cover"
                      sizes="(max-width: 1024px) 100vw, 33vw"
                    />
                  ) : null}
                  <div className="absolute left-3 top-3">
                    <Badge variant={product?.stock && product.stock > 0 ? "secondary" : "destructive"}>
                      {product?.stock && product.stock > 0 ? "In stock" : "Sold out"}
                    </Badge>
                  </div>
                </div>
                <CardContent className="space-y-4 p-5">
                  <div>
                    <Link
                      href={productId ? `/products/${productId}` : "/products"}
                      className="font-heading text-2xl font-semibold hover:underline"
                    >
                      {product?.name ?? "Unavailable product"}
                    </Link>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {product?.style ?? "style"} / {product?.color ?? "color"}
                    </p>
                  </div>
                  <p className="font-heading text-2xl font-semibold">{formatVND(product?.price)}</p>
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
