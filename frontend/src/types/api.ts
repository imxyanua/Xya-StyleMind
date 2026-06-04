import type { components } from "@/types/openapi";

export type PaginationMeta = components["schemas"]["PaginationMeta"];

export type ApiResponse<T> = {
  success: boolean;
  message: string;
  data?: T;
  meta?: PaginationMeta;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}
