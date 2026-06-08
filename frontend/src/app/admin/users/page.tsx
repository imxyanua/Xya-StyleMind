"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  fetchAdminUser,
  fetchAdminUsers,
  updateAdminUserRole,
  updateAdminUserStatus,
  type AdminUserListParams,
} from "@/lib/api";
import type { PaginationMeta } from "@/types/api";
import type { AdminUser } from "@/types/user";

type UserRole = AdminUser["role"];
type UserStatus = AdminUser["status"];

type FilterState = {
  query: string;
  role: "" | UserRole;
  status: "" | UserStatus;
  sort: "newest" | "oldest";
};

const initialFilters: FilterState = {
  query: "",
  role: "",
  status: "",
  sort: "newest",
};

const roleTone: Record<UserRole, "secondary" | "outline"> = {
  admin: "secondary",
  user: "outline",
};

const statusTone: Record<UserStatus, "secondary" | "outline" | "destructive"> = {
  active: "secondary",
  disabled: "destructive",
};

function buildParams(filters: FilterState, page: number): AdminUserListParams {
  return {
    page,
    limit: 10,
    q: filters.query || undefined,
    role: filters.role || undefined,
    status: filters.status || undefined,
    sort: filters.sort,
  };
}

function formatDate(value?: string) {
  if (!value) {
    return "Unknown";
  }
  return new Intl.DateTimeFormat("vi-VN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function shortID(value?: string) {
  if (!value) {
    return "-";
  }
  return value.length > 18 ? `${value.slice(0, 8)}...${value.slice(-6)}` : value;
}

export default function AdminUsersPage() {
  const [filters, setFilters] = useState<FilterState>(initialFilters);
  const [appliedFilters, setAppliedFilters] = useState<FilterState>(initialFilters);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null);
  const [meta, setMeta] = useState<PaginationMeta | undefined>();
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [savingUserId, setSavingUserId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function loadUsers(nextPage = page, nextFilters = appliedFilters, selectFirst = false) {
    setLoading(true);
    setError(null);
    try {
      const response = await fetchAdminUsers(buildParams(nextFilters, nextPage));
      const nextUsers = response.data ?? [];
      setUsers(nextUsers);
      setMeta(response.meta);
      if (selectFirst && nextUsers[0]?.id) {
        await loadUserDetail(nextUsers[0].id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load users");
    } finally {
      setLoading(false);
    }
  }

  async function loadUserDetail(id: string) {
    setDetailLoading(true);
    setDetailError(null);
    try {
      const response = await fetchAdminUser(id);
      setSelectedUser(response.data ?? null);
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : "Failed to load user detail");
    } finally {
      setDetailLoading(false);
    }
  }

  useEffect(() => {
    let active = true;
    fetchAdminUsers(buildParams(initialFilters, 1))
      .then((response) => {
        if (!active) {
          return;
        }
        const nextUsers = response.data ?? [];
        setUsers(nextUsers);
        setMeta(response.meta);
      })
      .catch((err) => {
        if (active) {
          setError(err instanceof Error ? err.message : "Failed to load users");
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, []);

  const totalPages = useMemo(() => meta?.total_pages ?? meta?.total_page ?? 1, [meta]);

  function updateFilter<K extends keyof FilterState>(key: K, value: FilterState[K]) {
    setFilters((current) => ({ ...current, [key]: value }));
  }

  async function applyFilters(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSuccess(null);
    setSelectedUser(null);
    setAppliedFilters(filters);
    setPage(1);
    await loadUsers(1, filters, true);
  }

  async function resetFilters() {
    setFilters(initialFilters);
    setAppliedFilters(initialFilters);
    setPage(1);
    setSuccess(null);
    setSelectedUser(null);
    await loadUsers(1, initialFilters, true);
  }

  async function goToPage(nextPage: number) {
    setPage(nextPage);
    setSuccess(null);
    await loadUsers(nextPage, appliedFilters, true);
  }

  async function changeRole(user: AdminUser, nextRole: UserRole) {
    if (user.role === nextRole) {
      return;
    }
    const action = nextRole === "admin" ? "promote" : "demote";
    if (!window.confirm(`Confirm ${action} ${user.email} to ${nextRole}?`)) {
      return;
    }

    setSavingUserId(user.id);
    setError(null);
    setDetailError(null);
    setSuccess(null);
    try {
      const response = await updateAdminUserRole(user.id, { role: nextRole });
      const updated = response.data;
      setSuccess(`${user.email} is now ${nextRole}.`);
      await loadUsers(page, appliedFilters);
      if (updated?.id) {
        await loadUserDetail(updated.id);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to update user role";
      setDetailError(message);
    } finally {
      setSavingUserId(null);
    }
  }

  async function changeStatus(user: AdminUser, nextStatus: UserStatus) {
    if (user.status === nextStatus) {
      return;
    }
    if (
      nextStatus === "disabled" &&
      !window.confirm(`Confirm disable ${user.email}? This user will no longer be able to login.`)
    ) {
      return;
    }

    setSavingUserId(user.id);
    setError(null);
    setDetailError(null);
    setSuccess(null);
    try {
      const response = await updateAdminUserStatus(user.id, { status: nextStatus });
      const updated = response.data;
      setSuccess(`${user.email} is now ${nextStatus}.`);
      await loadUsers(page, appliedFilters);
      if (updated?.id) {
        await loadUserDetail(updated.id);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to update user status";
      setDetailError(message);
    } finally {
      setSavingUserId(null);
    }
  }

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-6">
        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <p className="eyebrow">Identity operations</p>
                <h1 className="mt-2 font-heading text-3xl font-semibold">Admin Users</h1>
                <p className="mt-2 max-w-2xl text-sm text-muted-foreground">
                  Review customer/admin accounts and promote or demote roles with persisted audit
                  trails.
                </p>
              </div>
              <Badge variant="secondary">{meta?.total ?? users.length} users</Badge>
            </div>
          </CardHeader>
          <CardContent>
            <form className="grid gap-3 lg:grid-cols-[1.4fr_150px_150px_150px]" onSubmit={applyFilters}>
              <div className="space-y-1.5">
                <label htmlFor="user-search" className="text-sm font-medium">
                  Search email/name
                </label>
                <Input
                  id="user-search"
                  value={filters.query}
                  onChange={(event) => updateFilter("query", event.target.value)}
                  placeholder="user@example.com or Linh"
                />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="role-filter" className="text-sm font-medium">
                  Role
                </label>
                <select
                  id="role-filter"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.role}
                  onChange={(event) => updateFilter("role", event.target.value as FilterState["role"])}
                >
                  <option value="">All</option>
                  <option value="user">User</option>
                  <option value="admin">Admin</option>
                </select>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="status-filter" className="text-sm font-medium">
                  Status
                </label>
                <select
                  id="status-filter"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.status}
                  onChange={(event) => updateFilter("status", event.target.value as FilterState["status"])}
                >
                  <option value="">All</option>
                  <option value="active">Active</option>
                  <option value="disabled">Disabled</option>
                </select>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="sort-filter" className="text-sm font-medium">
                  Sort
                </label>
                <select
                  id="sort-filter"
                  className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                  value={filters.sort}
                  onChange={(event) => updateFilter("sort", event.target.value as FilterState["sort"])}
                >
                  <option value="newest">Newest</option>
                  <option value="oldest">Oldest</option>
                </select>
              </div>
              <div className="flex gap-2 lg:col-span-4">
                <Button type="submit" disabled={loading}>
                  Apply filters
                </Button>
                <Button type="button" variant="outline" onClick={resetFilters} disabled={loading}>
                  Reset
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow">Safe account data</p>
              <h1 className="text-3xl font-semibold">User List</h1>
            </div>
            {meta ? (
              <p className="text-sm text-muted-foreground">
                Page {meta.page} of {totalPages || 1}
              </p>
            ) : null}
          </CardHeader>
          <CardContent>
            {error ? (
              <div className="state-panel border-destructive/30 bg-destructive/10 text-destructive">
                <p className="text-xl font-semibold">Could not load users.</p>
                <p className="text-sm">{error}</p>
              </div>
            ) : null}

            {loading ? (
              <div className="grid gap-3">
                {Array.from({ length: 5 }).map((_, index) => (
                  <div key={index} className="h-24 animate-pulse rounded-2xl bg-muted/80" />
                ))}
              </div>
            ) : null}

            {!loading && !error && users.length === 0 ? (
              <div className="state-panel">
                <p className="text-xl font-semibold">No users found.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Try clearing filters or registering a new account from the storefront.
                </p>
              </div>
            ) : null}

            {!loading && !error && users.length > 0 ? (
              <div className="overflow-hidden rounded-2xl border border-border">
                <div className="hidden grid-cols-[1.4fr_110px_110px_260px] gap-4 bg-muted/60 px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground lg:grid">
                  <span>User</span>
                  <span>Role</span>
                  <span>Status</span>
                  <span className="text-right">Actions</span>
                </div>
                <div className="divide-y divide-border">
                  {users.map((user) => (
                    <div
                      key={user.id}
                      className="grid gap-4 p-4 lg:grid-cols-[1.4fr_110px_110px_260px] lg:items-center"
                    >
                      <button
                        type="button"
                        className="min-w-0 text-left"
                        onClick={() => loadUserDetail(user.id)}
                      >
                        <p className="truncate font-heading text-xl font-semibold">{user.full_name}</p>
                        <p className="truncate text-sm text-muted-foreground">{user.email}</p>
                        <p className="mt-1 font-mono text-xs text-muted-foreground">{shortID(user.id)}</p>
                      </button>
                      <Badge variant={roleTone[user.role]} className="w-fit capitalize">
                        {user.role}
                      </Badge>
                      <Badge variant={statusTone[user.status]} className="w-fit capitalize">
                        {user.status}
                      </Badge>
                      <div className="flex flex-wrap justify-start gap-2 lg:justify-end">
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          disabled={savingUserId === user.id || user.role === "admin"}
                          onClick={() => changeRole(user, "admin")}
                        >
                          Promote
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="destructive"
                          disabled={savingUserId === user.id || user.role === "user"}
                          onClick={() => changeRole(user, "user")}
                        >
                          Demote
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          disabled={savingUserId === user.id || user.status === "active"}
                          onClick={() => changeStatus(user, "active")}
                        >
                          Enable
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="destructive"
                          disabled={savingUserId === user.id || user.status === "disabled"}
                          onClick={() => changeStatus(user, "disabled")}
                        >
                          Disable
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}

            {meta && totalPages > 1 ? (
              <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
                <Button
                  type="button"
                  variant="outline"
                  disabled={page <= 1 || loading}
                  onClick={() => goToPage(page - 1)}
                >
                  Previous
                </Button>
                <p className="text-sm text-muted-foreground">
                  Showing {users.length} of {meta.total}
                </p>
                <Button
                  type="button"
                  variant="outline"
                  disabled={page >= totalPages || loading}
                  onClick={() => goToPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>

      <div className="space-y-6 xl:sticky xl:top-28 xl:self-start">
        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <p className="eyebrow">Selected account</p>
            <h2 className="font-heading text-3xl font-semibold">User Detail</h2>
          </CardHeader>
          <CardContent>
            {detailLoading ? <div className="h-44 animate-pulse rounded-2xl bg-muted/80" /> : null}
            {detailError ? (
              <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {detailError}
              </p>
            ) : null}
            {success ? <p className="mb-3 text-sm text-primary">{success}</p> : null}
            {!detailLoading && !selectedUser ? (
              <div className="state-panel">
                <p className="text-xl font-semibold">No user selected.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Pick a user from the table to inspect profile metadata and role state.
                </p>
              </div>
            ) : null}
            {!detailLoading && selectedUser ? (
              <div className="space-y-5">
                <div className="rounded-2xl border border-border bg-card/70 p-4">
                  <p className="text-xs uppercase tracking-[0.22em] text-muted-foreground">User ID</p>
                  <p className="mt-2 break-all font-mono text-sm">{selectedUser.id}</p>
                </div>
                <div>
                  <h2 className="font-heading text-3xl font-semibold">{selectedUser.full_name}</h2>
                  <p className="mt-1 break-all text-sm text-muted-foreground">{selectedUser.email}</p>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Badge variant={roleTone[selectedUser.role]} className="capitalize">
                      {selectedUser.role}
                    </Badge>
                    <Badge variant={statusTone[selectedUser.status]} className="capitalize">
                      {selectedUser.status}
                    </Badge>
                  </div>
                </div>
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="rounded-2xl bg-muted/60 p-4">
                    <p className="text-sm text-muted-foreground">Created</p>
                    <p className="mt-2 text-sm font-medium">{formatDate(selectedUser.created_at)}</p>
                  </div>
                  <div className="rounded-2xl bg-muted/60 p-4">
                    <p className="text-sm text-muted-foreground">Updated</p>
                    <p className="mt-2 text-sm font-medium">{formatDate(selectedUser.updated_at)}</p>
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    disabled={savingUserId === selectedUser.id || selectedUser.role === "admin"}
                    onClick={() => changeRole(selectedUser, "admin")}
                  >
                    Promote to admin
                  </Button>
                  <Button
                    type="button"
                    variant="destructive"
                    disabled={savingUserId === selectedUser.id || selectedUser.role === "user"}
                    onClick={() => changeRole(selectedUser, "user")}
                  >
                    Demote to user
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    disabled={savingUserId === selectedUser.id || selectedUser.status === "active"}
                    onClick={() => changeStatus(selectedUser, "active")}
                  >
                    Enable account
                  </Button>
                  <Button
                    type="button"
                    variant="destructive"
                    disabled={savingUserId === selectedUser.id || selectedUser.status === "disabled"}
                    onClick={() => changeStatus(selectedUser, "disabled")}
                  >
                    Disable account
                  </Button>
                </div>
                <p className="text-xs leading-5 text-muted-foreground">
                  Role changes are persisted to the admin audit log. Password hashes and tokens are
                  never returned by this screen.
                </p>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
