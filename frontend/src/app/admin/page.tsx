import Link from "next/link";

import { Button } from "@/components/ui/button";
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
    <div className="grid gap-5 md:grid-cols-3">
      {adminLinks.map((item, index) => (
        <Card
          key={item.href}
          className="surface-card group h-full rounded-[1.75rem] transition hover:-translate-y-1"
        >
          <CardHeader className="space-y-4">
            <span className="grid size-11 place-items-center rounded-2xl bg-secondary text-sm font-semibold">
              0{index + 1}
            </span>
            <div>
              <CardTitle className="text-2xl">{item.title}</CardTitle>
              <CardDescription className="mt-2 leading-6">{item.description}</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <Button asChild variant="outline" className="w-full">
              <Link href={item.href}>Open</Link>
            </Button>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
