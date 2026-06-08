"use client";

import { FormEvent, useEffect, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { createCategory, fetchCategories } from "@/lib/api";
import type { Category } from "@/types/category";

export default function AdminCategoriesPage() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function loadCategories() {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchCategories();
      setCategories(res.data ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch categories");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let cancelled = false;

    fetchCategories()
      .then((res) => {
        if (!cancelled) {
          setCategories(res.data ?? []);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to fetch categories");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError(null);
    setSuccess(null);

    try {
      await createCategory({ name, slug });
      setName("");
      setSlug("");
      setSuccess("Category created.");
      await loadCategories();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create category");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="grid min-w-0 gap-6 lg:grid-cols-[380px_minmax(0,1fr)]">
      <Card className="surface-card h-fit rounded-[1.75rem] lg:sticky lg:top-28">
        <CardHeader>
          <p className="eyebrow">Catalog taxonomy</p>
          <CardTitle className="text-3xl">Create Category</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-1.5">
              <label htmlFor="name" className="text-sm font-medium">
                Name
              </label>
              <Input
                id="name"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Outerwear"
                required
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="slug" className="text-sm font-medium">
                Slug
              </label>
              <Input
                id="slug"
                value={slug}
                onChange={(event) => setSlug(event.target.value)}
                placeholder="outerwear"
                required
              />
              <p className="text-xs text-muted-foreground">
                Use lowercase words and hyphens for stable public URLs.
              </p>
            </div>
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            {success ? <p className="text-sm text-primary">{success}</p> : null}
            <Button type="submit" disabled={saving}>
              {saving ? "Saving..." : "Create"}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card className="surface-card rounded-[1.75rem]">
        <CardHeader>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow">Navigation groups</p>
              <CardTitle className="mt-2 text-3xl">Categories</CardTitle>
            </div>
            <Badge variant="secondary">{categories.length} categories</Badge>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="grid gap-3 sm:grid-cols-2">
              {Array.from({ length: 6 }).map((_, index) => (
                <div key={index} className="h-24 animate-pulse rounded-2xl bg-muted/80" />
              ))}
            </div>
          ) : null}

          {!loading && categories.length === 0 ? (
            <div className="state-panel">
              <p className="text-xl font-semibold">No categories yet.</p>
              <p className="max-w-md text-sm text-muted-foreground">
                Create categories before adding product records.
              </p>
            </div>
          ) : null}

          {!loading && categories.length > 0 ? (
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              {categories.map((category, index) => (
                <div key={category.id} className="rounded-2xl border border-border bg-card/70 p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-heading text-2xl font-semibold">{category.name}</p>
                      <p className="mt-1 text-sm text-muted-foreground">/{category.slug}</p>
                    </div>
                    <span className="rounded-full bg-secondary px-2.5 py-1 text-xs font-semibold">
                      #{index + 1}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
