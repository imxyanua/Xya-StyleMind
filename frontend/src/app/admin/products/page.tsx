"use client";

import { FormEvent, useEffect, useState } from "react";

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
    <div className="grid gap-6 xl:grid-cols-[420px_1fr]">
      <Card>
        <CardHeader>
          <CardTitle>{editingId ? "Update Product" : "Create Product"}</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={onSubmit}>
            <Input value={form.name} onChange={(event) => updateField("name", event.target.value)} placeholder="Name" required />
            <textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
              value={form.description}
              onChange={(event) => updateField("description", event.target.value)}
              placeholder="Description"
              required
            />
            <div className="grid gap-3 sm:grid-cols-2">
              <Input type="number" value={form.price} onChange={(event) => updateField("price", event.target.value)} placeholder="Price" required />
              <Input type="number" value={form.stock} onChange={(event) => updateField("stock", event.target.value)} placeholder="Stock" required />
            </div>
            <select
              className="h-8 w-full rounded-lg border border-input bg-background px-3 text-sm"
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
            <div className="grid gap-3 sm:grid-cols-2">
              <select
                className="h-8 rounded-lg border border-input bg-background px-3 text-sm"
                value={form.style}
                onChange={(event) => updateField("style", event.target.value)}
              >
                {styles.map((style) => (
                  <option key={style} value={style}>
                    {style}
                  </option>
                ))}
              </select>
              <select
                className="h-8 rounded-lg border border-input bg-background px-3 text-sm"
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
            <Input value={form.image_url} onChange={(event) => updateField("image_url", event.target.value)} placeholder="Image URL" required />
            {error ? <p className="text-sm text-red-600">{error}</p> : null}
            {success ? <p className="text-sm text-green-700">{success}</p> : null}
            <div className="flex gap-2">
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

      <Card>
        <CardHeader>
          <CardTitle>Products</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? <p className="text-sm text-muted-foreground">Loading products...</p> : null}
          <div className="space-y-3">
            {products.map((product) => (
              <div key={product.id} className="flex flex-col gap-3 rounded-md border p-3 lg:flex-row lg:items-center lg:justify-between">
                <div>
                  <p className="font-medium">{product.name}</p>
                  <p className="text-sm text-muted-foreground">
                    {product.style} · {product.color} · Stock {product.stock}
                  </p>
                </div>
                <div className="flex gap-2">
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
        </CardContent>
      </Card>
    </div>
  );
}
