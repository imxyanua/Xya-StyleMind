"use client";

import { useEffect, useMemo, useState } from "react";

import { ProductCard } from "@/components/product/product-card";
import { Button } from "@/components/ui/button";
import { fetchProducts } from "@/lib/api";
import type { PaginationMeta } from "@/types/api";
import type { Product } from "@/types/product";

const styleOptions = ["streetwear", "minimal", "korean", "formal", "casual", "sporty"];
const colorOptions = ["black", "white", "beige", "blue", "gray", "brown"];

export default function ProductsPage() {
  const [style, setStyle] = useState("");
  const [color, setColor] = useState("");
  const [page, setPage] = useState(1);
  const [products, setProducts] = useState<Product[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadProducts() {
      setLoading(true);
      setError(null);
      try {
        const response = await fetchProducts({
          style: style || undefined,
          color: color || undefined,
          page,
          limit: 20,
        });
        if (!cancelled) {
          setProducts(response.data ?? []);
          setMeta(response.meta ?? null);
        }
      } catch (err) {
        if (!cancelled) {
          const message = err instanceof Error ? err.message : "Failed to fetch products";
          setError(message);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadProducts();
    return () => {
      cancelled = true;
    };
  }, [style, color, page]);

  const totalPages = useMemo(() => meta?.total_page ?? 1, [meta]);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end gap-4">
        <div className="space-y-1">
          <label htmlFor="style" className="text-sm font-medium">
            Style
          </label>
          <select
            id="style"
            className="h-8 rounded-lg border border-input bg-background px-3 text-sm"
            value={style}
            onChange={(event) => {
              setPage(1);
              setStyle(event.target.value);
            }}
          >
            <option value="">All styles</option>
            {styleOptions.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </div>

        <div className="space-y-1">
          <label htmlFor="color" className="text-sm font-medium">
            Color
          </label>
          <select
            id="color"
            className="h-8 rounded-lg border border-input bg-background px-3 text-sm"
            value={color}
            onChange={(event) => {
              setPage(1);
              setColor(event.target.value);
            }}
          >
            <option value="">All colors</option>
            {colorOptions.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </div>
      </div>

      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">Loading products...</p> : null}

      {!loading && !error ? (
        <>
          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {products.map((product) => (
              <ProductCard key={product.id} product={product} />
            ))}
          </div>

          {products.length === 0 ? (
            <p className="text-sm text-muted-foreground">No products found.</p>
          ) : null}

          <div className="flex items-center justify-between pt-4">
            <p className="text-sm text-muted-foreground">
              Page {meta?.page ?? 1} / {totalPages}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => setPage((prev) => Math.max(1, prev - 1))}
                disabled={(meta?.page ?? 1) <= 1}
              >
                Previous
              </Button>
              <Button
                variant="outline"
                onClick={() => setPage((prev) => prev + 1)}
                disabled={(meta?.page ?? 1) >= totalPages}
              >
                Next
              </Button>
            </div>
          </div>
        </>
      ) : null}
    </div>
  );
}
