import Link from "next/link";
import { Mail, MapPin, MessageCircle, Sparkles } from "lucide-react";

const footerGroups = [
  {
    title: "Shop",
    links: [
      { href: "/products", label: "All products" },
      { href: "/products?sort=newest", label: "New arrivals" },
      { href: "/products?sort=rating_desc", label: "Top rated" },
      { href: "/wishlist", label: "Wishlist" },
    ],
  },
  {
    title: "Support",
    links: [
      { href: "/shipping", label: "Shipping" },
      { href: "/returns", label: "Returns" },
      { href: "/contact", label: "Contact" },
      { href: "/orders", label: "Order history" },
    ],
  },
  {
    title: "Company",
    links: [
      { href: "/about", label: "About" },
      { href: "/privacy", label: "Privacy" },
      { href: "/terms", label: "Terms" },
    ],
  },
];

export function SiteFooter() {
  return (
    <footer className="mt-14 border-t border-border/80 bg-card/55">
      <div className="mx-auto grid max-w-7xl gap-10 px-4 py-10 sm:px-6 lg:grid-cols-[1.1fr_1.4fr] lg:px-8">
        <div className="space-y-5">
          <Link href="/" className="inline-flex items-center gap-3" aria-label="Home">
            <span className="grid size-11 place-items-center rounded-2xl bg-primary text-sm font-bold text-primary-foreground shadow-soft">
              XS
            </span>
            <span>
              <span className="block font-heading text-xl font-semibold">Xya-StyleMind</span>
              <span className="block text-xs text-muted-foreground">AI-powered fashion ecommerce</span>
            </span>
          </Link>
          <p className="max-w-md text-sm leading-6 text-muted-foreground">
            A modern fashion commerce demo built around curated products, verified reviews,
            wishlist-first discovery, and an AI styling roadmap for smarter outfit decisions.
          </p>
          <div className="flex flex-wrap gap-3 text-sm text-muted-foreground">
            <span className="inline-flex items-center gap-2 rounded-full border border-border bg-background/70 px-3 py-2">
              <Mail className="size-4" /> hello@xya-stylemind.demo
            </span>
            <span className="inline-flex items-center gap-2 rounded-full border border-border bg-background/70 px-3 py-2">
              <MapPin className="size-4" /> Vietnam / Global demo
            </span>
            <span className="inline-flex items-center gap-2 rounded-full border border-border bg-background/70 px-3 py-2">
              <MessageCircle className="size-4" /> @xyastylemind
            </span>
          </div>
        </div>

        <div className="grid gap-7 sm:grid-cols-3">
          {footerGroups.map((group) => (
            <div key={group.title} className="space-y-3">
              <h2 className="text-sm font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                {group.title}
              </h2>
              <ul className="space-y-2 text-sm">
                {group.links.map((link) => (
                  <li key={link.href}>
                    <Link href={link.href} className="text-muted-foreground transition hover:text-foreground">
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
      <div className="border-t border-border/80">
        <div className="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-5 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
          <p>© 2026 Xya-StyleMind. Demo ecommerce experience for public portfolio use.</p>
          <p className="inline-flex items-center gap-2">
            <Sparkles className="size-3.5" /> Built with Next.js, Go, PostgreSQL, and AI-ready architecture.
          </p>
        </div>
      </div>
    </footer>
  );
}
