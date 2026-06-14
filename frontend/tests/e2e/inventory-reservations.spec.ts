import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { Client } from "pg";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://127.0.0.1:8080/api/v1";
const PASSWORD = "Password123!";

type ApiEnvelope<T> = {
  success: boolean;
  message: string;
  data?: T;
  meta?: {
    total: number;
  };
};

type AuthPayload = {
  token: string;
  user: {
    id: string;
    email: string;
    role: string;
  };
};

type Order = {
  id: string;
  items: Array<{ product_id: string; quantity: number }>;
};

type Reservation = {
  id: string;
  user_id: string;
  product_id: string;
  quantity: number;
  expires_at: string;
};

type DashboardStats = {
  active_reservations: number;
  low_stock_products: Array<{
    id: string;
    name: string;
    stock: number;
    reserved_quantity: number;
    available_stock: number;
  }>;
};

function dbClient() {
  return new Client({
    host: process.env.E2E_DB_HOST ?? "127.0.0.1",
    port: Number(process.env.E2E_DB_PORT ?? "5432"),
    user: process.env.E2E_DB_USER ?? "postgres",
    password: process.env.E2E_DB_PASSWORD ?? "postgres",
    database: process.env.E2E_DB_NAME ?? "stylemind",
  });
}

async function apiPost(
  request: APIRequestContext,
  path: string,
  body: Record<string, unknown>,
  token?: string
) {
  const response = await request.post(`${API_BASE_URL}${path}`, {
    data: body,
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  return response;
}

async function apiGet<T>(request: APIRequestContext, path: string, token?: string) {
  const response = await request.get(`${API_BASE_URL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()) as ApiEnvelope<T>;
}

async function registerViaApi(request: APIRequestContext, email: string, fullName: string) {
  const response = await apiPost(request, "/auth/register", {
    email,
    full_name: fullName,
    password: PASSWORD,
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()) as ApiEnvelope<AuthPayload>;
}

async function loginViaApi(request: APIRequestContext, email: string) {
  const response = await apiPost(request, "/auth/login", {
    email,
    password: PASSWORD,
  });
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()) as ApiEnvelope<AuthPayload>;
}

async function loginThroughUi(page: Page, email: string) {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page).toHaveURL(/\/products|\/admin/);
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

async function createCatalogProduct(suffix: string, stock: number) {
  const client = dbClient();
  await client.connect();
  const categoryID = crypto.randomUUID();
  const productID = crypto.randomUUID();
  const productName = `E2E Reserved Stock ${suffix}`;
  try {
    await client.query(
      `
        INSERT INTO categories (id, name, slug)
        VALUES ($1, $2, $3)
      `,
      [categoryID, `E2E Inventory ${suffix}`, `e2e-inventory-${suffix}`]
    );
    await client.query(
      `
        INSERT INTO products (
          id, name, description, price, stock, category_id, style, color, image_url
        )
        VALUES ($1, $2, $3, $4, $5, $6, 'minimal', 'black', $7)
      `,
      [
        productID,
        productName,
        "E2E product for inventory reservation regression.",
        250000,
        stock,
        categoryID,
        `https://picsum.photos/seed/${productID}/640/800`,
      ]
    );
  } finally {
    await client.end();
  }
  return { categoryID, productID, productName };
}

async function insertReservation(userID: string, productID: string, quantity: number, expiresInMinutes: number) {
  const client = dbClient();
  await client.connect();
  const reservationID = crypto.randomUUID();
  try {
    await client.query(
      `
        INSERT INTO inventory_reservations (id, user_id, product_id, quantity, expires_at)
        VALUES ($1, $2, $3, $4, NOW() + ($5::text || ' minutes')::interval)
      `,
      [reservationID, userID, productID, quantity, expiresInMinutes]
    );
  } finally {
    await client.end();
  }
  return reservationID;
}

async function productStock(productID: string) {
  const client = dbClient();
  await client.connect();
  try {
    const result = await client.query<{ stock: number }>("SELECT stock FROM products WHERE id = $1", [
      productID,
    ]);
    return result.rows[0]?.stock ?? 0;
  } finally {
    await client.end();
  }
}

async function activeReservationCount(productID: string) {
  const client = dbClient();
  await client.connect();
  try {
    const result = await client.query<{ count: string }>(
      "SELECT COUNT(*)::text FROM inventory_reservations WHERE product_id = $1 AND expires_at > NOW()",
      [productID]
    );
    return Number(result.rows[0]?.count ?? 0);
  } finally {
    await client.end();
  }
}

async function addToCart(request: APIRequestContext, token: string, productID: string, quantity: number) {
  const response = await apiPost(request, "/cart/items", { product_id: productID, quantity }, token);
  expect(response.ok(), await response.text()).toBeTruthy();
}

async function checkout(request: APIRequestContext, token: string) {
  return apiPost(
    request,
    "/orders",
    {
      recipient_name: "Inventory E2E Buyer",
      phone: "0901234567",
      address_line: "123 Reservation Street",
      city: "Ho Chi Minh City",
      district: "District 1",
      note: "Inventory reservation regression",
      shipping_method: "standard",
      payment_method: "cod",
    },
    token
  );
}

test("checkout reservations prevent overselling and update dashboard stock signals", async ({
  page,
  request,
}) => {
  const suffix = `${Date.now()}-${Math.round(Math.random() * 10000)}`;

  const userA = await registerViaApi(request, `inventory-a-${suffix}@example.com`, "Inventory User A");
  const userB = await registerViaApi(request, `inventory-b-${suffix}@example.com`, "Inventory User B");
  const userC = await registerViaApi(request, `inventory-c-${suffix}@example.com`, "Inventory User C");
  const admin = await registerViaApi(request, `inventory-admin-${suffix}@example.com`, "Inventory Admin");
  const tokenA = userA.data?.token;
  const tokenB = userB.data?.token;
  const tokenC = userC.data?.token;
  const adminEmail = admin.data?.user.email;
  let adminToken = admin.data?.token;
  expect(tokenA).toBeTruthy();
  expect(tokenB).toBeTruthy();
  expect(tokenC).toBeTruthy();
  expect(adminEmail).toBeTruthy();
  expect(adminToken).toBeTruthy();
  await promoteUserToAdmin(adminEmail as string);
  adminToken = (await loginViaApi(request, adminEmail as string)).data?.token;
  expect(adminToken).toBeTruthy();

  const successProduct = await createCatalogProduct(`success-${suffix}`, 2);
  await addToCart(request, tokenA as string, successProduct.productID, 2);
  const successCheckout = await checkout(request, tokenA as string);
  expect(successCheckout.status(), await successCheckout.text()).toBe(201);
  const successOrder = (await successCheckout.json()) as ApiEnvelope<Order>;
  expect(successOrder.data?.items).toContainEqual(
    expect.objectContaining({ product_id: successProduct.productID, quantity: 2 })
  );
  await expect.poll(() => productStock(successProduct.productID)).toBe(0);
  await expect.poll(() => activeReservationCount(successProduct.productID)).toBe(0);

  const reservedProduct = await createCatalogProduct(`active-${suffix}`, 2);
  await insertReservation(userA.data!.user.id, reservedProduct.productID, 1, 15);
  await addToCart(request, tokenB as string, reservedProduct.productID, 2);
  const oversellCheckout = await checkout(request, tokenB as string);
  expect(oversellCheckout.status(), await oversellCheckout.text()).toBe(400);
  await expect(oversellCheckout.json()).resolves.toMatchObject({
    success: false,
    message: "insufficient stock",
  });
  await expect.poll(() => productStock(reservedProduct.productID)).toBe(2);

  const expiredProduct = await createCatalogProduct(`expired-${suffix}`, 2);
  await insertReservation(userA.data!.user.id, expiredProduct.productID, 2, -15);
  await addToCart(request, tokenC as string, expiredProduct.productID, 2);
  const expiredIgnoredCheckout = await checkout(request, tokenC as string);
  expect(expiredIgnoredCheckout.status(), await expiredIgnoredCheckout.text()).toBe(201);
  await expect.poll(() => productStock(expiredProduct.productID)).toBe(0);

  const userAReservations = await apiGet<Reservation[]>(
    request,
    "/me/reservations?limit=20",
    tokenA
  );
  expect(userAReservations.data?.some((item) => item.id && item.product_id === reservedProduct.productID)).toBe(
    true
  );
  const userBReservations = await apiGet<Reservation[]>(
    request,
    "/me/reservations?limit=20",
    tokenB
  );
  expect(userBReservations.data?.some((item) => item.product_id === reservedProduct.productID)).toBe(
    false
  );

  const dashboardProduct = await createCatalogProduct(`dashboard-${suffix}`, 1);
  await insertReservation(userA.data!.user.id, dashboardProduct.productID, 1, 15);

  const dashboard = await apiGet<DashboardStats>(request, "/admin/dashboard/stats", adminToken);
  expect(dashboard.data?.active_reservations ?? 0).toBeGreaterThan(0);
  const lowStockReservedProduct = dashboard.data?.low_stock_products.find(
    (item) => item.id === dashboardProduct.productID
  );
  expect(lowStockReservedProduct).toMatchObject({
    stock: 1,
    reserved_quantity: 1,
    available_stock: 0,
  });

  await loginThroughUi(page, adminEmail as string);
  await page.goto("/admin");
  await expect(page.getByText("Reservations")).toBeVisible();
  const dashboardProductLink = page.getByRole("link", {
    name: new RegExp(dashboardProduct.productName),
  });
  await expect(dashboardProductLink).toBeVisible();
  await expect(dashboardProductLink.getByText("Available 0")).toBeVisible();
  await expect(dashboardProductLink.getByText("1 reserved / 1 stock")).toBeVisible();
});
