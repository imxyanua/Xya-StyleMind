import type { components } from "@/types/openapi";

type GeneratedNotification = components["schemas"]["Notification"];

export type NotificationMetadata = {
  order_id?: string;
  return_request_id?: string;
  status?: string;
  payment_status?: string;
  old_status?: string;
  new_status?: string;
  old_payment_status?: string;
  new_payment_status?: string;
  total_amount?: number;
  discount_amount?: number;
  coupon_code?: string;
  [key: string]: unknown;
};

export type UserNotification = Omit<GeneratedNotification, "metadata"> & {
  id: string;
  user_id: string;
  type: string;
  title: string;
  message: string;
  metadata?: NotificationMetadata;
  read_at?: string | null;
  created_at: string;
};
