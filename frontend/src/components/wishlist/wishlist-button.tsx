import { Heart } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type WishlistButtonProps = {
  active: boolean;
  loading?: boolean;
  onToggle: () => void;
  label?: boolean;
  className?: string;
};

export function WishlistButton({
  active,
  loading = false,
  onToggle,
  label = false,
  className,
}: WishlistButtonProps) {
  return (
    <Button
      type="button"
      variant={active ? "secondary" : "outline"}
      size={label ? "default" : "icon"}
      onClick={onToggle}
      disabled={loading}
      aria-pressed={active}
      aria-label={active ? "Remove from wishlist" : "Add to wishlist"}
      className={className}
    >
      <Heart className={cn("size-4", active ? "fill-current" : "")} />
      {label ? <span>{loading ? "Saving..." : active ? "Wishlisted" : "Wishlist"}</span> : null}
    </Button>
  );
}
