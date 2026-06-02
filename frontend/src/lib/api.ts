import { ApiError, type ApiResponse } from "@/types/api";
import type { Cart } from "@/types/cart";
import type { Category } from "@/types/category";
import type { Order } from "@/types/order";
import type { Product } from "@/types/product";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";

type RequestOptions = {
  auth?: boolean;
};

function getTokenFromStorage(): string | null {
  if (typeof window === "undefined") {
    return null;
  }
  return window.localStorage.getItem("stylemind_token");
}

export async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
  options: RequestOptions = {}
): Promise<ApiResponse<T>> {
  const auth = options.auth ?? true;
  const headers = new Headers(init.headers ?? {});

  if (!headers.has("Content-Type") && init.body) {
    headers.set("Content-Type", "application/json");
  }

  if (auth) {
    const token = getTokenFromStorage();
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
  });

  let payload: ApiResponse<T>;
  try {
    payload = (await response.json()) as ApiResponse<T>;
  } catch {
    throw new ApiError("Invalid server response", response.status);
  }

  if (!response.ok || !payload.success) {
    throw new ApiError(payload.message || "Request failed", response.status);
  }

  return payload;
}

type ProductListParams = {
  style?: string;
  color?: string;
  page?: number;
  limit?: number;
};

export async function fetchProducts(params: ProductListParams = {}) {
  const searchParams = new URLSearchParams();
  if (params.style) {
    searchParams.set("style", params.style);
  }
  if (params.color) {
    searchParams.set("color", params.color);
  }
  if (params.page) {
    searchParams.set("page", String(params.page));
  }
  if (params.limit) {
    searchParams.set("limit", String(params.limit));
  }

  const query = searchParams.toString();
  return apiRequest<Product[]>(
    `/products${query ? `?${query}` : ""}`,
    { method: "GET" },
    { auth: false }
  );
}

export async function fetchProductById(id: string) {
  return apiRequest<Product>(`/products/${id}`, { method: "GET" }, { auth: false });
}

export async function fetchCategories() {
  return apiRequest<Category[]>("/categories?limit=100", { method: "GET" }, { auth: false });
}

export type ProductInput = {
  name: string;
  description: string;
  price: number;
  stock: number;
  category_id: string;
  style: string;
  color: string;
  image_url: string;
};

export async function createCategory(input: { name: string; slug: string }) {
  return apiRequest<Category>("/admin/categories", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function createProduct(input: ProductInput) {
  return apiRequest<Product>("/admin/products", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateProduct(id: string, input: ProductInput) {
  return apiRequest<Product>(`/admin/products/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteProduct(id: string) {
  return apiRequest<{ id: string }>(`/admin/products/${id}`, { method: "DELETE" });
}

export async function addToCart(productId: string, quantity: number) {
  return apiRequest<Cart>("/cart/items", {
    method: "POST",
    body: JSON.stringify({
      product_id: productId,
      quantity,
    }),
  });
}

export async function fetchCart() {
  return apiRequest<Cart>("/cart", { method: "GET" });
}

export async function updateCartItem(itemId: string, quantity: number) {
  return apiRequest<Cart>(`/cart/items/${itemId}`, {
    method: "PUT",
    body: JSON.stringify({ quantity }),
  });
}

export async function removeCartItem(itemId: string) {
  return apiRequest<Cart>(`/cart/items/${itemId}`, { method: "DELETE" });
}

export async function checkoutOrder() {
  return apiRequest<Order>("/orders", { method: "POST" });
}

type OrderListParams = {
  page?: number;
  limit?: number;
};

export async function fetchMyOrders(params: OrderListParams = {}) {
  const searchParams = new URLSearchParams();
  if (params.page) {
    searchParams.set("page", String(params.page));
  }
  if (params.limit) {
    searchParams.set("limit", String(params.limit));
  }
  const query = searchParams.toString();
  return apiRequest<Order[]>(`/orders${query ? `?${query}` : ""}`, { method: "GET" });
}

export async function updateOrderStatus(id: string, status: Order["status"]) {
  return apiRequest<Order>(`/admin/orders/${id}/status`, {
    method: "PUT",
    body: JSON.stringify({ status }),
  });
}
