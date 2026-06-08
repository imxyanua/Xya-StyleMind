import Link from "next/link";
import Image from "next/image";
import { Star } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { WishlistButton } from "@/components/wishlist/wishlist-button";
import { PRODUCT_IMAGE_BLUR } from "@/lib/images";
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
    <Card className="group h-full overflow-hidden border-border/80 bg-card/90 shadow-product transition duration-300 hover:-translate-y-1 hover:shadow-soft">
      <div className="relative h-72 w-full overflow-hidden bg-muted">
        <Image
          src={product.image_url}
          alt={product.name}
          fill
          className="object-cover transition duration-700 group-hover:scale-105"
          sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw"
          placeholder="blur"
          blurDataURL={PRODUCT_IMAGE_BLUR}
        />
        <div className="absolute inset-x-0 top-0 flex items-start justify-between gap-3 bg-gradient-to-b from-black/45 to-transparent p-3">
          <Badge variant={inStock ? "secondary" : "destructive"} className="bg-card/95">
            {inStock ? "In stock" : "Sold out"}
          </Badge>
          {hasRating ? (
            <Badge variant="outline" className="gap-1 border-white/50 bg-card/95">
              <Star className="size-3 fill-current" />
              {product.average_rating.toFixed(1)} ({product.review_count})
            </Badge>
          ) : null}
        </div>
        {onToggleWishlist ? (
          <WishlistButton
            active={wishlisted}
            loading={wishlistLoading}
            onToggle={() => onToggleWishlist(product.id)}
            className="absolute bottom-3 right-3 bg-card/95 shadow-soft"
          />
        ) : null}
      </div>
      <CardHeader className="space-y-2 pb-0">
        <CardTitle className="line-clamp-2 text-xl">{product.name}</CardTitle>
        {categoryName ? (
          <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">{categoryName}</p>
        ) : null}
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div className="font-heading text-2xl font-semibold">{formatVND(product.price)}</div>
          <p className="text-xs text-muted-foreground">
            {inStock ? `${product.stock} left` : "Sold out"}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Badge variant="secondary">{product.style}</Badge>
          <Badge variant="outline">{product.color}</Badge>
        </div>
        <p className="min-h-5 text-sm text-muted-foreground">
          {inStock ? `${product.stock} items available` : "Currently out of stock"}
        </p>
      </CardContent>
      <CardFooter className="bg-muted/40">
        <Link
          href={`/products/${product.id}`}
          className="inline-flex w-full items-center justify-center rounded-full border border-border bg-card px-4 py-2 text-sm font-medium text-primary transition hover:bg-primary hover:text-primary-foreground"
        >
          View details
        </Link>
      </CardFooter>
    </Card>
  );
}
