"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchAdminUser, fetchMyOrders, fetchProductReviews, fetchWishlist } from "@/lib/api";
import { getMe, getStoredUser, getToken, logout, type AuthUser } from "@/lib/auth";
import { ApiError } from "@/types/api";
import type { Order } from "@/types/order";

type ProfileStats = {
  recentOrders: Order[];
  wishlistCount: number;
  reviewCount: number;
};

function formatVND(value?: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
    maximumFractionDigits: 0,
  }).format(value ?? 0);
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
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

export default function ProfilePage() {
  const router = useRouter();
  const [profile, setProfile] = useState<AuthUser | null>(null);
  const [stats, setStats] = useState<ProfileStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login?redirect=/profile");
      return;
    }

    let cancelled = false;
    async function loadProfile() {
      setLoading(true);
      setError(null);
      try {
        const meResponse = await getMe();
        const me = meResponse.data;
        if (!me?.user_id) {
          throw new Error("Profile session is missing user id");
        }

        const cached = getStoredUser();
        let nextProfile: AuthUser = {
          id: me.user_id,
          email: cached?.email ?? "Email available after next login",
          full_name: cached?.full_name ?? "StyleMind member",
          role: me.role,
          status: cached?.status ?? "active",
        };

        if (me.role === "admin") {
          try {
            const adminUser = await fetchAdminUser(me.user_id);
            if (adminUser.data) {
              nextProfile = {
                id: adminUser.data.id,
                email: adminUser.data.email,
                full_name: adminUser.data.full_name,
                role: adminUser.data.role,
                status: adminUser.data.status,
              };
            }
          } catch {
            // Admin detail hydration is best-effort; /auth/me still proves the session.
          }
        }

        const [ordersResponse, wishlistResponse] = await Promise.all([
          fetchMyOrders({ page: 1, limit: 5 }),
          fetchWishlist({ page: 1, limit: 100 }),
        ]);
        const recentOrders = ordersResponse.data ?? [];
        const wishlistCount = wishlistResponse.meta?.total ?? wishlistResponse.data?.length ?? 0;
        const purchasedProductIds = Array.from(
          new Set(recentOrders.flatMap((order) => order.items.map((item) => item.product_id)).filter(Boolean))
        ).slice(0, 8);

        const reviewLists = await Promise.all(
          purchasedProductIds.map((productId) =>
            fetchProductReviews(productId, { page: 1, limit: 100 })
              .then((response) => response.data ?? [])
              .catch(() => [])
          )
        );
        const reviewCount = reviewLists.flat().filter((review) => review.user_id === me.user_id).length;

        if (!cancelled) {
          setProfile(nextProfile);
          setStats({
            recentOrders,
            wishlistCount,
            reviewCount,
          });
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        if (err instanceof ApiError && err.status === 401) {
          logout();
          router.replace("/login?redirect=/profile");
          return;
        }
        setError(err instanceof Error ? err.message : "Failed to load profile");
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadProfile();
    return () => {
      cancelled = true;
    };
  }, [router]);

  const totalRecentSpend = useMemo(() => {
    return stats?.recentOrders.reduce((sum, order) => sum + (order.total_amount ?? 0), 0) ?? 0;
  }, [stats]);

  function onLogout() {
    logout();
    router.push("/login");
  }

  if (loading) {
    return (
      <div className="grid gap-5 lg:grid-cols-[0.8fr_1.2fr]">
        <div className="h-80 animate-pulse rounded-[2rem] bg-muted/80" />
        <div className="grid gap-4">
          {Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="h-28 animate-pulse rounded-[1.5rem] bg-muted/80" />
          ))}
        </div>
      </div>
    );
  }

  if (error || !profile || !stats) {
    return (
      <Card className="surface-card rounded-[2rem]">
        <CardContent className="state-panel">
          <p className="text-xl font-semibold">Could not load your profile.</p>
          <p className="max-w-md text-sm text-muted-foreground">{error ?? "No profile data returned."}</p>
          <Button onClick={() => window.location.reload()}>Retry</Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-7">
      <section className="surface-card overflow-hidden rounded-[2rem] p-6 sm:p-8">
        <div className="grid gap-6 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] lg:items-end">
          <div className="min-w-0">
            <p className="eyebrow">Member profile</p>
            <h1 className="mt-2 break-words text-4xl font-semibold sm:text-5xl">{profile.full_name}</h1>
            <p className="mt-3 break-all text-sm text-muted-foreground">{profile.email}</p>
            <div className="mt-4 flex flex-wrap gap-2">
              <Badge variant={profile.role === "admin" ? "secondary" : "outline"}>{profile.role}</Badge>
              <Badge variant={profile.status === "disabled" ? "destructive" : "secondary"}>
                {profile.status ?? "active"}
              </Badge>
              <Badge variant="outline" className="font-mono">{shortID(profile.id)}</Badge>
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="rounded-3xl bg-muted/60 p-4">
              <p className="text-sm text-muted-foreground">Wishlist</p>
              <p className="mt-2 font-heading text-3xl font-semibold">{stats.wishlistCount}</p>
            </div>
            <div className="rounded-3xl bg-muted/60 p-4">
              <p className="text-sm text-muted-foreground">Reviews</p>
              <p className="mt-2 font-heading text-3xl font-semibold">{stats.reviewCount}</p>
            </div>
            <div className="rounded-3xl bg-muted/60 p-4">
              <p className="text-sm text-muted-foreground">Recent spend</p>
              <p className="mt-2 font-heading text-xl font-semibold">{formatVND(totalRecentSpend)}</p>
            </div>
          </div>
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
        <Card id="orders" className="surface-card rounded-[1.75rem]">
          <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="eyebrow">Order pulse</p>
              <CardTitle className="text-3xl">Recent Orders</CardTitle>
            </div>
            <Button asChild variant="outline">
              <Link href="/orders">View all</Link>
            </Button>
          </CardHeader>
          <CardContent className="space-y-3">
            {stats.recentOrders.length === 0 ? (
              <div className="state-panel min-h-52">
                <p className="text-xl font-semibold">No orders yet.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Your checkout history will appear here once you place an order.
                </p>
                <Button asChild>
                  <Link href="/products">Browse products</Link>
                </Button>
              </div>
            ) : (
              stats.recentOrders.map((order) => (
                <Link
                  key={order.id}
                  href="/orders"
                  className="grid gap-3 rounded-3xl border border-border bg-card/70 p-4 transition hover:bg-muted/60 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
                >
                  <div className="min-w-0">
                    <p className="font-mono text-xs text-muted-foreground">{shortID(order.id)}</p>
                    <p className="mt-1 text-sm text-muted-foreground">{formatDate(order.created_at)}</p>
                  </div>
                  <div className="flex flex-wrap items-center gap-3 sm:justify-end">
                    <Badge variant="outline">{order.status}</Badge>
                    <span className="font-heading text-xl font-semibold">{formatVND(order.total_amount)}</span>
                  </div>
                </Link>
              ))
            )}
          </CardContent>
        </Card>

        <Card className="surface-card h-fit rounded-[1.75rem]">
          <CardHeader>
            <p className="eyebrow">Quick actions</p>
            <CardTitle className="text-3xl">Account</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3">
            <div className="rounded-3xl border border-border bg-muted/50 p-4">
              <p className="text-sm font-medium">Account status</p>
              <p className="mt-2 text-sm leading-6 text-muted-foreground">
                Your account is currently{" "}
                <span className="font-semibold text-foreground">{profile.status ?? "active"}</span>.
                Protected shopping actions are available while the account remains active.
              </p>
            </div>
            <Button asChild variant="outline">
              <Link href="/orders">Open orders</Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="/wishlist">Open wishlist</Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="/profile/addresses">Manage addresses</Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="/cart">Open cart</Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="#reviews">Review activity</Link>
            </Button>
            {profile.role === "admin" ? (
              <Button asChild variant="outline">
                <Link href="/admin">Admin dashboard</Link>
              </Button>
            ) : null}
            <Button variant="destructive" onClick={onLogout}>
              Logout
            </Button>
          </CardContent>
        </Card>
      </div>

      <Card id="reviews" className="surface-card rounded-[1.75rem]">
        <CardHeader>
          <p className="eyebrow">Review center</p>
          <CardTitle className="text-3xl">Your product voice</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <p className="max-w-2xl text-sm leading-7 text-muted-foreground">
            You have {stats.reviewCount} review{stats.reviewCount === 1 ? "" : "s"} detected from
            recently purchased products. Open your orders to revisit products and update verified reviews.
          </p>
          <Button asChild>
            <Link href="/orders">Go to Orders</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
