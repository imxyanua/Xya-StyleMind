import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

const highlights = [
  ["36+", "seeded products"],
  ["6", "style moods"],
  ["Real API", "cart, orders, reviews"],
];

const journeys = [
  {
    title: "Browse the catalog",
    description: "Search, filter, sort, and save pieces from the live Go backend.",
  },
  {
    title: "Build a cart",
    description: "Move from detail page to cart, checkout, and order history.",
  },
  {
    title: "Review after purchase",
    description: "Ratings and review permissions follow the backend order policy.",
  },
];

export default function HomePage() {
  return (
    <div className="space-y-10">
      <section className="relative overflow-hidden rounded-[2.5rem] border border-border bg-card shadow-soft">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(255,255,255,0.95),transparent_28rem),radial-gradient(circle_at_80%_0%,rgba(88,106,75,0.2),transparent_24rem)]" />
        <div className="relative grid gap-10 p-6 sm:p-10 lg:grid-cols-[1.1fr_0.9fr] lg:p-14">
          <div className="flex flex-col justify-center gap-7">
            <p className="eyebrow">AI-powered fashion ecommerce</p>
            <div className="space-y-4">
              <h1 className="max-w-4xl text-5xl font-semibold leading-[0.95] sm:text-6xl lg:text-7xl">
                Style discovery with a real commerce backbone.
              </h1>
              <p className="max-w-2xl text-base leading-7 text-muted-foreground sm:text-lg">
                Xya-StyleMind combines a polished shopping experience with a production-minded Go
                backend: auth, product search, cart, checkout, orders, wishlist, and reviews.
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Button size="lg" asChild>
                <Link href="/products">Explore products</Link>
              </Button>
              <Button size="lg" variant="outline" asChild>
                <Link href="/register">Create account</Link>
              </Button>
            </div>
          </div>

          <div className="rounded-[2rem] border border-border/80 bg-foreground p-4 text-primary-foreground shadow-product">
            <div className="aspect-[4/5] rounded-[1.5rem] bg-[linear-gradient(160deg,rgba(255,255,255,0.25),transparent_35%),radial-gradient(circle_at_70%_20%,rgba(212,188,143,0.85),transparent_10rem),linear-gradient(145deg,#2b3327,#11140f)] p-5">
              <div className="flex h-full flex-col justify-between">
                <div className="flex justify-between text-xs uppercase tracking-[0.25em] text-primary-foreground/70">
                  <span>Nova edit</span>
                  <span>SS26</span>
                </div>
                <div className="space-y-3">
                  <div className="h-40 rounded-full bg-primary-foreground/10 blur-2xl" />
                  <div>
                    <p className="font-heading text-4xl">Minimal streetwear</p>
                    <p className="mt-2 text-sm leading-6 text-primary-foreground/72">
                      Curated silhouettes, backend-driven inventory, and reviews that only unlock
                      after checkout.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="grid gap-4 md:grid-cols-3">
        {highlights.map(([value, label]) => (
          <Card key={label} className="surface-card">
            <CardContent className="p-6">
              <p className="font-heading text-4xl font-semibold">{value}</p>
              <p className="mt-1 text-sm text-muted-foreground">{label}</p>
            </CardContent>
          </Card>
        ))}
      </section>

      <section className="grid gap-4 lg:grid-cols-3">
        {journeys.map((item, index) => (
          <Card key={item.title} className="surface-card">
            <CardContent className="p-6">
              <span className="grid size-10 place-items-center rounded-2xl bg-secondary text-sm font-semibold">
                0{index + 1}
              </span>
              <h2 className="mt-5 text-2xl font-semibold">{item.title}</h2>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">{item.description}</p>
            </CardContent>
          </Card>
        ))}
      </section>
    </div>
  );
}
