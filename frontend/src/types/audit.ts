import type { components } from "@/types/openapi";

type GeneratedAuditLog = components["schemas"]["AuditLog"];

export type AuditLog = GeneratedAuditLog & {
  id: string;
  actor_role: "user" | "admin";
  action: string;
  resource_type: "product" | "category" | "order" | "user";
  result: "success" | "failed";
  metadata: Record<string, unknown>;
  created_at: string;
};
