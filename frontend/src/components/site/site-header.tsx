"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Heart,
  Menu,
  Package,
  Search,
  ShieldCheck,
  ShoppingBag,
  UserCircle,
} from "lucide-react";

import { ThemeToggle } from "@/components/theme-toggle";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { fetchCart, fetchWishlist } from "@/lib/api";
import { getMe, getStoredUser, getToken, logout, type AuthUser } from "@/lib/auth";

const publicLinks = [
  { href: "/products", label: "Products" },
  { href: "/about", label: "About" },
  { href: "/shipping", label: "Shipping" },
  { href: "/returns", label: "Returns" },
];

function BadgeCount({ value, label }: { value: number; label: string }) {
  if (value <= 0) {
    return null;
  }
  return (
    <span
      aria-label={label}
      className="absolute -right-2 -top-2 grid min-h-5 min-w-5 place-items-center rounded-full bg-primary px-1 text-[0.65rem] font-semibold text-primary-foreground shadow-soft"
    >
      {value > 99 ? "99+" : value}
    </span>
  );
}

function SearchForm({ onSubmitted }: { onSubmitted?: () => void }) {
  const router = useRouter();
  const [query, setQuery] = useState("");

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = query.trim();
    router.push(value ? `/products?q=${encodeURIComponent(value)}` : "/products");
    onSubmitted?.();
  }

  return (
    <form
      onSubmit={onSubmit}
      role="search"
      className="relative flex min-w-0 items-center rounded-full border border-border bg-card/90 px-3 py-2 shadow-sm transition focus-within:border-ring focus-within:ring-2 focus-within:ring-ring/20"
    >
      <Search className="mr-2 size-4 text-muted-foreground" aria-hidden="true" />
      <label htmlFor="site-search" className="sr-only">
        Search products
      </label>
      <input
        id="site-search"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Search styles, colors, outfits..."
        className="h-6 min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
      />
    </form>
  );
}

export function SiteHeader() {
  const router = useRouter();
  const [user, setUser] = useState<AuthUser | null>(null);
  const [cartCount, setCartCount] = useState(0);
  const [wishlistCount, setWishlistCount] = useState(0);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    let active = true;

    async function hydrateShell() {
      if (!getToken()) {
        if (active) {
          setUser(null);
          setCartCount(0);
          setWishlistCount(0);
        }
        return;
      }

      const storedUser = getStoredUser();
      if (storedUser && active) {
        setUser(storedUser);
      }

      try {
        const [meResponse, cartResponse, wishlistResponse] = await Promise.allSettled([
          getMe(),
          fetchCart(),
          fetchWishlist({ page: 1, limit: 100 }),
        ]);
        if (!active) {
          return;
        }
        if (meResponse.status === "fulfilled") {
          const me = meResponse.value.data;
          setUser({
            id: me?.user_id ?? storedUser?.id ?? "",
            email: storedUser?.email ?? "",
            full_name: storedUser?.full_name ?? "",
            role: (me?.role ?? storedUser?.role ?? "user") as AuthUser["role"],
            status: storedUser?.status,
          });
        }
        if (cartResponse.status === "fulfilled") {
          const items = cartResponse.value.data?.items ?? [];
          setCartCount(items.reduce((sum, item) => sum + (item.quantity ?? 0), 0));
        }
        if (wishlistResponse.status === "fulfilled") {
          setWishlistCount(wishlistResponse.value.meta?.total ?? wishlistResponse.value.data?.length ?? 0);
        }
      } catch {
        if (active) {
          setCartCount(0);
          setWishlistCount(0);
        }
      }
    }

    void hydrateShell();
    const refresh = () => void hydrateShell();
    window.addEventListener("focus", refresh);
    window.addEventListener("storage", refresh);
    return () => {
      active = false;
      window.removeEventListener("focus", refresh);
      window.removeEventListener("storage", refresh);
    };
  }, []);

  const isAdmin = user?.role === "admin";
  const initials = useMemo(() => {
    const name = user?.full_name || user?.email || "A";
    return name
      .split(" ")
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase())
      .join("") || "A";
  }, [user]);

  function onLogout() {
    logout();
    setUser(null);
    setCartCount(0);
    setWishlistCount(0);
    router.push("/login");
  }

  const navLinks = isAdmin ? [...publicLinks, { href: "/admin", label: "Admin" }] : publicLinks;

  return (
    <header className="sticky top-0 z-40 border-b border-border/70 bg-background/85 backdrop-blur-xl supports-[backdrop-filter]:bg-background/70">
      <div className="mx-auto flex max-w-7xl items-center gap-3 px-4 py-3 sm:px-6 lg:px-8">
        <Link href="/" className="group flex min-w-0 items-center gap-3" aria-label="Home">
          <span className="grid size-10 shrink-0 place-items-center rounded-2xl bg-primary text-sm font-bold text-primary-foreground shadow-soft transition group-hover:rotate-[-3deg]">
            XS
          </span>
          <span className="hidden min-w-0 sm:block">
            <span className="block font-heading text-lg font-semibold leading-tight">Xya-StyleMind</span>
            <span className="block truncate text-xs text-muted-foreground">AI fashion ecommerce</span>
          </span>
        </Link>

        <nav className="ml-3 hidden items-center gap-1 lg:flex" aria-label="Primary navigation">
          {navLinks.map((link) => (
            <Button key={link.href} variant="ghost" asChild>
              <Link href={link.href}>{link.label}</Link>
            </Button>
          ))}
        </nav>

        <div className="ml-auto hidden w-full max-w-sm lg:block">
          <SearchForm />
        </div>

        <div className="ml-auto flex items-center gap-2 lg:ml-2">
          <Button variant="ghost" size="icon" asChild className="relative" aria-label="Wishlist">
            <Link href="/wishlist">
              <Heart aria-hidden="true" />
              <BadgeCount value={wishlistCount} label={`${wishlistCount} wishlist items`} />
            </Link>
          </Button>
          <Button variant="ghost" size="icon" asChild className="relative" aria-label="Cart">
            <Link href="/cart">
              <ShoppingBag aria-hidden="true" />
              <BadgeCount value={cartCount} label={`${cartCount} cart items`} />
            </Link>
          </Button>
          <ThemeToggle />

          {user ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="hidden gap-2 rounded-full sm:inline-flex" aria-label="User menu">
                  <span className="grid size-6 place-items-center rounded-full bg-primary text-xs font-semibold text-primary-foreground">
                    {initials}
                  </span>
                  <span className="max-w-28 truncate">{user.full_name || user.email}</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuLabel>
                  <span className="block truncate">{user.email}</span>
                  <Badge variant={user.status === "disabled" ? "destructive" : "secondary"} className="mt-2 capitalize">
                    {user.role}
                  </Badge>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem asChild>
                  <Link href="/profile"><UserCircle /> Profile</Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link href="/orders"><Package /> Orders</Link>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <Link href="/wishlist"><Heart /> Wishlist</Link>
                </DropdownMenuItem>
                {isAdmin ? (
                  <DropdownMenuItem asChild>
                    <Link href="/admin"><ShieldCheck /> Admin</Link>
                  </DropdownMenuItem>
                ) : null}
                <DropdownMenuSeparator />
                <DropdownMenuItem onSelect={onLogout}>Logout</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <div className="hidden items-center gap-2 sm:flex">
              <Button variant="ghost" asChild>
                <Link href="/login">Login</Link>
              </Button>
              <Button asChild>
                <Link href="/register">Register</Link>
              </Button>
            </div>
          )}

          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            <SheetTrigger asChild>
              <Button variant="outline" size="icon" className="lg:hidden" aria-label="Open menu">
                <Menu aria-hidden="true" />
              </Button>
            </SheetTrigger>
            <SheetContent className="w-[86vw] gap-5 p-0">
              <SheetHeader className="border-b border-border p-5 text-left">
                <SheetTitle>Xya-StyleMind</SheetTitle>
                <SheetDescription>Shop smarter outfits from your phone.</SheetDescription>
              </SheetHeader>
              <div className="px-5">
                <SearchForm onSubmitted={() => setMobileOpen(false)} />
              </div>
              <nav className="grid gap-1 px-5" aria-label="Mobile navigation">
                {[...navLinks, { href: "/cart", label: "Cart" }, { href: "/wishlist", label: "Wishlist" }, { href: "/orders", label: "Orders" }, { href: "/profile", label: "Profile" }].map((link) => (
                  <SheetClose asChild key={link.href}>
                    <Link className="rounded-2xl px-4 py-3 text-sm font-medium transition hover:bg-muted" href={link.href}>
                      {link.label}
                    </Link>
                  </SheetClose>
                ))}
              </nav>
              <div className="mt-auto grid gap-2 border-t border-border p-5">
                {user ? (
                  <Button variant="outline" onClick={onLogout}>Logout</Button>
                ) : (
                  <>
                    <SheetClose asChild>
                      <Button asChild>
                        <Link href="/login">Login</Link>
                      </Button>
                    </SheetClose>
                    <SheetClose asChild>
                      <Button variant="outline" asChild>
                        <Link href="/register">Register</Link>
                      </Button>
                    </SheetClose>
                  </>
                )}
              </div>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  );
}
