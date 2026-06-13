"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Bell, CheckCheck, PackageOpen } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  fetchNotificationPreferences,
  fetchMyNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  updateNotificationPreferences,
} from "@/lib/api";
import { getToken } from "@/lib/auth";
import { ApiError, type PaginationMeta } from "@/types/api";
import type {
  NotificationPreferences,
  UpdateNotificationPreferencesInput,
  UserNotification,
} from "@/types/notification";

function formatDate(value?: string) {
  if (!value) {
    return "Unknown";
  }
  return new Intl.DateTimeFormat("vi-VN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function orderLink(notification: UserNotification) {
  const orderID = notification.metadata?.order_id;
  return typeof orderID === "string" && orderID ? `/orders/${orderID}` : null;
}

type PreferenceKey = keyof Pick<
  NotificationPreferences,
  | "order_updates_enabled"
  | "payment_updates_enabled"
  | "return_updates_enabled"
  | "promotion_enabled"
>;

const preferenceRows: Array<{
  key: PreferenceKey;
  label: string;
  description: string;
}> = [
  {
    key: "order_updates_enabled",
    label: "Order updates",
    description: "Checkout confirmations and fulfillment status changes.",
  },
  {
    key: "payment_updates_enabled",
    label: "Payment updates",
    description: "Payment status changes, refunds, and payment review events.",
  },
  {
    key: "return_updates_enabled",
    label: "Return updates",
    description: "Return/refund request approvals and rejections.",
  },
  {
    key: "promotion_enabled",
    label: "Promotions",
    description: "Future coupon, sale, and campaign notifications.",
  },
];

export default function NotificationsPage() {
  const router = useRouter();
  const [items, setItems] = useState<UserNotification[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | undefined>();
  const [preferences, setPreferences] = useState<NotificationPreferences | null>(null);
  const [page, setPage] = useState(1);
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [loading, setLoading] = useState(true);
  const [preferencesLoading, setPreferencesLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [savingPreference, setSavingPreference] = useState<PreferenceKey | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function loadNotifications(nextPage = page, nextUnreadOnly = unreadOnly) {
    setLoading(true);
    setError(null);
    try {
      const response = await fetchMyNotifications({
        page: nextPage,
        limit: 20,
        unread: nextUnreadOnly || undefined,
      });
      setItems(response.data ?? []);
      setMeta(response.meta);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/notifications");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to load notifications");
    } finally {
      setLoading(false);
    }
  }

  async function loadPreferences() {
    setPreferencesLoading(true);
    try {
      const response = await fetchNotificationPreferences();
      setPreferences(response.data ?? null);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/notifications");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to load notification preferences");
    } finally {
      setPreferencesLoading(false);
    }
  }

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login?redirect=/notifications");
      return;
    }
    void (async () => {
      await Promise.all([loadNotifications(1, unreadOnly), loadPreferences()]);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [router, unreadOnly]);

  const totalPages = useMemo(() => meta?.total_pages ?? meta?.total_page ?? 1, [meta]);
  const unreadCount = useMemo(() => items.filter((item) => !item.read_at).length, [items]);

  async function markOneRead(notification: UserNotification) {
    if (notification.read_at) {
      const href = orderLink(notification);
      if (href) {
        router.push(href);
      }
      return;
    }
    setBusy(true);
    setError(null);
    setSuccess(null);
    try {
      const response = await markNotificationRead(notification.id);
      const updated = response.data;
      if (updated) {
        setItems((current) => current.map((item) => (item.id === updated.id ? updated : item)));
      }
      const href = orderLink(notification);
      if (href) {
        router.push(href);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to mark notification read");
    } finally {
      setBusy(false);
    }
  }

  async function markAllRead() {
    setBusy(true);
    setError(null);
    setSuccess(null);
    try {
      const response = await markAllNotificationsRead();
      const updated = response.data?.updated ?? 0;
      setSuccess(`${updated} notifications marked as read.`);
      await loadNotifications(page, unreadOnly);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to mark notifications read");
    } finally {
      setBusy(false);
    }
  }

  async function togglePreference(key: PreferenceKey, checked: boolean) {
    setSavingPreference(key);
    setError(null);
    setSuccess(null);
    const previous = preferences;
    if (previous) {
      setPreferences({ ...previous, [key]: checked });
    }
    try {
      const input: UpdateNotificationPreferencesInput = { [key]: checked };
      const response = await updateNotificationPreferences(input);
      setPreferences(response.data ?? previous);
      setSuccess("Notification preferences saved.");
    } catch (err) {
      if (previous) {
        setPreferences(previous);
      }
      setError(err instanceof Error ? err.message : "Failed to update notification preference");
    } finally {
      setSavingPreference(null);
    }
  }

  async function goToPage(nextPage: number) {
    setPage(nextPage);
    await loadNotifications(nextPage, unreadOnly);
  }

  return (
    <div className="space-y-7">
      <section className="surface-card flex flex-col gap-4 rounded-[2rem] p-6 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="eyebrow">Account updates</p>
          <h1 className="mt-2 text-4xl font-semibold">Notifications</h1>
          <p className="mt-2 max-w-xl text-sm text-muted-foreground">
            Track order, payment, and return/refund events. Email delivery is intentionally a
            placeholder for now; events are stored here.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant={unreadOnly ? "default" : "outline"}
            onClick={() => {
              setPage(1);
              setUnreadOnly((current) => !current);
            }}
          >
            {unreadOnly ? "Showing unread" : "Show unread"}
          </Button>
          <Button type="button" variant="outline" onClick={markAllRead} disabled={busy || unreadCount === 0}>
            <CheckCheck className="size-4" /> Mark all read
          </Button>
        </div>
      </section>

      {error ? (
        <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}
      {success ? <p className="text-sm text-primary">{success}</p> : null}

      <Card className="surface-card rounded-[2rem]">
        <CardHeader>
          <p className="eyebrow">Settings</p>
          <CardTitle className="text-3xl">Notification Preferences</CardTitle>
        </CardHeader>
        <CardContent>
          {preferencesLoading ? (
            <div className="grid gap-3">
              {Array.from({ length: 4 }).map((_, index) => (
                <div key={index} className="h-20 animate-pulse rounded-2xl bg-muted/80" />
              ))}
            </div>
          ) : preferences ? (
            <div className="grid gap-3">
              {preferenceRows.map((item) => {
                const checked = Boolean(preferences[item.key]);
                const inputID = `preference-${item.key}`;
                return (
                  <label
                    key={item.key}
                    htmlFor={inputID}
                    className="flex cursor-pointer flex-col gap-3 rounded-2xl border border-border bg-card/70 p-4 transition hover:bg-muted/50 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <span>
                      <span className="block font-medium">{item.label}</span>
                      <span className="mt-1 block text-sm text-muted-foreground">{item.description}</span>
                    </span>
                    <span className="flex items-center gap-3">
                      <span className="text-sm text-muted-foreground">{checked ? "Enabled" : "Disabled"}</span>
                      <input
                        id={inputID}
                        type="checkbox"
                        className="size-5 rounded border-border accent-primary"
                        checked={checked}
                        disabled={savingPreference === item.key}
                        onChange={(event) => void togglePreference(item.key, event.target.checked)}
                      />
                    </span>
                  </label>
                );
              })}
            </div>
          ) : (
            <div className="state-panel">
              <p className="text-xl font-semibold">Preferences unavailable.</p>
              <p className="max-w-md text-sm text-muted-foreground">
                Refresh the page or sign in again to manage notification settings.
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {loading ? (
        <div className="grid gap-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="h-32 animate-pulse rounded-[1.5rem] bg-muted/80" />
          ))}
        </div>
      ) : null}

      {!loading && !error && items.length === 0 ? (
        <Card className="surface-card rounded-[2rem]">
          <CardContent className="state-panel">
            <Bell className="size-10 text-muted-foreground" />
            <p className="text-xl font-semibold">No notifications yet.</p>
            <p className="max-w-md text-sm text-muted-foreground">
              Checkout, order status, payment, and return/refund updates will appear here.
            </p>
            <Button asChild>
              <Link href="/products">Continue shopping</Link>
            </Button>
          </CardContent>
        </Card>
      ) : null}

      {!loading && items.length > 0 ? (
        <div className="grid gap-4">
          {items.map((notification) => {
            const href = orderLink(notification);
            return (
              <Card key={notification.id} className="surface-card rounded-[1.5rem]">
                <CardHeader>
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <CardTitle className="text-2xl">{notification.title}</CardTitle>
                        {!notification.read_at ? <Badge>Unread</Badge> : <Badge variant="outline">Read</Badge>}
                      </div>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {notification.type} / {formatDate(notification.created_at)}
                      </p>
                    </div>
                    {href ? (
                      <Button asChild variant="outline">
                        <Link href={href}>
                          <PackageOpen className="size-4" /> View order
                        </Link>
                      </Button>
                    ) : null}
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-sm leading-6 text-muted-foreground">{notification.message}</p>
                  {notification.metadata?.coupon_code ? (
                    <p className="rounded-2xl bg-muted/60 p-3 text-sm">
                      Coupon: {notification.metadata.coupon_code}
                    </p>
                  ) : null}
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      variant={notification.read_at ? "outline" : "default"}
                      disabled={busy}
                      onClick={() => markOneRead(notification)}
                    >
                      {notification.read_at ? "Open" : "Mark read"}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      ) : null}

      {meta && totalPages > 1 ? (
        <div className="surface-card flex items-center justify-between rounded-[1.5rem] p-4">
          <p className="text-sm text-muted-foreground">
            Page {meta.page} / {totalPages}
          </p>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={page <= 1 || loading}
              onClick={() => goToPage(page - 1)}
            >
              Previous
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={page >= totalPages || loading}
              onClick={() => goToPage(page + 1)}
            >
              Next
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
