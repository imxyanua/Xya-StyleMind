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
    return <p className="text-sm text-muted-foreground">Checking admin access...</p>;
  }

  if (!allowed) {
    return (
      <Card>
        <CardContent className="py-8">
          <p className="text-sm text-muted-foreground">{message ?? "Access denied."}</p>
        </CardContent>
      </Card>
    );
  }

  return <>{children}</>;
}
