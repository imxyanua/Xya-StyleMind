import type { components } from "@/types/openapi";
import type { Order, OrderUser } from "@/types/order";

type GeneratedReturnOrder = components["schemas"]["ReturnOrder"];
type GeneratedReturnUser = components["schemas"]["ReturnUser"];
type GeneratedReturnRequest = components["schemas"]["ReturnRequest"];

export type ReturnRequestStatus = "requested" | "approved" | "rejected" | "cancelled";

export type ReturnOrder = GeneratedReturnOrder &
  Pick<
    Order,
    | "id"
    | "user_id"
    | "status"
    | "payment_status"
    | "payment_method"
    | "shipping_method"
    | "total_amount"
    | "created_at"
  >;

export type ReturnUser = GeneratedReturnUser &
  Pick<OrderUser, "id" | "email" | "full_name" | "role">;

export type ReturnRequest = Omit<GeneratedReturnRequest, "order" | "user"> & {
  id: string;
  order_id: string;
  user_id: string;
  reason: string;
  status: ReturnRequestStatus;
  admin_note?: string;
  order?: ReturnOrder;
  user?: ReturnUser;
  created_at: string;
  updated_at: string;
};
