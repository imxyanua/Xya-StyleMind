"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { useParams, useRouter } from "next/navigation";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { addToCart, fetchProductById } from "@/lib/api";
import { getToken } from "@/lib/auth";
import type { Product } from "@/types/product";

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

export default function ProductDetailPage() {
  const router = useRouter();
  const params = useParams();
  const productId = Array.isArray(params.id) ? params.id[0] : params.id;
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const id = productId;
    if (!id) {
      return;
    }

    async function loadProduct() {
      setLoading(true);
      setError(null);
      try {
        const response = await fetchProductById(id);
        if (!cancelled) {
          setProduct(response.data ?? null);
        }
      } catch (err) {
        if (!cancelled) {
          const message = err instanceof Error ? err.message : "Failed to fetch product";
          setError(message);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadProduct();
    return () => {
      cancelled = true;
    };
  }, [productId]);

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading product...</p>;
  }

  if (error) {
    return <p className="text-sm text-red-600">{error}</p>;
  }

  if (!product) {
    return <p className="text-sm text-muted-foreground">Product not found.</p>;
  }

  async function onAddToCart() {
    if (!getToken()) {
      router.push(`/login?redirect=/products/${product.id}`);
      return;
    }

    setAdding(true);
    setError(null);
    setSuccess(null);
    try {
      await addToCart(product.id, 1);
      setSuccess("Added to cart");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to add to cart";
      setError(message);
    } finally {
      setAdding(false);
    }
  }

  return (
    <Card className="overflow-hidden">
      <div className="relative h-80 w-full sm:h-[28rem]">
        <Image
          src={product.image_url}
          alt={product.name}
          fill
          className="object-cover"
          sizes="100vw"
        />
      </div>
      <CardHeader>
        <CardTitle>{product.name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="text-xl font-semibold">{formatVND(product.price)}</div>
        <div className="flex flex-wrap gap-2">
          <Badge variant="secondary">{product.style}</Badge>
          <Badge variant="outline">{product.color}</Badge>
        </div>
        <p className="text-sm text-muted-foreground">Stock: {product.stock}</p>
        <p className="leading-relaxed">{product.description}</p>
        {error ? <p className="text-sm text-red-600">{error}</p> : null}
        {success ? <p className="text-sm text-green-700">{success}</p> : null}
        <div className="flex flex-wrap gap-2">
          <Button onClick={onAddToCart} disabled={adding || product.stock <= 0}>
            {adding ? "Adding..." : "Add to Cart"}
          </Button>
          <Button variant="outline" asChild>
            <Link href="/cart">Go to Cart</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href="/products">Back to products</Link>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
