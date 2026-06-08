"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Image from "next/image";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  createProduct,
  deleteProduct,
  fetchCategories,
  fetchProducts,
  updateProduct,
  type ProductInput,
} from "@/lib/api";
import { PRODUCT_IMAGE_BLUR } from "@/lib/images";
import type { Category } from "@/types/category";
import type { Product } from "@/types/product";

const styles = ["streetwear", "minimal", "korean", "formal", "casual", "sporty"];
const colors = ["black", "white", "beige", "blue", "gray", "brown"];

const emptyForm = {
  name: "",
  description: "",
  price: "",
  stock: "",
  category_id: "",
  style: "casual",
  color: "black",
  image_url: "",
};

type ProductFormState = typeof emptyForm;

function toInput(form: ProductFormState): ProductInput {
  return {
    name: form.name,
    description: form.description,
    price: Number(form.price),
    stock: Number(form.stock),
    category_id: form.category_id,
    style: form.style,
    color: form.color,
    image_url: form.image_url,
  };
}

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

export default function AdminProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [form, setForm] = useState<ProductFormState>(emptyForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function loadData() {
    setLoading(true);
    setError(null);
    try {
      const [productRes, categoryRes] = await Promise.all([
        fetchProducts({ limit: 100 }),
        fetchCategories(),
      ]);
      setProducts(productRes.data ?? []);
      setCategories(categoryRes.data ?? []);
      if (!form.category_id && categoryRes.data?.[0]?.id) {
        setForm((prev) => ({ ...prev, category_id: categoryRes.data?.[0]?.id ?? "" }));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load admin products");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let cancelled = false;

    Promise.all([fetchProducts({ limit: 100 }), fetchCategories()])
      .then(([productRes, categoryRes]) => {
        if (cancelled) {
          return;
        }
        setProducts(productRes.data ?? []);
        setCategories(categoryRes.data ?? []);
        if (categoryRes.data?.[0]?.id) {
          setForm((prev) => ({ ...prev, category_id: categoryRes.data?.[0]?.id ?? "" }));
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load admin products");
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

  const categoryById = useMemo(() => {
    return new Map(categories.map((category) => [category.id, category.name]));
  }, [categories]);

  function updateField(field: keyof ProductFormState, value: string) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  function startEdit(product: Product) {
    setEditingId(product.id);
    setSuccess(null);
    setError(null);
    setForm({
      name: product.name,
      description: product.description,
      price: String(product.price),
      stock: String(product.stock),
      category_id: product.category_id,
      style: product.style,
      color: product.color,
      image_url: product.image_url,
    });
  }

  function resetForm() {
    setEditingId(null);
    setForm({
      ...emptyForm,
      category_id: categories[0]?.id ?? "",
    });
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError(null);
    setSuccess(null);

    try {
      if (editingId) {
        await updateProduct(editingId, toInput(form));
        setSuccess("Product updated.");
      } else {
        await createProduct(toInput(form));
        setSuccess("Product created.");
      }
      resetForm();
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save product");
    } finally {
      setSaving(false);
    }
  }

  async function onDelete(id: string) {
    if (!window.confirm("Delete this product?")) {
      return;
    }

    setError(null);
    setSuccess(null);
    try {
      await deleteProduct(id);
      setSuccess("Product deleted.");
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete product");
    }
  }

  return (
    <div className="grid min-w-0 gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
      <Card className="surface-card h-fit rounded-[1.75rem] xl:sticky xl:top-28">
        <CardHeader>
          <p className="eyebrow">{editingId ? "Edit catalog item" : "New catalog item"}</p>
          <CardTitle className="text-3xl">{editingId ? "Update Product" : "Create Product"}</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-1.5">
              <label htmlFor="product-name" className="text-sm font-medium">
                Product name
              </label>
              <Input
                id="product-name"
                value={form.name}
                onChange={(event) => updateField("name", event.target.value)}
                placeholder="Minimal Beige Jacket"
                required
              />
            </div>

            <div className="space-y-1.5">
              <label htmlFor="product-description" className="text-sm font-medium">
                Description
              </label>
              <textarea
                id="product-description"
                className="min-h-28 w-full rounded-xl border border-input bg-card px-3 py-2 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                value={form.description}
                onChange={(event) => updateField("description", event.target.value)}
                placeholder="Material, fit, mood, and styling notes..."
                required
              />
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <label htmlFor="product-price" className="text-sm font-medium">
                  Price
                </label>
                <Input
                  id="product-price"
                  type="number"
                  min="1"
                  value={form.price}
                  onChange={(event) => updateField("price", event.target.value)}
                  placeholder="990000"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="product-stock" className="text-sm font-medium">
                  Stock
                </label>
                <Input
                  id="product-stock"
                  type="number"
                  min="0"
                  value={form.stock}
                  onChange={(event) => updateField("stock", event.target.value)}
                  placeholder="24"
                  required
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label htmlFor="product-category" className="text-sm font-medium">
                Category
              </label>
              <select
                id="product-category"
                className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                value={form.category_id}
                onChange={(event) => updateField("category_id", event.target.value)}
                required
              >
                <option value="">Select category</option>
                {categories.map((category) => (
                  <option key={category.id} value={category.id}>
                    {category.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <label htmlFor="product-style" className="text-sm font-medium">
                  Style
                </label>
                <select
                  id="product-style"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={form.style}
                  onChange={(event) => updateField("style", event.target.value)}
                >
                  {styles.map((style) => (
                    <option key={style} value={style}>
                      {style}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="product-color" className="text-sm font-medium">
                  Color
                </label>
                <select
                  id="product-color"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={form.color}
                  onChange={(event) => updateField("color", event.target.value)}
                >
                  {colors.map((color) => (
                    <option key={color} value={color}>
                      {color}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="space-y-1.5">
              <label htmlFor="product-image" className="text-sm font-medium">
                Image URL
              </label>
              <Input
                id="product-image"
                value={form.image_url}
                onChange={(event) => updateField("image_url", event.target.value)}
                placeholder="https://..."
                required
              />
            </div>

            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            {success ? <p className="text-sm text-primary">{success}</p> : null}

            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={saving}>
                {saving ? "Saving..." : editingId ? "Update" : "Create"}
              </Button>
              {editingId ? (
                <Button type="button" variant="outline" onClick={resetForm}>
                  Cancel
                </Button>
              ) : null}
            </div>
          </form>
        </CardContent>
      </Card>

      <Card className="surface-card rounded-[1.75rem]">
        <CardHeader>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow">Inventory</p>
              <CardTitle className="mt-2 text-3xl">Products</CardTitle>
            </div>
            <Badge variant="secondary">{products.length} items</Badge>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="grid gap-3">
              {Array.from({ length: 5 }).map((_, index) => (
                <div key={index} className="h-24 animate-pulse rounded-2xl bg-muted/80" />
              ))}
            </div>
          ) : null}

          {!loading && products.length === 0 ? (
            <div className="state-panel">
              <p className="text-xl font-semibold">No products yet.</p>
              <p className="max-w-md text-sm text-muted-foreground">
                Create the first catalog item with the form on the left.
              </p>
            </div>
          ) : null}

          {!loading && products.length > 0 ? (
            <div className="overflow-x-auto rounded-2xl border border-border">
              <div className="hidden grid-cols-[1.8fr_1fr_120px_180px] gap-4 bg-muted/60 px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground lg:grid">
                <span>Product</span>
                <span>Category</span>
                <span>Stock</span>
                <span className="text-right">Actions</span>
              </div>
              <div className="divide-y divide-border">
                {products.map((product) => (
                  <div
                    key={product.id}
                    className="grid min-w-0 gap-4 p-4 lg:grid-cols-[minmax(0,1.8fr)_minmax(0,1fr)_120px_180px] lg:items-center"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <div className="relative size-16 shrink-0 overflow-hidden rounded-2xl bg-muted">
                        <Image
                          src={product.image_url}
                          alt={product.name}
                          fill
                          className="object-cover"
                          sizes="64px"
                          placeholder="blur"
                          blurDataURL={PRODUCT_IMAGE_BLUR}
                        />
                      </div>
                      <div className="min-w-0">
                        <p className="truncate font-heading text-xl font-semibold">{product.name}</p>
                        <p className="text-sm text-muted-foreground">{formatVND(product.price)}</p>
                        <div className="mt-2 flex flex-wrap gap-1.5">
                          <Badge variant="secondary">{product.style}</Badge>
                          <Badge variant="outline">{product.color}</Badge>
                        </div>
                      </div>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {categoryById.get(product.category_id) ?? "Uncategorized"}
                    </p>
                    <Badge variant={product.stock > 0 ? "outline" : "destructive"}>
                      Stock {product.stock}
                    </Badge>
                    <div className="flex flex-wrap justify-start gap-2 lg:justify-end">
                      <Button variant="outline" onClick={() => startEdit(product)}>
                        Edit
                      </Button>
                      <Button variant="destructive" onClick={() => onDelete(product.id)}>
                        Delete
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
