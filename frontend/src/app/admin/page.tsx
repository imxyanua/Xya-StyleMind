import Link from "next/link";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const adminLinks = [
  {
    href: "/admin/categories",
    title: "Categories",
    description: "Create product categories for catalog organization.",
  },
  {
    href: "/admin/products",
    title: "Products",
    description: "Create, update, and remove catalog products.",
  },
  {
    href: "/admin/orders",
    title: "Orders",
    description: "Update order status by order ID.",
  },
];

export default function AdminPage() {
  return (
    <div className="grid gap-4 md:grid-cols-3">
      {adminLinks.map((item) => (
        <Link key={item.href} href={item.href}>
          <Card className="h-full transition hover:ring-foreground/20">
            <CardHeader>
              <CardTitle>{item.title}</CardTitle>
              <CardDescription>{item.description}</CardDescription>
            </CardHeader>
            <CardContent>
              <span className="text-sm font-medium">Open</span>
            </CardContent>
          </Card>
        </Link>
      ))}
    </div>
  );
}
