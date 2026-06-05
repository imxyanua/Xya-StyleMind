"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { Card, CardContent } from "@/components/ui/card";
import { getMe, getToken } from "@/lib/auth";

type AdminGuardProps = {
  children: React.ReactNode;
};

export function AdminGuard({ children }: AdminGuardProps) {
  const router = useRouter();
  const [allowed, setAllowed] = useState(false);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function checkAdmin() {
      if (!getToken()) {
        router.replace("/login?redirect=/admin");
        return;
      }

      try {
        const res = await getMe();
        if (cancelled) {
          return;
        }

        if (res.data?.role !== "admin") {
          setMessage("Admin access required.");
          setAllowed(false);
          return;
        }

        setAllowed(true);
      } catch {
        if (!cancelled) {
          router.replace("/login?redirect=/admin");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    checkAdmin();
    return () => {
      cancelled = true;
    };
  }, [router]);

  if (loading) {
    return (
      <div className="surface-card rounded-[2rem] p-8">
        <p className="eyebrow">Admin security</p>
        <div className="mt-4 h-24 animate-pulse rounded-2xl bg-muted/80" />
        <p className="mt-4 text-sm text-muted-foreground">Checking admin access...</p>
      </div>
    );
  }

  if (!allowed) {
    return (
      <Card className="surface-card">
        <CardContent className="state-panel">
          <p className="text-xl font-semibold">Access denied</p>
          <p className="text-sm text-muted-foreground">{message ?? "Access denied."}</p>
        </CardContent>
      </Card>
    );
  }

  return <>{children}</>;
}
