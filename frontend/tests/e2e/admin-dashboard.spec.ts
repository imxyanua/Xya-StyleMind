import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { Client } from "pg";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080/api/v1";
const PASSWORD = "Password123!";

type ApiEnvelope<T> = {
  success: boolean;
  message: string;
  data?: T;
};

type AuthPayload = {
  token: string;
  user: {
    id: string;
    email: string;
    role: string;
  };
};

type Product = {
  id: string;
  name: string;
  stock: number;
};

type Order = {
  id: string;
  status: string;
  user?: {
    email?: string;
    full_name?: string;
  };
};

async function apiGet<T>(request: APIRequestContext, path: string, token?: string) {
  const response = await request.get(`${API_BASE_URL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()) as ApiEnvelope<T>;
}

async function apiPost<T>(
  request: APIRequestContext,
  path: string,
  body: Record<string, unknown> = {},
  token?: string
) {
  const response = await request.post(`${API_BASE_URL}${path}`, {
    data: body,
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()) as ApiEnvelope<T>;
}

async function registerViaApi(request: APIRequestContext, email: string, fullName = "E2E User") {
  return apiPost<AuthPayload>(request, "/auth/register", {
    email,
    full_name: fullName,
    password: PASSWORD,
  });
}

async function loginThroughUi(page: Page, email: string) {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page).toHaveURL(/\/products|\/admin/);
}

function dbClient() {
  return new Client({
    host: process.env.E2E_DB_HOST ?? "localhost",
    port: Number(process.env.E2E_DB_PORT ?? "5432"),
    user: process.env.E2E_DB_USER ?? "postgres",
    password: process.env.E2E_DB_PASSWORD ?? "postgres",
    database: process.env.E2E_DB_NAME ?? "stylemind",
  });
}

async function promoteUserToAdmin(email: string) {
  const client = dbClient();
  await client.connect();
  try {
    await client.query("UPDATE users SET role = 'admin', updated_at = NOW() WHERE email = $1", [
      email.toLowerCase(),
    ]);
  } finally {
    await client.end();
  }
}

async function setupAdmin(page: Page, request: APIRequestContext, suffix: string) {
  const email = `admin-${suffix}@example.com`;
  await registerViaApi(request, email, "E2E Admin");
  await promoteUserToAdmin(email);
  await loginThroughUi(page, email);
  return email;
}

async function createBuyerOrder(request: APIRequestContext, suffix: string) {
  const buyerEmail = `buyer-${suffix}@example.com`;
  const buyer = await registerViaApi(request, buyerEmail, "E2E Buyer");
  const token = buyer.data?.token;
  expect(token).toBeTruthy();

  const productResponse = await apiGet<Product[]>(
    request,
    "/products?in_stock=true&sort=newest&limit=10"
  );
  const product = productResponse.data?.find((item) => item.stock > 0);
  expect(product, "seed data must include an in-stock product for checkout").toBeTruthy();

  await apiPost(
    request,
    "/cart/items",
    {
      product_id: (product as Product).id,
      quantity: 1,
    },
    token
  );
  const orderResponse = await apiPost<Order>(request, "/orders", {}, token);
  expect(orderResponse.data?.id).toBeTruthy();
  return { order: orderResponse.data as Order, buyerEmail };
}

test("admin auth protects dashboard access", async ({ page, request }) => {
  const suffix = String(Date.now());

  await page.goto("/admin");
  await expect(page).toHaveURL(/\/login\?redirect=\/admin/);

  const userEmail = `regular-${suffix}@example.com`;
  await registerViaApi(request, userEmail, "E2E Regular User");
  await loginThroughUi(page, userEmail);
  await page.goto("/admin");
  await expect(page.getByText("Admin access required.")).toBeVisible();
  await page.goto("/admin/orders");
  await expect(page.getByText("Admin access required.")).toBeVisible();
  await page.goto("/admin/activity");
  await expect(page.getByText("Admin access required.")).toBeVisible();

  await setupAdmin(page, request, suffix);
  await page.goto("/admin");
  await expect(page.getByRole("heading", { name: "Admin Dashboard" })).toBeVisible();
  await expect(page.getByText("Revenue").first()).toBeVisible();
  await expect(page.getByText("Orders by Status")).toBeVisible();
  await expect(page.getByText("Revenue by Day")).toBeVisible();
  await expect(page.getByText("Recent Orders")).toBeVisible();
  await expect(page.getByText("Low-Stock Products")).toBeVisible();
  await expect(page.getByRole("main").getByRole("link", { name: "Products", exact: true }).first()).toBeVisible();
  await expect(page.getByRole("main").getByRole("link", { name: "Categories", exact: true }).first()).toBeVisible();
  await expect(page.getByRole("main").getByRole("link", { name: "Orders", exact: true }).first()).toBeVisible();
  await expect(page.getByRole("main").getByRole("link", { name: "Activity", exact: true }).first()).toBeVisible();
});

test("admin can create categories and manage products", async ({ page, request }) => {
  const suffix = String(Date.now());
  const categoryName = `E2E Category ${suffix}`;
  const categorySlug = `e2e-category-${suffix}`;
  const productName = `E2E Admin Jacket ${suffix}`;

  await setupAdmin(page, request, suffix);

  await page.goto("/admin/categories");
  await expect(page.getByText("Create Category")).toBeVisible();
  await page.getByRole("button", { name: "Create" }).click();
  const nameIsMissing = await page
    .locator("#name")
    .evaluate((element: HTMLInputElement) => element.validity.valueMissing);
  expect(nameIsMissing).toBe(true);

  await page.getByLabel("Name").fill(categoryName);
  await page.getByLabel("Slug").fill(categorySlug);
  await page.getByRole("button", { name: "Create" }).click();
  await expect(page.getByText("Category created.")).toBeVisible();
  await expect(page.getByText(categoryName)).toBeVisible();
  await expect(page.getByText(`/${categorySlug}`)).toBeVisible();

  await page.goto("/admin/products");
  await expect(page.getByText("Create Product")).toBeVisible();
  await page.getByLabel("Product name").fill(productName);
  await page.getByLabel("Description").fill("Admin E2E product used for dashboard coverage.");
  await page.getByLabel("Price").fill("1234000");
  await page.getByLabel("Stock").fill("11");
  await page.getByLabel("Category").selectOption({ label: categoryName });
  await page.getByLabel("Style").selectOption("formal");
  await page.getByLabel("Color").selectOption("brown");
  await page
    .getByLabel("Image URL")
    .fill(`https://picsum.photos/seed/admin-product-${suffix}/640/800`);
  await page.getByRole("button", { name: "Create" }).click();
  await expect(page.getByText("Product created.")).toBeVisible();
  await expect(page.getByText(productName)).toBeVisible();

  await page.goto(`/products?q=${encodeURIComponent(productName)}&limit=12`);
  await expect(page.getByText(productName)).toBeVisible();

  await page.goto("/admin/products");
  const productRow = page
    .getByText(productName)
    .locator(
      "xpath=ancestor::div[.//button[normalize-space()='Edit'] and .//button[normalize-space()='Delete']][1]"
    );
  await productRow.getByRole("button", { name: "Edit" }).click();
  await expect(page.getByText("Update Product")).toBeVisible();
  await page.getByLabel("Price").fill("2345000");
  await page.getByLabel("Stock").fill("7");
  await page.getByLabel("Style").selectOption("minimal");
  await page.getByLabel("Color").selectOption("black");
  await page.getByRole("button", { name: "Update" }).click();
  await expect(page.getByText("Product updated.")).toBeVisible();
  const editedRow = page
    .getByText(productName)
    .locator(
      "xpath=ancestor::div[.//button[normalize-space()='Edit'] and .//button[normalize-space()='Delete']][1]"
    );
  await expect(editedRow.getByText("Stock 7")).toBeVisible();

  page.once("dialog", async (dialog) => {
    expect(dialog.message()).toContain("Delete this product?");
    await dialog.accept();
  });
  const updatedRow = page
    .getByText(productName)
    .locator(
      "xpath=ancestor::div[.//button[normalize-space()='Edit'] and .//button[normalize-space()='Delete']][1]"
    );
  await updatedRow.getByRole("button", { name: "Delete" }).click();
  await expect(page.getByText("Product deleted.")).toBeVisible();
  await expect(page.getByText(productName)).toBeHidden();
});

test("admin can update order status and sees invalid transition errors", async ({
  page,
  request,
}) => {
  const suffix = String(Date.now());
  const { order, buyerEmail } = await createBuyerOrder(request, suffix);

  await setupAdmin(page, request, `orders-${suffix}`);
  await page.goto("/admin/orders");
  await expect(page.getByText("Update Order Status")).toBeVisible();
  await expect(page.getByText("Admin Orders")).toBeVisible();

  await page.getByLabel("Search order ID").fill(order.id);
  await page.getByRole("button", { name: "Apply" }).click();
  await expect(page.getByText(order.id)).toBeVisible();
  await expect(page.getByText(buyerEmail).first()).toBeVisible();

  await page.getByLabel("Order ID", { exact: true }).fill(order.id);
  await page.getByLabel("Status", { exact: true }).selectOption("paid");
  await page.getByRole("button", { name: "Update status" }).click();
  await expect(page.getByText("Order status updated.")).toBeVisible();
  await expect(page.getByText("Last Updated Order")).toBeVisible();
  await expect(page.getByText(order.id)).toBeVisible();
  await expect(page.locator("p").filter({ hasText: /^paid$/ }).first()).toBeVisible();

  await page.getByLabel("Status filter").selectOption("paid");
  await page.getByRole("button", { name: "Apply" }).click();
  await expect(page.getByText(order.id)).toBeVisible();
  await expect(page.getByText(buyerEmail).first()).toBeVisible();

  await page.getByLabel("Order ID", { exact: true }).fill(order.id);
  await page.getByLabel("Status", { exact: true }).selectOption("completed");
  await page.getByRole("button", { name: "Update status" }).click();
  await expect(page.getByText(/invalid order status transition/i)).toBeVisible();

  await page.goto("/admin/activity");
  await expect(page.getByRole("heading", { name: "Activity Log" })).toBeVisible();
  await page.getByLabel("Action").fill("admin.order_status.update");
  await page.getByLabel("Resource").selectOption("order");
  await page.getByLabel("Result").selectOption("success");
  await page.getByRole("button", { name: "Apply" }).click();
  await expect(page.getByText("admin.order_status.update").first()).toBeVisible();
  await expect(page.getByText(/\"old_status\": \"pending\"/).first()).toBeVisible();
  await expect(page.getByText(/\"new_status\": \"paid\"/).first()).toBeVisible();
});
