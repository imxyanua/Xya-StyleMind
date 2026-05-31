"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { useParams } from "next/navigation";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchProductById } from "@/lib/api";
import type { Product } from "@/types/product";

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

export default function ProductDetailPage() {
  const params = useParams();
  const productId = Array.isArray(params.id) ? params.id[0] : params.id;
  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
        <Button variant="outline" asChild>
          <Link href="/products">Back to products</Link>
        </Button>
      </CardContent>
    </Card>
  );
}
