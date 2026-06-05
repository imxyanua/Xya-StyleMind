"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import { ProductCard } from "@/components/product/product-card";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  addWishlistProduct,
  fetchCategories,
  fetchProducts,
  fetchWishlist,
  removeWishlistProduct,
  type ProductListParams,
} from "@/lib/api";
import { getToken } from "@/lib/auth";
import { ApiError, type PaginationMeta } from "@/types/api";
import type { Category } from "@/types/category";
import type { Product } from "@/types/product";

const styleOptions = ["streetwear", "minimal", "korean", "formal", "casual", "sporty"];
const colorOptions = ["black", "white", "beige", "blue", "gray", "brown"];
const sortOptions: Array<{ label: string; value: NonNullable<ProductListParams["sort"]> }> = [
  { label: "Newest", value: "newest" },
  { label: "Price: low to high", value: "price_asc" },
  { label: "Price: high to low", value: "price_desc" },
  { label: "Highest rated", value: "rating_desc" },
  { label: "Popular", value: "popular" },
];
const limitOptions = [12, 20, 36, 50];

function getNumberParam(params: URLSearchParams, key: string): number | undefined {
  const value = params.get(key);
  if (!value) {
    return undefined;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function getPositiveIntParam(params: URLSearchParams, key: string, fallback: number): number {
  const value = getNumberParam(params, key);
  if (!value || value < 1) {
    return fallback;
  }
  return Math.floor(value);
}

function getBooleanParam(params: URLSearchParams, key: string): boolean | undefined {
  const value = params.get(key);
  if (value === "true") {
    return true;
  }
  if (value === "false") {
    return false;
  }
  return undefined;
}

function getSortParam(params: URLSearchParams): ProductListParams["sort"] {
  const value = params.get("sort");
  if (sortOptions.some((option) => option.value === value)) {
    return value as ProductListParams["sort"];
  }
  return "newest";
}

function buildProductParams(params: URLSearchParams): ProductListParams {
  return {
    q: params.get("q") || undefined,
    category_id: params.get("category_id") || undefined,
    min_price: getNumberParam(params, "min_price"),
    max_price: getNumberParam(params, "max_price"),
    style: params.get("style") || undefined,
    color: params.get("color") || undefined,
    min_rating: getNumberParam(params, "min_rating"),
    in_stock: getBooleanParam(params, "in_stock"),
    sort: getSortParam(params),
    page: getPositiveIntParam(params, "page", 1),
    limit: getPositiveIntParam(params, "limit", 20),
  };
}

type SearchBoxProps = {
  initialValue: string;
  onSearch: (value: string) => void;
};

function SearchBox({ initialValue, onSearch }: SearchBoxProps) {
  const [value, setValue] = useState(initialValue);

  useEffect(() => {
    if (value.trim() === initialValue) {
      return;
    }

    const timeout = window.setTimeout(() => {
      onSearch(value.trim());
    }, 350);

    return () => window.clearTimeout(timeout);
  }, [initialValue, onSearch, value]);

  return (
    <Input
      id="q"
      value={value}
      onChange={(event) => setValue(event.target.value)}
      placeholder="hoodie, shirt, Korean..."
    />
  );
}

function ProductsBrowser() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const canonicalQuery = searchParams.toString();

  const queryParams = useMemo(
    () => buildProductParams(new URLSearchParams(canonicalQuery)),
    [canonicalQuery]
  );
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [wishlistedProductIds, setWishlistedProductIds] = useState<Set<string>>(new Set());
  const [wishlistBusyId, setWishlistBusyId] = useState<string | null>(null);
  const [wishlistError, setWishlistError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  function buildPathWithParams(updates: Record<string, string | number | boolean | undefined>) {
    const next = new URLSearchParams(canonicalQuery);
    for (const [key, value] of Object.entries(updates)) {
      if (value === undefined || value === "") {
        next.delete(key);
      } else {
        next.set(key, String(value));
      }
    }
    const query = next.toString();
    return query ? `${pathname}?${query}` : pathname;
  }

  function replaceParams(updates: Record<string, string | number | boolean | undefined>) {
    router.replace(buildPathWithParams(updates), { scroll: false });
  }

  function updateFilter(key: string, value: string | number | boolean | undefined) {
    replaceParams({ [key]: value, page: 1 });
  }

  useEffect(() => {
    let cancelled = false;

    async function loadData() {
      setLoading(true);
      setError(null);
      try {
        const token = getToken();
        const [productResponse, categoryResponse] = await Promise.all([
          fetchProducts(queryParams),
          fetchCategories(),
        ]);

        let wishlistIds = new Set<string>();
        if (token) {
          try {
            const wishlistResponse = await fetchWishlist({ limit: 100 });
            wishlistIds = new Set(
              (wishlistResponse.data ?? [])
                .map((item) => item.product_id)
                .filter((id): id is string => Boolean(id))
            );
          } catch (err) {
            if (!(err instanceof ApiError && err.status === 401)) {
              setWishlistError("Could not sync wishlist state.");
            }
          }
        }

        if (!cancelled) {
          setProducts(productResponse.data ?? []);
          setMeta(productResponse.meta ?? null);
          setCategories(categoryResponse.data ?? []);
          setWishlistedProductIds(wishlistIds);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to fetch products");
          setProducts([]);
          setMeta(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadData();
    return () => {
      cancelled = true;
    };
  }, [queryParams]);

  async function toggleWishlist(productId: string) {
    if (!getToken()) {
      router.push(`/login?redirect=/products${canonicalQuery ? `?${canonicalQuery}` : ""}`);
      return;
    }

    const wasWishlisted = wishlistedProductIds.has(productId);
    setWishlistBusyId(productId);
    setWishlistError(null);
    setWishlistedProductIds((prev) => {
      const next = new Set(prev);
      if (wasWishlisted) {
        next.delete(productId);
      } else {
        next.add(productId);
      }
      return next;
    });

    try {
      if (wasWishlisted) {
        await removeWishlistProduct(productId);
      } else {
        await addWishlistProduct(productId);
      }
    } catch (err) {
      setWishlistedProductIds((prev) => {
        const next = new Set(prev);
        if (wasWishlisted) {
          next.add(productId);
        } else {
          next.delete(productId);
        }
        return next;
      });
      if (err instanceof ApiError && err.status === 401) {
        router.push(`/login?redirect=/products${canonicalQuery ? `?${canonicalQuery}` : ""}`);
        return;
      }
      setWishlistError(err instanceof Error ? err.message : "Could not update wishlist.");
    } finally {
      setWishlistBusyId(null);
    }
  }

  const categoryById = useMemo(() => {
    return new Map(categories.map((category) => [category.id, category.name]));
  }, [categories]);

  const totalPages = meta?.total_pages ?? meta?.total_page ?? 1;
  const activeFilterCount = [
    queryParams.q,
    queryParams.category_id,
    queryParams.min_price,
    queryParams.max_price,
    queryParams.style,
    queryParams.color,
    queryParams.min_rating,
    queryParams.in_stock,
  ].filter((value) => value !== undefined && value !== "").length;

  return (
    <div className="space-y-8">
      <section className="relative overflow-hidden rounded-[2.25rem] border border-border bg-card p-6 shadow-soft sm:p-9">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_12%_20%,rgba(255,255,255,0.9),transparent_22rem),radial-gradient(circle_at_88%_0%,rgba(95,115,78,0.18),transparent_22rem)]" />
        <div className="relative max-w-4xl space-y-4">
          <p className="eyebrow">Product discovery</p>
          <h1 className="text-4xl font-semibold leading-none sm:text-6xl">
            Search the catalog by mood, price, rating, and stock.
          </h1>
          <p className="max-w-2xl text-sm leading-6 text-muted-foreground sm:text-base">
            Filters are synced to the URL, so the exact product view can be refreshed or shared.
          </p>
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-[300px_1fr]">
        <aside className="lg:sticky lg:top-6 lg:self-start">
          <Card className="surface-card rounded-[1.75rem]">
            <CardHeader className="space-y-2">
              <CardTitle className="text-2xl">Filters</CardTitle>
              <p className="text-sm text-muted-foreground">
                {activeFilterCount} active filter{activeFilterCount === 1 ? "" : "s"}
              </p>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="space-y-2">
                <label htmlFor="q" className="text-sm font-medium">
                  Search
                </label>
                <SearchBox
                  key={queryParams.q ?? ""}
                  initialValue={queryParams.q ?? ""}
                  onSearch={(value) => {
                    if (value !== (queryParams.q ?? "")) {
                      updateFilter("q", value || undefined);
                    }
                  }}
                />
              </div>

              <div className="space-y-2">
                <label htmlFor="category_id" className="text-sm font-medium">
                  Category
                </label>
                <select
                  id="category_id"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={queryParams.category_id ?? ""}
                  onChange={(event) => updateFilter("category_id", event.target.value || undefined)}
                >
                  <option value="">All categories</option>
                  {categories.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))}
                </select>
              </div>

              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
                <div className="space-y-2">
                  <label htmlFor="style" className="text-sm font-medium">
                    Style
                  </label>
                  <select
                    id="style"
                    className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                    value={queryParams.style ?? ""}
                    onChange={(event) => updateFilter("style", event.target.value || undefined)}
                  >
                    <option value="">All styles</option>
                    {styleOptions.map((option) => (
                      <option key={option} value={option}>
                        {option}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <label htmlFor="color" className="text-sm font-medium">
                    Color
                  </label>
                  <select
                    id="color"
                    className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                    value={queryParams.color ?? ""}
                    onChange={(event) => updateFilter("color", event.target.value || undefined)}
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

              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
                <div className="space-y-2">
                  <label htmlFor="min_price" className="text-sm font-medium">
                    Min price
                  </label>
                  <Input
                    id="min_price"
                    type="number"
                    min="0"
                    value={queryParams.min_price ?? ""}
                    onChange={(event) => updateFilter("min_price", event.target.value || undefined)}
                    placeholder="100000"
                  />
                </div>
                <div className="space-y-2">
                  <label htmlFor="max_price" className="text-sm font-medium">
                    Max price
                  </label>
                  <Input
                    id="max_price"
                    type="number"
                    min="0"
                    value={queryParams.max_price ?? ""}
                    onChange={(event) => updateFilter("max_price", event.target.value || undefined)}
                    placeholder="800000"
                  />
                </div>
              </div>

              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
                <div className="space-y-2">
                  <label htmlFor="min_rating" className="text-sm font-medium">
                    Minimum rating
                  </label>
                  <select
                    id="min_rating"
                    className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                    value={queryParams.min_rating ?? ""}
                    onChange={(event) => updateFilter("min_rating", event.target.value || undefined)}
                  >
                    <option value="">Any rating</option>
                    {[5, 4, 3, 2, 1].map((rating) => (
                      <option key={rating} value={rating}>
                        {rating}+ stars
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <label htmlFor="in_stock" className="text-sm font-medium">
                    Stock
                  </label>
                  <select
                    id="in_stock"
                    className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                    value={
                      queryParams.in_stock === undefined ? "" : String(queryParams.in_stock)
                    }
                    onChange={(event) => updateFilter("in_stock", event.target.value || undefined)}
                  >
                    <option value="">All stock states</option>
                    <option value="true">In stock</option>
                    <option value="false">Sold out</option>
                  </select>
                </div>
              </div>

              <Button
                type="button"
                variant="outline"
                className="w-full"
                onClick={() => {
                  router.replace(pathname, { scroll: false });
                }}
              >
                Clear filters
              </Button>
            </CardContent>
          </Card>
        </aside>

        <main className="space-y-5">
          <div className="surface-card flex flex-col gap-3 rounded-[1.5rem] p-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-base font-semibold">
                {loading ? "Searching catalog..." : `${meta?.total ?? products.length} products found`}
              </p>
              <p className="text-xs text-muted-foreground">
                Page {meta?.page ?? queryParams.page ?? 1} of {totalPages}
              </p>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row">
              <select
                aria-label="Sort products"
                className="h-10 rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                value={queryParams.sort ?? "newest"}
                onChange={(event) => updateFilter("sort", event.target.value)}
              >
                {sortOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
              <select
                aria-label="Products per page"
                className="h-10 rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                value={queryParams.limit ?? 20}
                onChange={(event) => updateFilter("limit", Number(event.target.value))}
              >
                {limitOptions.map((option) => (
                  <option key={option} value={option}>
                    {option} / page
                  </option>
                ))}
              </select>
            </div>
          </div>

          {error ? (
            <Card className="border-destructive/30 bg-destructive/10">
              <CardContent className="p-6">
                <p className="font-medium text-destructive">Could not load products.</p>
                <p className="mt-1 text-sm text-muted-foreground">{error}</p>
              </CardContent>
            </Card>
          ) : null}

          {wishlistError ? (
            <p className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {wishlistError}
            </p>
          ) : null}

          {loading ? (
            <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
              {Array.from({ length: 6 }).map((_, index) => (
                <div key={index} className="h-[470px] animate-pulse rounded-[1.75rem] bg-muted/80" />
              ))}
            </div>
          ) : null}

          {!loading && !error && products.length === 0 ? (
            <Card className="surface-card">
              <CardContent className="state-panel">
                <p className="text-lg font-semibold">No products matched those filters.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Try lowering the rating threshold, expanding the price range, or clearing filters.
                </p>
                <Button
                  variant="outline"
                  onClick={() => {
                    router.replace(pathname, { scroll: false });
                  }}
                >
                  Reset search
                </Button>
              </CardContent>
            </Card>
          ) : null}

          {!loading && !error && products.length > 0 ? (
            <>
              <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
                {products.map((product) => (
                  <ProductCard
                    key={product.id}
                    product={product}
                    categoryName={categoryById.get(product.category_id)}
                    wishlisted={wishlistedProductIds.has(product.id)}
                    wishlistLoading={wishlistBusyId === product.id}
                    onToggleWishlist={toggleWishlist}
                  />
                ))}
              </div>

              <div className="surface-card flex flex-col gap-3 rounded-[1.5rem] p-4 sm:flex-row sm:items-center sm:justify-between">
                <p className="text-sm text-muted-foreground">
                  Showing page {meta?.page ?? 1} of {totalPages}. Total{" "}
                  {meta?.total ?? products.length} products.
                </p>
                <div className="flex gap-2">
                  {(meta?.page ?? 1) <= 1 ? (
                    <Button variant="outline" disabled>
                      Previous
                    </Button>
                  ) : (
                    <Button variant="outline" asChild>
                      <Link
                        href={buildPathWithParams({
                          page: Math.max(1, (meta?.page ?? 1) - 1),
                        })}
                        scroll={false}
                      >
                        Previous
                      </Link>
                    </Button>
                  )}
                  {(meta?.page ?? 1) >= totalPages ? (
                    <Button variant="outline" disabled>
                      Next
                    </Button>
                  ) : (
                    <Button variant="outline" asChild>
                      <Link href={buildPathWithParams({ page: (meta?.page ?? 1) + 1 })} scroll={false}>
                        Next
                      </Link>
                    </Button>
                  )}
                </div>
              </div>
            </>
          ) : null}
        </main>
      </div>
    </div>
  );
}

export default function ProductsPage() {
  return (
    <Suspense fallback={<p className="text-sm text-muted-foreground">Loading products...</p>}>
      <ProductsBrowser />
    </Suspense>
  );
}
