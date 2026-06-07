import type { components } from "@/types/openapi";

type GeneratedAdminUser = components["schemas"]["AdminUser"];

export type AdminUser = GeneratedAdminUser & {
  id: string;
  email: string;
  full_name: string;
  role: "user" | "admin";
  status: "active" | "disabled";
  created_at: string;
  updated_at: string;
};
