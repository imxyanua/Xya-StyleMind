import type { components } from "@/types/openapi";

type GeneratedStats = components["schemas"]["AdminDashboardStats"];
type GeneratedOrdersByStatus = components["schemas"]["OrdersByStatus"];
type GeneratedRecentOrder = components["schemas"]["DashboardRecentOrder"];
type GeneratedLowStockProduct = components["schemas"]["DashboardLowStockProduct"];
type GeneratedRevenueByDay = components["schemas"]["DashboardRevenueByDay"];
type GeneratedTopProduct = components["schemas"]["DashboardTopProduct"];

export type DashboardOrdersByStatus = GeneratedOrdersByStatus & {
  pending: number;
  paid: number;
  shipping: number;
  completed: number;
  cancelled: number;
};

export type DashboardRecentOrder = GeneratedRecentOrder & {
  id: string;
  user_id: string;
  user_email: string;
  user_name: string;
  status: "pending" | "paid" | "shipping" | "completed" | "cancelled";
  total_amount: number;
  created_at: string;
};

export type DashboardLowStockProduct = GeneratedLowStockProduct & {
  id: string;
  name: string;
  stock: number;
  reserved_quantity: number;
  available_stock: number;
  price: number;
  image_url: string;
};

export type DashboardRevenueByDay = GeneratedRevenueByDay & {
  date: string;
  revenue: number;
};

export type DashboardTopProduct = GeneratedTopProduct & {
  id: string;
  name: string;
  image_url: string;
  quantity_sold: number;
  revenue: number;
};

export type AdminDashboardStats = GeneratedStats & {
  total_revenue: number;
  total_orders: number;
  total_products: number;
  total_users: number;
  active_reservations: number;
  orders_by_status: DashboardOrdersByStatus;
  recent_orders: DashboardRecentOrder[];
  low_stock_products: DashboardLowStockProduct[];
  revenue_by_day: DashboardRevenueByDay[];
  top_products: DashboardTopProduct[];
};
