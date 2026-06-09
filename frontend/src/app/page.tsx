"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowRight, Heart, PackageCheck, Search, ShieldCheck, Sparkles, Truck } from "lucide-react";

import { ProductCard } from "@/components/product/product-card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchCategories, fetchProducts } from "@/lib/api";
import type { Category } from "@/types/category";
import type { Product } from "@/types/product";

const styleCollections = [
  {
    label: "Streetwear",
    href: "/products?style=streetwear",
    description: "Graphic layers, utility cuts, and city-ready silhouettes.",
    tone: "from-neutral-950 to-neutral-700",
  },
  {
    label: "Minimal",
    href: "/products?style=minimal",
    description: "Clean essentials with quiet colors and sharper proportions.",
    tone: "from-stone-500 to-stone-300",
  },
  {
    label: "Korean",
    href: "/products?style=korean",
    description: "Soft structure, relaxed volume, and editorial everyday styling.",
    tone: "from-rose-300 to-amber-100",
  },
  {
    label: "Formal",
    href: "/products?style=formal",
    description: "Polished pieces for office days, events, and smart dinners.",
    tone: "from-slate-800 to-slate-500",
  },
];

const trustItems = [
  {
    icon: Sparkles,
    title: "AI-ready discovery",
    description: "Built for future outfit recommendation and smart search flows.",
  },
  {
    icon: ShieldCheck,
    title: "Verified review flow",
    description: "Reviews are tied to real checkout history so ratings stay useful.",
  },
  {
    icon: PackageCheck,
    title: "Real ecommerce core",
    description: "Cart, checkout, orders, wishlist, and admin inventory are connected.",
  },
  {
    icon: Truck,
    title: "Demo operations",
    description: "Shipping and return pages are ready for storefront expectations.",
  },
];

type HomeSections = {
  featured: Product[];
  newest: Product[];
  topRated: Product[];
  categories: Category[];
};

function ProductSkeleton() {
  return <div className="h-[470px] animate-pulse rounded-[1.5rem] bg-muted/80" />;
}

function SectionHeader({ eyebrow, title, description, href }: { eyebrow: string; title: string; description: string; href: string }) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p className="eyebrow">{eyebrow}</p>
        <h2 className="mt-2 text-3xl font-semibold sm:text-4xl">{title}</h2>
        <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">{description}</p>
      </div>
      <Button variant="outline" asChild>
        <Link href={href}>
          View all <ArrowRight className="size-4" />
        </Link>
      </Button>
    </div>
  );
}

function ProductRail({ products, loading }: { products: Product[]; loading: boolean }) {
  if (loading) {
    return (
      <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <ProductSkeleton key={index} />
        ))}
      </div>
    );
  }

  if (products.length === 0) {
    return (
      <Card className="surface-card rounded-[1.75rem]">
        <CardContent className="state-panel">
          <p className="text-xl font-semibold">No products to show yet.</p>
          <p className="max-w-md text-sm text-muted-foreground">
            Start the backend and seed products to populate this storefront section.
          </p>
          <Button asChild>
            <Link href="/products">Open shop</Link>
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="grid gap-5 sm:grid-cols-2 xl:grid-cols-4">
      {products.slice(0, 4).map((product) => (
        <ProductCard key={product.id} product={product} />
      ))}
    </div>
  );
}

export default function HomePage() {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [sections, setSections] = useState<HomeSections>({
    featured: [],
    newest: [],
    topRated: [],
    categories: [],
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    async function loadHome() {
      setLoading(true);
      setError(null);
      try {
        const [featured, newest, topRated, categories] = await Promise.all([
          fetchProducts({ page: 1, limit: 4, sort: "rating_desc", in_stock: true }),
          fetchProducts({ page: 1, limit: 4, sort: "newest", in_stock: true }),
          fetchProducts({ page: 1, limit: 4, sort: "rating_desc", min_rating: 4 }),
          fetchCategories(),
        ]);

        if (!active) {
          return;
        }

        setSections({
          featured: featured.data ?? [],
          newest: newest.data ?? [],
          topRated: topRated.data ?? featured.data ?? [],
          categories: categories.data ?? [],
        });
      } catch (err) {
        if (active) {
          setError(err instanceof Error ? err.message : "Could not load storefront data.");
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void loadHome();
    return () => {
      active = false;
    };
  }, []);

  const categoryCards = useMemo(() => sections.categories.slice(0, 6), [sections.categories]);

  function onSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const query = search.trim();
    router.push(query ? `/products?q=${encodeURIComponent(query)}` : "/products");
  }

  return (
    <div className="space-y-14">
      <section className="relative overflow-hidden rounded-[2.5rem] border border-border bg-card shadow-soft">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_12%,rgba(206,222,184,0.55),transparent_18rem),radial-gradient(circle_at_80%_20%,rgba(255,255,255,0.55),transparent_16rem)] dark:bg-[radial-gradient(circle_at_18%_12%,rgba(206,222,184,0.16),transparent_18rem),radial-gradient(circle_at_80%_20%,rgba(255,255,255,0.08),transparent_16rem)]" />
        <div className="relative grid min-h-[560px] gap-8 p-6 sm:p-8 lg:grid-cols-[1.08fr_0.92fr] lg:p-10">
          <div className="flex flex-col justify-center space-y-7">
            <Badge variant="secondary" className="w-fit gap-2 px-3 py-1">
              <Sparkles className="size-3.5" /> Fashion commerce with AI-ready intelligence
            </Badge>
            <div>
              <h1 className="max-w-4xl text-5xl font-semibold leading-[0.96] sm:text-6xl lg:text-7xl">
                Shop outfits that feel curated before you even filter.
              </h1>
              <p className="mt-5 max-w-2xl text-base leading-7 text-muted-foreground sm:text-lg">
                Xya-StyleMind brings together product search, wishlist, verified reviews,
                checkout, and admin-ready catalog operations in one polished storefront.
              </p>
            </div>
            <form onSubmit={onSearchSubmit} role="search" className="flex max-w-2xl flex-col gap-3 rounded-[1.5rem] border border-border bg-background/80 p-2 shadow-soft backdrop-blur sm:flex-row">
              <label htmlFor="home-search" className="sr-only">
                Search products
              </label>
              <div className="flex min-w-0 flex-1 items-center gap-2 px-3">
                <Search className="size-4 text-muted-foreground" aria-hidden="true" />
                <input
                  id="home-search"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="Try black streetwear, beige minimal, formal blazer..."
                  className="h-11 min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
                />
              </div>
              <Button type="submit" size="lg" className="rounded-2xl">
                Search shop
              </Button>
            </form>
            <div className="flex flex-wrap gap-3">
              <Button asChild size="lg">
                <Link href="/products">
                  Shop all products <ArrowRight className="size-4" />
                </Link>
              </Button>
              <Button asChild variant="outline" size="lg">
                <Link href="/wishlist">
                  Open wishlist <Heart className="size-4" />
                </Link>
              </Button>
            </div>
          </div>

          <div className="grid content-center gap-4 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
            <Card className="surface-card rounded-[2rem] p-2">
              <CardHeader>
                <p className="eyebrow">Live catalog</p>
                <CardTitle className="text-3xl">{loading ? "Loading" : `${sections.featured.length}+`} picks</CardTitle>
                <CardDescription>Products are pulled from the backend API, not mock data.</CardDescription>
              </CardHeader>
            </Card>
            <Card className="surface-card rounded-[2rem] p-2 sm:translate-y-6">
              <CardHeader>
                <p className="eyebrow">Verified flow</p>
                <CardTitle className="text-3xl">Cart ? Order ? Review</CardTitle>
                <CardDescription>Core ecommerce paths are wired end-to-end.</CardDescription>
              </CardHeader>
            </Card>
            <Card className="surface-card rounded-[2rem] p-2 xl:col-span-2">
              <CardContent className="grid gap-3 p-5 sm:grid-cols-3">
                <div>
                  <p className="text-3xl font-semibold">6</p>
                  <p className="text-xs text-muted-foreground">style filters</p>
                </div>
                <div>
                  <p className="text-3xl font-semibold">JWT</p>
                  <p className="text-xs text-muted-foreground">secure account flow</p>
                </div>
                <div>
                  <p className="text-3xl font-semibold">Admin</p>
                  <p className="text-xs text-muted-foreground">catalog operations</p>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </section>

      {error ? (
        <p className="rounded-2xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      <section className="space-y-6">
        <SectionHeader
          eyebrow="Featured products"
          title="Curated now"
          description="A quick rail of high-signal products for shoppers who want to start browsing immediately."
          href="/products?sort=rating_desc"
        />
        <ProductRail products={sections.featured} loading={loading} />
      </section>

      <section className="space-y-6">
        <SectionHeader
          eyebrow="New arrivals"
          title="Fresh drops"
          description="Newest catalog items, ideal for homepage traffic that wants the latest edit first."
          href="/products?sort=newest"
        />
        <ProductRail products={sections.newest} loading={loading} />
      </section>

      <section className="space-y-6">
        <SectionHeader
          eyebrow="Best sellers / top rated"
          title="Loved by shoppers"
          description="Rating-aware listing powered by product reviews, ready for real merchandising."
          href="/products?sort=rating_desc&min_rating=4"
        />
        <ProductRail products={sections.topRated} loading={loading} />
      </section>

      <section className="space-y-6">
        <SectionHeader
          eyebrow="Collections"
          title="Shop by style"
          description="Direct links into product filters so homepage discovery becomes real browsing."
          href="/products"
        />
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {styleCollections.map((collection) => (
            <Link
              key={collection.href}
              href={collection.href}
              className={`group relative min-h-56 overflow-hidden rounded-[2rem] border border-border bg-gradient-to-br ${collection.tone} p-6 text-white shadow-soft transition hover:-translate-y-1`}
            >
              <div className="absolute inset-0 bg-[radial-gradient(circle_at_80%_15%,rgba(255,255,255,0.36),transparent_10rem)]" />
              <div className="relative flex h-full flex-col justify-between">
                <Badge className="w-fit border-white/30 bg-white/15 text-white" variant="outline">
                  {collection.label}
                </Badge>
                <div>
                  <h3 className="text-3xl font-semibold">{collection.label}</h3>
                  <p className="mt-2 text-sm leading-6 text-white/78">{collection.description}</p>
                </div>
              </div>
            </Link>
          ))}
        </div>
      </section>

      <section className="space-y-6">
        <SectionHeader
          eyebrow="Category edits"
          title="Browse the catalog map"
          description="Categories come from the backend, so admin-created groups appear here automatically."
          href="/products"
        />
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {loading
            ? Array.from({ length: 6 }).map((_, index) => (
                <div key={index} className="h-24 animate-pulse rounded-[1.5rem] bg-muted/80" />
              ))
            : categoryCards.map((category) => (
                <Link
                  key={category.id}
                  href={`/products?category_id=${encodeURIComponent(category.id)}`}
                  className="surface-card flex items-center justify-between gap-4 rounded-[1.5rem] p-5 transition hover:-translate-y-0.5 hover:bg-muted/60"
                >
                  <div>
                    <h3 className="text-xl font-semibold">{category.name}</h3>
                    <p className="text-xs text-muted-foreground">/{category.slug}</p>
                  </div>
                  <ArrowRight className="size-5 text-muted-foreground" />
                </Link>
              ))}
        </div>
      </section>

      <section className="grid gap-5 lg:grid-cols-[0.9fr_1.1fr]">
        <Card className="surface-card rounded-[2rem]">
          <CardHeader className="p-6 sm:p-8">
            <p className="eyebrow">Why choose us</p>
            <CardTitle className="text-4xl">A storefront built like the backend matters.</CardTitle>
            <CardDescription className="text-base leading-7">
              This is not just a portfolio landing page. The frontend is connected to real API flows:
              auth, product filters, wishlist, checkout, orders, reviews, and admin operations.
            </CardDescription>
          </CardHeader>
        </Card>
        <div className="grid gap-4 sm:grid-cols-2">
          {trustItems.map((item) => {
            const Icon = item.icon;
            return (
              <Card key={item.title} className="surface-card rounded-[1.75rem]">
                <CardHeader>
                  <span className="grid size-11 place-items-center rounded-2xl bg-secondary text-secondary-foreground">
                    <Icon className="size-5" />
                  </span>
                  <CardTitle className="text-xl">{item.title}</CardTitle>
                  <CardDescription>{item.description}</CardDescription>
                </CardHeader>
              </Card>
            );
          })}
        </div>
      </section>

      <section className="surface-card overflow-hidden rounded-[2.5rem] p-6 sm:p-8 lg:p-10">
        <div className="grid gap-7 lg:grid-cols-[1fr_auto] lg:items-center">
          <div>
            <p className="eyebrow">Ready to style the next cart?</p>
            <h2 className="mt-2 max-w-3xl text-4xl font-semibold sm:text-5xl">
              Start with product search, save favorites, then checkout when the fit feels right.
            </h2>
          </div>
          <div className="flex flex-wrap gap-3">
            <Button asChild size="lg">
              <Link href="/products">Shop now</Link>
            </Button>
            <Button asChild variant="outline" size="lg">
              <Link href="/about">Learn about StyleMind</Link>
            </Button>
          </div>
        </div>
      </section>
    </div>
  );
}
