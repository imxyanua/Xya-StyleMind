import type { components } from "@/types/openapi";

type GeneratedOrderProduct = components["schemas"]["OrderProduct"];
type GeneratedOrderUser = components["schemas"]["OrderUser"];
type GeneratedOrderItem = components["schemas"]["OrderItem"];
type GeneratedOrder = components["schemas"]["Order"];

export type OrderProduct = GeneratedOrderProduct & {
  id: string;
  name: string;
  image_url: string;
  style: string;
  color: string;
};

export type OrderUser = GeneratedOrderUser & {
  id: string;
  email: string;
  full_name: string;
  role: "user" | "admin";
};

export type OrderItem = Omit<GeneratedOrderItem, "product"> & {
  id: string;
  product_id: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
  product: OrderProduct;
};

export type Order = Omit<GeneratedOrder, "items" | "user"> & {
  id: string;
  user_id: string;
  user?: OrderUser;
  status: "pending" | "paid" | "shipping" | "completed" | "cancelled";
  total_amount: number;
  items: OrderItem[];
  created_at: string;
  updated_at: string;
};
