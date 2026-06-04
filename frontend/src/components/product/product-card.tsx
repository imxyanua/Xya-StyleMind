import Link from "next/link";
import Image from "next/image";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { WishlistButton } from "@/components/wishlist/wishlist-button";
import type { Product } from "@/types/product";

type ProductCardProps = {
  product: Product;
  categoryName?: string;
  wishlisted?: boolean;
  wishlistLoading?: boolean;
  onToggleWishlist?: (productId: string) => void;
};

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

export function ProductCard({
  product,
  categoryName,
  wishlisted = false,
  wishlistLoading = false,
  onToggleWishlist,
}: ProductCardProps) {
  const hasRating = product.review_count > 0;
  const inStock = product.stock > 0;

  return (
    <Card className="h-full overflow-hidden">
      <div className="relative h-64 w-full">
        <Image
          src={product.image_url}
          alt={product.name}
          fill
          className="object-cover"
          sizes="(max-width: 1024px) 100vw, 33vw"
        />
        <div className="absolute left-3 top-3 flex flex-wrap gap-2">
          <Badge variant={inStock ? "secondary" : "destructive"}>
            {inStock ? "In stock" : "Sold out"}
          </Badge>
          {hasRating ? (
            <Badge variant="outline" className="bg-background/90">
              {product.average_rating.toFixed(1)} / 5 ({product.review_count})
            </Badge>
          ) : null}
        </div>
        {onToggleWishlist ? (
          <WishlistButton
            active={wishlisted}
            loading={wishlistLoading}
            onToggle={() => onToggleWishlist(product.id)}
            className="absolute right-3 top-3 bg-background/90"
          />
        ) : null}
      </div>
      <CardHeader className="space-y-2">
        <CardTitle className="line-clamp-2">{product.name}</CardTitle>
        {categoryName ? (
          <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">{categoryName}</p>
        ) : null}
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="text-lg font-semibold">{formatVND(product.price)}</div>
        <div className="flex flex-wrap gap-2">
          <Badge variant="secondary">{product.style}</Badge>
          <Badge variant="outline">{product.color}</Badge>
        </div>
        <p className="text-sm text-muted-foreground">
          {inStock ? `${product.stock} items available` : "Currently out of stock"}
        </p>
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
