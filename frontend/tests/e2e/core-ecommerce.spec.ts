import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { Client } from "pg";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080/api/v1";
const PASSWORD = "Password123!";

type ApiEnvelope<T> = {
  success: boolean;
  message: string;
  data?: T;
  meta?: {
    page: number;
    limit: number;
    total: number;
    total_page: number;
    total_pages?: number;
  };
};

type Product = {
  id: string;
  name: string;
  stock: number;
  style: string;
  color: string;
};

type Order = {
  id: string;
  status: string;
  payment_status: string;
  items: Array<{ product_id: string }>;
};

async function apiGet<T>(request: APIRequestContext, path: string, token?: string) {
  const response = await request.get(`${API_BASE_URL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()) as ApiEnvelope<T>;
}

async function registerThroughUi(page: Page, email: string) {
  await page.goto("/register");
  await page.getByLabel("Full name").fill("E2E Shopper");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Register" }).click();
  await expect(page).toHaveURL(/\/products/);
}

async function loginThroughUi(page: Page, email: string) {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page).toHaveURL(/\/products/);
}

async function authToken(page: Page) {
  const token = await page.evaluate(() => window.localStorage.getItem("stylemind_token"));
  expect(token).toBeTruthy();
  return token as string;
}

async function markOrderPaid(orderID: string) {
  const client = new Client({
    host: process.env.E2E_DB_HOST ?? "localhost",
    port: Number(process.env.E2E_DB_PORT ?? "5432"),
    user: process.env.E2E_DB_USER ?? "postgres",
    password: process.env.E2E_DB_PASSWORD ?? "postgres",
    database: process.env.E2E_DB_NAME ?? "stylemind",
  });
  await client.connect();
  try {
    await client.query("UPDATE orders SET status = 'paid', updated_at = NOW() WHERE id = $1", [
      orderID,
    ]);
  } finally {
    await client.end();
  }
}

async function createDefaultAddressThroughUi(page: Page) {
  await page.goto("/profile/addresses");
  await expect(page.getByRole("heading", { name: "Address Book" })).toBeVisible();
  await page.getByLabel("Recipient name").fill("E2E Saved Buyer");
  await page.getByLabel("Phone").fill("0909876543");
  await page.getByLabel("Address").fill("77 Saved Address Lane");
  await page.getByLabel("City").fill("Ho Chi Minh City");
  await page.getByLabel("District").fill("District 7");
  await page.getByLabel("Delivery note").fill("Saved address note");
  await page.getByRole("button", { name: "Save address" }).click();
  await expect(page.getByText("Address saved.")).toBeVisible();
  await page.getByRole("button", { name: "Set default" }).first().click();
  await expect(page.getByText("Default address updated.")).toBeVisible();
  await expect(page.getByText("E2E Saved Buyer").last()).toBeVisible();
}

test("core ecommerce flows work against the real backend", async ({ page, request }) => {
  const productsResponse = await apiGet<Product[]>(
    request,
    "/products?in_stock=true&sort=newest&limit=3"
  );
  const product = productsResponse.data?.find((item) => item.stock > 1);
  expect(product, "seed data must contain at least one in-stock product").toBeTruthy();
  const selectedProduct = product as Product;
  const searchKeyword = selectedProduct.name.split(" ")[0];

  await page.goto(`/products?limit=12&q=${encodeURIComponent(searchKeyword)}`);
  await expect(page.getByText(selectedProduct.name)).toBeVisible();
  await page.getByLabel("Style").selectOption(selectedProduct.style);
  await expect(page).toHaveURL(new RegExp(`style=${selectedProduct.style}`));
  await page.getByLabel("Color").selectOption(selectedProduct.color);
  await expect(page).toHaveURL(new RegExp(`color=${selectedProduct.color}`));
  await page.getByLabel("Sort products").selectOption("price_desc");
  await expect(page).toHaveURL(/sort=price_desc/);

  await page.goto("/products?limit=12");
  const nextPageLink = page.getByRole("link", { name: "Next" });
  if (await nextPageLink.isVisible()) {
    await nextPageLink.click();
    await expect(page).toHaveURL(/page=2/);
  }

  const email = `e2e-${Date.now()}@example.com`;
  await registerThroughUi(page, email);
  await page.evaluate(() => window.localStorage.removeItem("stylemind_token"));
  await page.goto("/cart");
  await expect(page).toHaveURL(/\/login\?redirect=\/cart/);
  await loginThroughUi(page, email);
  await createDefaultAddressThroughUi(page);

  await page.goto(`/products/${selectedProduct.id}`);
  await expect(page.getByRole("heading", { name: selectedProduct.name })).toBeVisible();
  await expect(page.getByText(/Based on \d+ reviews?/)).toBeVisible();
  await expect(page.getByText(/Reviews are available after checkout/)).toBeVisible();

  await page.getByRole("button", { name: "Add to wishlist" }).click();
  await expect(page.getByRole("button", { name: "Remove from wishlist" })).toBeVisible();

  await page.route("**/api/v1/wishlist/products/**", async (route) => {
    if (route.request().method() === "DELETE") {
      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ success: false, message: "forced wishlist failure" }),
      });
      return;
    }
    await route.continue();
  });
  await page.getByRole("button", { name: "Remove from wishlist" }).click();
  await expect(page.getByText("forced wishlist failure")).toBeVisible();
  await expect(page.getByRole("button", { name: "Remove from wishlist" })).toHaveAttribute(
    "aria-pressed",
    "true"
  );
  await page.unroute("**/api/v1/wishlist/products/**");

  await page.goto("/wishlist");
  await expect(page.getByText(selectedProduct.name)).toBeVisible();
  await page.getByRole("button", { name: "Remove" }).first().click();
  await expect(page.getByText(selectedProduct.name)).toBeHidden();

  await page.goto(`/products/${selectedProduct.id}`);
  await page.getByRole("button", { name: "Add to Cart" }).click();
  await expect(page.getByText("Added to cart")).toBeVisible();
  await page.goto("/cart");
  await expect(page.getByText(selectedProduct.name)).toBeVisible();
  await page.getByRole("spinbutton").first().fill("2");
  await page.getByRole("button", { name: "Update" }).first().click();
  await expect(page.getByLabel("Recipient name")).toHaveValue("E2E Saved Buyer");
  await expect(page.getByLabel("Phone")).toHaveValue("0909876543");
  await expect(page.getByLabel("Address")).toHaveValue("77 Saved Address Lane");
  await expect(page.getByLabel("City")).toHaveValue("Ho Chi Minh City");
  await expect(page.getByLabel("District")).toHaveValue("District 7");
  await expect(page.getByLabel("Delivery note")).toHaveValue("Saved address note");
  await page.getByLabel("Shipping method").selectOption("express");
  await page.getByLabel("Payment method").selectOption("demo_payment");
  await page.getByRole("button", { name: "Checkout" }).click();
  await expect(page).toHaveURL(/\/orders/);
  await expect(page.getByText(selectedProduct.name)).toBeVisible();
  await expect(page.getByText("E2E Saved Buyer")).toBeVisible();
  await expect(page.getByText(/77 Saved Address Lane/)).toBeVisible();
  await expect(page.getByText("Express delivery").first()).toBeVisible();
  await expect(page.getByText("Payment: Demo payment")).toBeVisible();
  await expect(page.getByText("Payment status: pending").first()).toBeVisible();

  const token = await authToken(page);
  const ordersResponse = await apiGet<Order[]>(request, "/orders?limit=20", token);
  const createdOrder = ordersResponse.data?.find((order) =>
    order.items.some((item) => item.product_id === selectedProduct.id)
  );
  expect(createdOrder, "checkout should create an order containing selected product").toBeTruthy();
  const createdOrderId = (createdOrder as Order).id;

  await page.getByRole("link", { name: "View details" }).first().click();
  await expect(page).toHaveURL(new RegExp(`/orders/${createdOrderId}`));
  await expect(page.getByText("Order progress")).toBeVisible();
  await expect(page.getByText("Shipping contact")).toBeVisible();
  await expect(page.getByText("Order summary")).toBeVisible();
  await expect(page.getByText("Payment", { exact: true })).toBeVisible();
  await expect(page.getByText(selectedProduct.name)).toBeVisible();
  await expect(page.getByText("E2E Saved Buyer")).toBeVisible();
  await expect(page.getByText(/77 Saved Address Lane/)).toBeVisible();
  await expect(page.getByText("Express delivery").first()).toBeVisible();
  await expect(page.getByText("Demo payment").first()).toBeVisible();
  await expect(page.getByText("Payment status: pending").first()).toBeVisible();

  await page.goto(`/orders/${createdOrderId}`);
  await expect(page.getByText("Order progress")).toBeVisible();
  await expect(page.getByText(selectedProduct.name)).toBeVisible();

  await markOrderPaid(createdOrderId);

  await page.goto(`/products/${selectedProduct.id}`);
  await expect(page.getByRole("button", { name: "Submit review" })).toBeVisible();
  await page.getByLabel("Rating").selectOption("5");
  await page.getByLabel("Comment").fill("Excellent E2E fit and fabric.");
  await page.getByRole("button", { name: "Submit review" }).click();
  await expect(page.getByText("Review created.")).toBeVisible();
  await expect(page.getByText("Excellent E2E fit and fabric.").last()).toBeVisible();

  await page.getByLabel("Rating").selectOption("4");
  await page.getByLabel("Comment").fill("Updated E2E review after another look.");
  await page.getByRole("button", { name: "Update review" }).click();
  await expect(page.getByText("Review updated.")).toBeVisible();
  await expect(page.getByText("Updated E2E review after another look.").last()).toBeVisible();

  await page.getByRole("button", { name: "Delete review" }).click();
  await expect(page.getByText("Review deleted.")).toBeVisible();

  await page.goto("/products?limit=12");
  await page.getByRole("button", { name: "Add to wishlist" }).first().click();
  await page.goto("/wishlist");
  await expect(page.getByRole("heading", { name: "Wishlist" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Add to Cart" }).first()).toBeVisible();

  await request.post(`${API_BASE_URL}/auth/logout`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  await page.goto(`/orders/${createdOrderId}`);
  await expect(page).toHaveURL(/\/login\?redirect=\/orders\//);
  await page.goto("/cart");
  await expect(page).toHaveURL(/\/login\?redirect=\/cart/);
});
