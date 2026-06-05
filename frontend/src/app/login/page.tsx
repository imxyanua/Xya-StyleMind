"use client";

import { FormEvent, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { login } from "@/lib/auth";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError(null);

    try {
      await login({ email, password });
      const searchParams = new URLSearchParams(window.location.search);
      const redirect = searchParams.get("redirect") || "/products";
      router.push(redirect);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Login failed";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto grid max-w-5xl overflow-hidden rounded-[2rem] border border-border bg-card shadow-soft lg:grid-cols-[0.9fr_1.1fr]">
      <div className="hidden bg-[radial-gradient(circle_at_30%_20%,rgba(255,255,255,0.28),transparent_16rem),linear-gradient(145deg,#273021,#11140f)] p-10 text-primary-foreground lg:flex lg:flex-col lg:justify-between">
        <p className="eyebrow text-primary-foreground/60">Member access</p>
        <div>
          <h1 className="text-5xl font-semibold leading-none">Welcome back to the edit.</h1>
          <p className="mt-4 text-sm leading-6 text-primary-foreground/70">
            Login to sync wishlist, cart, checkout, and review permissions with the backend.
          </p>
        </div>
      </div>
      <Card className="border-0 bg-transparent shadow-none">
        <CardHeader className="space-y-2 p-6 sm:p-8">
          <CardTitle className="text-4xl">Login</CardTitle>
          <CardDescription>Use your account to access protected shopping actions.</CardDescription>
        </CardHeader>
        <CardContent className="p-6 pt-0 sm:p-8 sm:pt-0">
          <form className="space-y-4" onSubmit={onSubmit}>
            <div className="space-y-1">
              <label htmlFor="email" className="text-sm font-medium">
                Email
              </label>
              <Input
                id="email"
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <label htmlFor="password" className="text-sm font-medium">
                Password
              </label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
              />
            </div>
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Logging in..." : "Login"}
            </Button>
            <p className="text-center text-sm text-muted-foreground">
              New here?{" "}
              <Link href="/register" className="font-medium text-primary hover:underline">
                Create an account
              </Link>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
