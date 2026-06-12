import type { components } from "@/types/openapi";

type GeneratedCoupon = components["schemas"]["Coupon"];
type GeneratedApplyCouponResult = components["schemas"]["ApplyCouponResult"];

export type CouponType = "percent" | "fixed";

export type Coupon = GeneratedCoupon & {
  id: string;
  code: string;
  type: CouponType;
  value: number;
  min_order_amount: number;
  max_discount_amount?: number | null;
  usage_limit?: number | null;
  used_count: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type ApplyCouponResult = GeneratedApplyCouponResult & {
  coupon_id: string;
  coupon_code: string;
  type: CouponType;
  value: number;
  subtotal_amount: number;
  discount_amount: number;
  total_amount: number;
};
