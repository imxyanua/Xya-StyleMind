import Link from "next/link";
import Image from "next/image";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import type { Product } from "@/types/product";

type ProductCardProps = {
  product: Product;
};

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

export function ProductCard({ product }: ProductCardProps) {
  return (
    <Card className="h-full">
      <div className="relative h-64 w-full">
        <Image
          src={product.image_url}
          alt={product.name}
          fill
          className="object-cover"
          sizes="(max-width: 1024px) 100vw, 33vw"
        />
      </div>
      <CardHeader>
        <CardTitle className="line-clamp-2">{product.name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="text-lg font-semibold">{formatVND(product.price)}</div>
        <div className="flex flex-wrap gap-2">
          <Badge variant="secondary">{product.style}</Badge>
          <Badge variant="outline">{product.color}</Badge>
        </div>
        <p className="text-sm text-muted-foreground">Stock: {product.stock}</p>
      </CardContent>
      <CardFooter>
        <Link
          href={`/products/${product.id}`}
          className="text-sm font-medium text-primary underline-offset-4 hover:underline"
        >
          View details
        </Link>
      </CardFooter>
    </Card>
  );
}
