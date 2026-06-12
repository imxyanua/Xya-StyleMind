import { ApiError, type ApiResponse } from "@/types/api";
import type { AuditLog } from "@/types/audit";
import type { Cart } from "@/types/cart";
import type { Category } from "@/types/category";
import type { AdminDashboardStats } from "@/types/dashboard";
import type { components, operations } from "@/types/openapi";
import type { Order } from "@/types/order";
import type { Product } from "@/types/product";
import type { RatingSummary, Review } from "@/types/review";
import type { AdminUser } from "@/types/user";
import type { WishlistItem } from "@/types/wishlist";

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

export type ProductListParams = NonNullable<
  operations["listProducts"]["parameters"]["query"]
>;

export async function fetchProducts(params: ProductListParams = {}) {
  const searchParams = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }
    searchParams.set(key, String(value));
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

export type ProductInput = components["schemas"]["ProductMutationRequest"];

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

export type ReviewListParams = NonNullable<
  operations["listProductReviews"]["parameters"]["query"]
>;
export type CreateReviewInput = components["schemas"]["CreateReviewRequest"];
export type UpdateReviewInput = components["schemas"]["UpdateReviewRequest"];

export async function fetchProductReviews(productId: string, params: ReviewListParams = {}) {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined) {
      continue;
    }
    searchParams.set(key, String(value));
  }
  const query = searchParams.toString();
  return apiRequest<Review[]>(
    `/products/${productId}/reviews${query ? `?${query}` : ""}`,
    { method: "GET" },
    { auth: false }
  );
}

export async function fetchRatingSummary(productId: string) {
  return apiRequest<RatingSummary>(
    `/products/${productId}/rating-summary`,
    { method: "GET" },
    { auth: false }
  );
}

export async function createReview(productId: string, input: CreateReviewInput) {
  return apiRequest<Review>(`/products/${productId}/reviews`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateReview(id: string, input: UpdateReviewInput) {
  return apiRequest<Review>(`/reviews/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteReview(id: string) {
  return apiRequest<{ id: string }>(`/reviews/${id}`, { method: "DELETE" });
}

export type WishlistListParams = NonNullable<operations["listWishlist"]["parameters"]["query"]>;

export async function fetchWishlist(params: WishlistListParams = {}) {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined) {
      continue;
    }
    searchParams.set(key, String(value));
  }
  const query = searchParams.toString();
  return apiRequest<WishlistItem[]>(`/wishlist${query ? `?${query}` : ""}`, { method: "GET" });
}

export async function addWishlistProduct(productId: string) {
  return apiRequest<{ product_id: string }>(`/wishlist/products/${productId}`, {
    method: "POST",
  });
}

export async function removeWishlistProduct(productId: string) {
  return apiRequest<{ product_id: string }>(`/wishlist/products/${productId}`, {
    method: "DELETE",
  });
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

export type CheckoutInput = components["schemas"]["CheckoutRequest"];
export type UserAddress = components["schemas"]["UserAddress"];
export type AddressInput = components["schemas"]["AddressRequest"];

export async function checkoutOrder(input: CheckoutInput) {
  return apiRequest<Order>("/orders", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function fetchMyAddresses() {
  return apiRequest<UserAddress[]>("/me/addresses", { method: "GET" });
}

export async function createMyAddress(input: AddressInput) {
  return apiRequest<UserAddress>("/me/addresses", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateMyAddress(id: string, input: AddressInput) {
  return apiRequest<UserAddress>(`/me/addresses/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteMyAddress(id: string) {
  return apiRequest<{ id: string }>(`/me/addresses/${id}`, { method: "DELETE" });
}

export async function setDefaultMyAddress(id: string) {
  return apiRequest<UserAddress>(`/me/addresses/${id}/default`, { method: "PATCH" });
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

export async function fetchMyOrder(id: string) {
  return apiRequest<Order>(`/orders/${id}`, { method: "GET" });
}

export type AdminOrderListParams = NonNullable<
  operations["listAdminOrders"]["parameters"]["query"]
>;

export async function fetchAdminOrders(params: AdminOrderListParams = {}) {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }
    searchParams.set(key, String(value));
  }
  const query = searchParams.toString();
  return apiRequest<Order[]>(`/admin/orders${query ? `?${query}` : ""}`, { method: "GET" });
}

export async function fetchAdminOrder(id: string) {
  return apiRequest<Order>(`/admin/orders/${id}`, { method: "GET" });
}

export async function updateOrderStatus(id: string, status: Order["status"]) {
  return apiRequest<Order>(`/admin/orders/${id}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });
}

export type AdminAuditLogListParams = NonNullable<
  operations["listAdminAuditLogs"]["parameters"]["query"]
>;

export async function fetchAdminAuditLogs(params: AdminAuditLogListParams = {}) {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }
    searchParams.set(key, String(value));
  }
  const query = searchParams.toString();
  return apiRequest<AuditLog[]>(`/admin/audit-logs${query ? `?${query}` : ""}`, {
    method: "GET",
  });
}

export type AdminUserListParams = NonNullable<
  operations["listAdminUsers"]["parameters"]["query"]
>;
export type UpdateAdminUserRoleInput = components["schemas"]["UpdateUserRoleRequest"];
export type UpdateAdminUserStatusInput = components["schemas"]["UpdateUserStatusRequest"];

export async function fetchAdminUsers(params: AdminUserListParams = {}) {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }
    searchParams.set(key, String(value));
  }
  const query = searchParams.toString();
  return apiRequest<AdminUser[]>(`/admin/users${query ? `?${query}` : ""}`, {
    method: "GET",
  });
}

export async function fetchAdminUser(id: string) {
  return apiRequest<AdminUser>(`/admin/users/${id}`, { method: "GET" });
}

export async function updateAdminUserRole(id: string, input: UpdateAdminUserRoleInput) {
  return apiRequest<AdminUser>(`/admin/users/${id}/role`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function updateAdminUserStatus(id: string, input: UpdateAdminUserStatusInput) {
  return apiRequest<AdminUser>(`/admin/users/${id}/status`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export type AdminDashboardStatsParams = NonNullable<
  operations["getAdminDashboardStats"]["parameters"]["query"]
>;

export async function fetchAdminDashboardStats(params: AdminDashboardStatsParams = {}) {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }
    searchParams.set(key, String(value));
  }
  const query = searchParams.toString();
  return apiRequest<AdminDashboardStats>(
    `/admin/dashboard/stats${query ? `?${query}` : ""}`,
    { method: "GET" }
  );
}
