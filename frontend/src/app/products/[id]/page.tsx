"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { useParams, useRouter } from "next/navigation";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { WishlistButton } from "@/components/wishlist/wishlist-button";
import {
  addToCart,
  addWishlistProduct,
  createReview,
  deleteReview,
  fetchMyOrders,
  fetchProductById,
  fetchProductReviews,
  fetchRatingSummary,
  fetchWishlist,
  removeWishlistProduct,
  updateReview,
} from "@/lib/api";
import { getMe, getToken } from "@/lib/auth";
import { ApiError } from "@/types/api";
import type { Order } from "@/types/order";
import type { Product } from "@/types/product";
import type { RatingSummary, Review } from "@/types/review";

const MAX_COMMENT_LENGTH = 1000;

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

function formatDate(value?: string) {
  if (!value) {
    return "Recently";
  }
  return new Intl.DateTimeFormat("vi-VN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function findPurchasedOrderId(orders: Order[], productId: string) {
  return orders.find(
    (order) =>
      ["paid", "shipping", "completed"].includes(order.status) &&
      order.items.some((item) => item.product_id === productId)
  )?.id;
}

function friendlyReviewError(err: unknown) {
  if (err instanceof ApiError) {
    if (err.status === 403) {
      return "You can review this product after buying it.";
    }
    if (err.status === 409) {
      return "You already reviewed this product. Update your existing review instead.";
    }
    if (err.status === 400) {
      return "Please check your rating and comment before submitting.";
    }
    if (err.status === 401) {
      return "Please login again to continue.";
    }
  }
  return "Could not save your review. Please try again.";
}

type RatingBreakdownProps = {
  summary: RatingSummary | null;
};

function RatingBreakdown({ summary }: RatingBreakdownProps) {
  const total = summary?.review_count ?? 0;
  const breakdown = summary?.rating_breakdown ?? {};

  return (
    <div className="space-y-2">
      {[5, 4, 3, 2, 1].map((rating) => {
        const count = breakdown[String(rating)] ?? 0;
        const percent = total > 0 ? Math.round((count / total) * 100) : 0;
        return (
          <div key={rating} className="grid grid-cols-[48px_1fr_44px] items-center gap-2 text-sm">
            <span>{rating} star</span>
            <div className="h-2 overflow-hidden rounded-full bg-muted">
              <div className="h-full rounded-full bg-primary" style={{ width: `${percent}%` }} />
            </div>
            <span className="text-right text-muted-foreground">{count}</span>
          </div>
        );
      })}
    </div>
  );
}

export default function ProductDetailPage() {
  const router = useRouter();
  const params = useParams();
  const productId = Array.isArray(params.id) ? params.id[0] : params.id;
  const [product, setProduct] = useState<Product | null>(null);
  const [summary, setSummary] = useState<RatingSummary | null>(null);
  const [reviews, setReviews] = useState<Review[]>([]);
  const [currentUserId, setCurrentUserId] = useState<string | null>(null);
  const [purchasedOrderId, setPurchasedOrderId] = useState<string | null>(null);
  const [wishlisted, setWishlisted] = useState(false);
  const [wishlistBusy, setWishlistBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [reviewsLoading, setReviewsLoading] = useState(true);
  const [viewerLoading, setViewerLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reviewsError, setReviewsError] = useState<string | null>(null);
  const [wishlistError, setWishlistError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const [reviewRating, setReviewRating] = useState<number | null>(null);
  const [reviewComment, setReviewComment] = useState<string | null>(null);
  const [reviewSaving, setReviewSaving] = useState(false);
  const [reviewDeleting, setReviewDeleting] = useState(false);
  const [reviewMessage, setReviewMessage] = useState<string | null>(null);
  const [reviewError, setReviewError] = useState<string | null>(null);

  const ownReview = useMemo(() => {
    if (!currentUserId) {
      return undefined;
    }
    return reviews.find((review) => review.user_id === currentUserId);
  }, [currentUserId, reviews]);

  const refreshReviewData = useCallback(async (id: string) => {
    setReviewsLoading(true);
    setReviewsError(null);
    try {
      const [summaryResponse, reviewsResponse] = await Promise.all([
        fetchRatingSummary(id),
        fetchProductReviews(id, { page: 1, limit: 20 }),
      ]);
      setSummary(summaryResponse.data ?? null);
      setReviews(reviewsResponse.data ?? []);
    } catch {
      setReviewsError("Could not load reviews right now.");
      setSummary(null);
      setReviews([]);
    } finally {
      setReviewsLoading(false);
    }
  }, []);

  const refreshViewerContext = useCallback(async (id: string) => {
    if (!getToken()) {
      setCurrentUserId(null);
      setPurchasedOrderId(null);
      setWishlisted(false);
      return;
    }

    setViewerLoading(true);
    try {
      const [meResponse, ordersResponse, wishlistResponse] = await Promise.all([
        getMe(),
        fetchMyOrders({ page: 1, limit: 100 }),
        fetchWishlist({ page: 1, limit: 100 }),
      ]);
      setCurrentUserId(meResponse.data?.user_id ?? null);
      setPurchasedOrderId(findPurchasedOrderId(ordersResponse.data ?? [], id) ?? null);
      setWishlisted((wishlistResponse.data ?? []).some((item) => item.product_id === id));
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setCurrentUserId(null);
        setPurchasedOrderId(null);
        setWishlisted(false);
      }
    } finally {
      setViewerLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    const id = productId;
    if (!id) {
      return;
    }
    const idForRequest = id;

    async function loadProduct() {
      setLoading(true);
      setError(null);
      try {
        const response = await fetchProductById(idForRequest);
        if (!cancelled) {
          setProduct(response.data ?? null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to fetch product");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadProduct();
    void Promise.resolve().then(() => refreshReviewData(idForRequest));
    void Promise.resolve().then(() => refreshViewerContext(idForRequest));

    return () => {
      cancelled = true;
    };
  }, [productId, refreshReviewData, refreshViewerContext]);

  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading product...</p>;
  }

  if (error) {
    return <p className="text-sm text-red-600">{error}</p>;
  }

  if (!product) {
    return <p className="text-sm text-muted-foreground">Product not found.</p>;
  }

  const displayedRating = summary?.average_rating ?? product.average_rating ?? 0;
  const displayedReviewCount = summary?.review_count ?? product.review_count ?? 0;
  const formRating = reviewRating ?? ownReview?.rating ?? 5;
  const formComment = reviewComment ?? ownReview?.comment ?? "";
  const canCreateReview = Boolean(currentUserId && purchasedOrderId);
  const canSubmitReview = Boolean(ownReview?.id || canCreateReview);

  async function onAddToCart() {
    if (!product) {
      return;
    }

    if (!getToken()) {
      router.push(`/login?redirect=/products/${product.id}`);
      return;
    }

    setAdding(true);
    setError(null);
    setSuccess(null);
    try {
      await addToCart(product.id, 1);
      setSuccess("Added to cart");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add to cart");
    } finally {
      setAdding(false);
    }
  }

  async function onToggleWishlist() {
    if (!product) {
      return;
    }

    if (!getToken()) {
      router.push(`/login?redirect=/products/${product.id}`);
      return;
    }

    const previous = wishlisted;
    setWishlistBusy(true);
    setWishlistError(null);
    setWishlisted(!previous);
    try {
      if (previous) {
        await removeWishlistProduct(product.id);
      } else {
        await addWishlistProduct(product.id);
      }
    } catch (err) {
      setWishlisted(previous);
      if (err instanceof ApiError && err.status === 401) {
        router.push(`/login?redirect=/products/${product.id}`);
        return;
      }
      setWishlistError(err instanceof Error ? err.message : "Could not update wishlist.");
    } finally {
      setWishlistBusy(false);
    }
  }

  async function onSubmitReview() {
    if (!product) {
      return;
    }

    if (!getToken()) {
      router.push(`/login?redirect=/products/${product.id}`);
      return;
    }
    if (formRating < 1 || formRating > 5) {
      setReviewError("Choose a rating from 1 to 5.");
      return;
    }
    if (formComment.length > MAX_COMMENT_LENGTH) {
      setReviewError(`Comment must be ${MAX_COMMENT_LENGTH} characters or fewer.`);
      return;
    }
    if (!canSubmitReview) {
      setReviewError("You can review this product after buying it.");
      return;
    }

    setReviewSaving(true);
    setReviewError(null);
    setReviewMessage(null);
    try {
      const payload = {
        rating: formRating,
        comment: formComment.trim() ? formComment.trim() : null,
      };
      if (ownReview?.id) {
        await updateReview(ownReview.id, payload);
        setReviewMessage("Review updated.");
      } else if (purchasedOrderId) {
        await createReview(product.id, { ...payload, order_id: purchasedOrderId });
        setReviewMessage("Review created.");
      }
      setReviewRating(null);
      setReviewComment(null);
      await refreshReviewData(product.id);
    } catch (err) {
      setReviewError(friendlyReviewError(err));
      if (err instanceof ApiError && err.status === 401) {
        router.push(`/login?redirect=/products/${product.id}`);
      }
    } finally {
      setReviewSaving(false);
    }
  }

  async function onDeleteReview() {
    if (!product) {
      return;
    }

    if (!ownReview?.id) {
      return;
    }
    setReviewDeleting(true);
    setReviewError(null);
    setReviewMessage(null);
    try {
      await deleteReview(ownReview.id);
      setReviewMessage("Review deleted.");
      setReviewRating(null);
      setReviewComment(null);
      await refreshReviewData(product.id);
    } catch (err) {
      setReviewError(friendlyReviewError(err));
    } finally {
      setReviewDeleting(false);
    }
  }

  return (
    <div className="space-y-8">
      <Card className="overflow-hidden">
        <div className="relative h-80 w-full sm:h-[28rem]">
          <Image
            src={product.image_url}
            alt={product.name}
            fill
            className="object-cover"
            sizes="100vw"
          />
          <div className="absolute right-4 top-4">
            <WishlistButton
              active={wishlisted}
              loading={wishlistBusy || viewerLoading}
              onToggle={onToggleWishlist}
              label
              className="bg-background/90"
            />
          </div>
        </div>
        <CardHeader>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <h1 className="font-heading text-3xl font-semibold leading-tight tracking-tight">
              {product.name}
            </h1>
            <div className="rounded-2xl border bg-muted/40 px-4 py-2 text-sm">
              <span className="font-semibold">{displayedRating.toFixed(1)}</span> / 5 ·{" "}
              {displayedReviewCount} review{displayedReviewCount === 1 ? "" : "s"}
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="text-xl font-semibold">{formatVND(product.price)}</div>
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary">{product.style}</Badge>
            <Badge variant="outline">{product.color}</Badge>
            <Badge variant={product.stock > 0 ? "outline" : "destructive"}>
              {product.stock > 0 ? `${product.stock} in stock` : "Sold out"}
            </Badge>
          </div>
          <p className="leading-relaxed">{product.description}</p>
          {error ? <p className="text-sm text-red-600">{error}</p> : null}
          {wishlistError ? <p className="text-sm text-red-600">{wishlistError}</p> : null}
          {success ? <p className="text-sm text-green-700">{success}</p> : null}
          <div className="flex flex-wrap gap-2">
            <Button onClick={onAddToCart} disabled={adding || product.stock <= 0}>
              {adding ? "Adding..." : "Add to Cart"}
            </Button>
            <Button variant="outline" asChild>
              <Link href="/cart">Go to Cart</Link>
            </Button>
            <Button variant="outline" asChild>
              <Link href="/products">Back to products</Link>
            </Button>
          </div>
        </CardContent>
      </Card>

      <section className="grid gap-6 lg:grid-cols-[360px_1fr]">
        <Card>
          <CardHeader>
            <CardTitle>Rating summary</CardTitle>
          </CardHeader>
          <CardContent className="space-y-5">
            {reviewsLoading ? (
              <p className="text-sm text-muted-foreground">Loading rating...</p>
            ) : (
              <>
                <div>
                  <p className="text-4xl font-semibold">{displayedRating.toFixed(1)}</p>
                  <p className="text-sm text-muted-foreground">
                    Based on {displayedReviewCount} review
                    {displayedReviewCount === 1 ? "" : "s"}
                  </p>
                </div>
                <RatingBreakdown summary={summary} />
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Write a review</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {!getToken() ? (
              <div className="rounded-xl border bg-muted/40 p-4">
                <p className="text-sm text-muted-foreground">Login to review this product.</p>
                <Button className="mt-3" asChild>
                  <Link href={`/login?redirect=/products/${product.id}`}>Login to review</Link>
                </Button>
              </div>
            ) : !ownReview && !purchasedOrderId ? (
              <p className="rounded-xl border bg-muted/40 p-4 text-sm text-muted-foreground">
                Reviews are available after checkout. If you already bought this product, refresh
                your orders and try again.
              </p>
            ) : (
              <>
                <div className="space-y-2">
                  <label htmlFor="rating" className="text-sm font-medium">
                    Rating
                  </label>
                  <select
                    id="rating"
                    className="h-9 w-full rounded-lg border border-input bg-background px-3 text-sm"
                    value={formRating}
                    onChange={(event) => setReviewRating(Number(event.target.value))}
                  >
                    {[5, 4, 3, 2, 1].map((rating) => (
                      <option key={rating} value={rating}>
                        {rating} star{rating === 1 ? "" : "s"}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="space-y-2">
                  <label htmlFor="comment" className="text-sm font-medium">
                    Comment
                  </label>
                  <textarea
                    id="comment"
                    className="min-h-28 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm"
                    value={formComment}
                    maxLength={MAX_COMMENT_LENGTH}
                    onChange={(event) => setReviewComment(event.target.value)}
                    placeholder="Share fit, fabric, sizing, or styling notes..."
                  />
                  <p className="text-xs text-muted-foreground">
                    {formComment.length}/{MAX_COMMENT_LENGTH} characters
                  </p>
                </div>
                {reviewError ? <p className="text-sm text-red-600">{reviewError}</p> : null}
                {reviewMessage ? <p className="text-sm text-green-700">{reviewMessage}</p> : null}
                <div className="flex flex-wrap gap-2">
                  <Button onClick={onSubmitReview} disabled={reviewSaving}>
                    {reviewSaving ? "Saving..." : ownReview ? "Update review" : "Submit review"}
                  </Button>
                  {ownReview?.id ? (
                    <Button variant="destructive" onClick={onDeleteReview} disabled={reviewDeleting}>
                      {reviewDeleting ? "Deleting..." : "Delete review"}
                    </Button>
                  ) : null}
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </section>

      <Card>
        <CardHeader>
          <CardTitle>Reviews</CardTitle>
        </CardHeader>
        <CardContent>
          {reviewsLoading ? (
            <p className="text-sm text-muted-foreground">Loading reviews...</p>
          ) : null}
          {reviewsError ? <p className="text-sm text-red-600">{reviewsError}</p> : null}
          {!reviewsLoading && !reviewsError && reviews.length === 0 ? (
            <p className="text-sm text-muted-foreground">No reviews yet.</p>
          ) : null}
          {!reviewsLoading && !reviewsError && reviews.length > 0 ? (
            <div className="space-y-4">
              {reviews.map((review) => (
                <div key={review.id} className="rounded-2xl border p-4">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <p className="font-medium">{review.rating ?? 0} / 5 stars</p>
                    <p className="text-xs text-muted-foreground">{formatDate(review.created_at)}</p>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-muted-foreground">
                    {review.comment || "No comment provided."}
                  </p>
                  {review.user_id === currentUserId ? (
                    <Badge className="mt-3" variant="secondary">
                      Your review
                    </Badge>
                  ) : null}
                </div>
              ))}
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
