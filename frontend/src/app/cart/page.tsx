"use client";

import { type FormEvent, useEffect, useMemo, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { MapPin, ShieldCheck, Truck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  checkoutOrder,
  fetchCart,
  fetchMyAddresses,
  removeCartItem,
  updateCartItem,
  type CheckoutInput,
  type UserAddress,
} from "@/lib/api";
import { getToken } from "@/lib/auth";
import { PRODUCT_IMAGE_BLUR } from "@/lib/images";
import { ApiError } from "@/types/api";
import type { Cart } from "@/types/cart";

function formatVND(value: number) {
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: "VND",
  }).format(value);
}

function checkoutFieldsFromAddress(address: UserAddress): Partial<CheckoutInput> {
  return {
    recipient_name: address.recipient_name ?? "",
    phone: address.phone ?? "",
    address_line: address.address_line ?? "",
    city: address.city ?? "",
    district: address.district ?? "",
    note: address.note ?? "",
  };
}

export default function CartPage() {
  const router = useRouter();
  const [cart, setCart] = useState<Cart | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [checkoutLoading, setCheckoutLoading] = useState(false);
  const [itemBusyId, setItemBusyId] = useState<string | null>(null);
  const [quantityInputs, setQuantityInputs] = useState<Record<string, number>>({});
  const [addresses, setAddresses] = useState<UserAddress[]>([]);
  const [addressesLoading, setAddressesLoading] = useState(false);
  const [addressError, setAddressError] = useState<string | null>(null);
  const [selectedAddressId, setSelectedAddressId] = useState("");
  const [checkoutForm, setCheckoutForm] = useState<CheckoutInput>({
    recipient_name: "",
    phone: "",
    address_line: "",
    city: "",
    district: "",
    note: "",
    shipping_method: "standard",
    payment_method: "cod",
  });

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login?redirect=/cart");
      return;
    }

    let cancelled = false;
    async function loadCart() {
      setLoading(true);
      setError(null);
      try {
        const res = await fetchCart();
        if (!cancelled) {
          const nextCart = res.data ?? null;
          setCart(nextCart);
          const seed: Record<string, number> = {};
          nextCart?.items.forEach((item) => {
            seed[item.id] = item.quantity;
          });
          setQuantityInputs(seed);
        }

        setAddressesLoading(true);
        setAddressError(null);
        try {
          const addressResponse = await fetchMyAddresses();
          if (!cancelled) {
            const savedAddresses = addressResponse.data ?? [];
            setAddresses(savedAddresses);
            const defaultAddress = savedAddresses.find((address) => address.is_default);
            if (defaultAddress?.id) {
              setSelectedAddressId(defaultAddress.id);
              setCheckoutForm((current) => ({
                ...current,
                ...checkoutFieldsFromAddress(defaultAddress),
              }));
            }
          }
        } catch (addressErr) {
          if (!cancelled) {
            if (addressErr instanceof ApiError && addressErr.status === 401) {
              router.replace("/login?redirect=/cart");
              return;
            }
            setAddressError(
              addressErr instanceof Error ? addressErr.message : "Saved addresses could not be loaded"
            );
          }
        } finally {
          if (!cancelled) {
            setAddressesLoading(false);
          }
        }
      } catch (err) {
        if (!cancelled) {
          if (err instanceof ApiError && err.status === 401) {
            router.replace("/login?redirect=/cart");
            return;
          }
          setError(err instanceof Error ? err.message : "Failed to fetch cart");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadCart();
    return () => {
      cancelled = true;
    };
  }, [router]);

  const isEmpty = useMemo(() => !cart || cart.items.length === 0, [cart]);

  async function onUpdateItem(itemId: string) {
    const quantity = quantityInputs[itemId];
    if (!quantity || quantity < 1) {
      setError("Quantity must be greater than 0");
      return;
    }

    setItemBusyId(itemId);
    setError(null);
    try {
      const res = await updateCartItem(itemId, quantity);
      setCart(res.data ?? null);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/cart");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to update cart item");
    } finally {
      setItemBusyId(null);
    }
  }

  async function onRemoveItem(itemId: string) {
    setItemBusyId(itemId);
    setError(null);
    try {
      const res = await removeCartItem(itemId);
      setCart(res.data ?? null);
      setQuantityInputs((prev) => {
        const copy = { ...prev };
        delete copy[itemId];
        return copy;
      });
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/cart");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to remove cart item");
    } finally {
      setItemBusyId(null);
    }
  }

  function updateCheckoutField<K extends keyof CheckoutInput>(key: K, value: CheckoutInput[K]) {
    setCheckoutForm((current) => ({ ...current, [key]: value }));
  }

  function applyAddress(address: UserAddress) {
    setCheckoutForm((current) => ({
      ...current,
      ...checkoutFieldsFromAddress(address),
    }));
  }

  function onSelectSavedAddress(addressId: string) {
    setSelectedAddressId(addressId);
    const selected = addresses.find((address) => address.id === addressId);
    if (selected) {
      applyAddress(selected);
    }
  }

  async function onCheckout(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCheckoutLoading(true);
    setError(null);
    try {
      await checkoutOrder(checkoutForm);
      router.push("/orders?checkout=success");
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/cart");
        return;
      }
      setError(err instanceof Error ? err.message : "Checkout failed");
    } finally {
      setCheckoutLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="grid gap-4">
        {Array.from({ length: 3 }).map((_, index) => (
          <div key={index} className="h-36 animate-pulse rounded-[1.5rem] bg-muted/80" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-7">
      <div className="surface-card flex flex-col gap-4 rounded-[2rem] p-6 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="eyebrow">Checkout path</p>
          <h1 className="mt-2 text-4xl font-semibold">Your Cart</h1>
          <p className="mt-2 max-w-xl text-sm text-muted-foreground">
            Review quantities and confirm your order from the live backend cart.
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href="/products">Continue shopping</Link>
        </Button>
      </div>

      {error ? (
        <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      {isEmpty ? (
        <Card className="surface-card">
          <CardContent className="state-panel">
            <p className="text-xl font-semibold">Your cart is empty.</p>
            <p className="max-w-md text-sm text-muted-foreground">
              Add a piece from product detail to begin the checkout flow.
            </p>
            <Button asChild>
              <Link href="/products">Browse products</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid min-w-0 gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
          <div className="space-y-4">
            {cart?.items.map((item) => (
              <Card key={item.id} className="surface-card overflow-hidden rounded-[1.5rem]">
                <CardContent className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex min-w-0 items-center gap-4">
                    <div className="relative h-28 w-24 shrink-0 overflow-hidden rounded-2xl bg-muted">
                      <Image
                        src={item.product.image_url}
                        alt={item.product.name}
                        fill
                        className="object-cover"
                        sizes="120px"
                        placeholder="blur"
                        blurDataURL={PRODUCT_IMAGE_BLUR}
                      />
                    </div>
                    <div className="min-w-0 space-y-1">
                      <p className="break-words font-heading text-xl font-semibold">{item.product.name}</p>
                      <p className="text-sm text-muted-foreground">
                        {item.product.style} / {item.product.color}
                      </p>
                      <p className="text-sm">{formatVND(item.product.price)}</p>
                      <p className="text-sm font-semibold">Subtotal: {formatVND(item.subtotal)}</p>
                    </div>
                  </div>

                  <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">
                    <input
                      type="number"
                      min={1}
                      className="h-10 w-20 rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                      value={quantityInputs[item.id] ?? item.quantity}
                      onChange={(event) =>
                        setQuantityInputs((prev) => ({
                          ...prev,
                          [item.id]: Number(event.target.value),
                        }))
                      }
                    />
                    <Button
                      variant="outline"
                      onClick={() => onUpdateItem(item.id)}
                      disabled={itemBusyId === item.id}
                    >
                      Update
                    </Button>
                    <Button
                      variant="destructive"
                      onClick={() => onRemoveItem(item.id)}
                      disabled={itemBusyId === item.id}
                    >
                      Remove
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          <Card className="surface-card h-fit rounded-[1.5rem] lg:sticky lg:top-28">
            <CardHeader>
              <CardTitle className="text-2xl">Checkout summary</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <form id="checkout-form" className="space-y-4" onSubmit={onCheckout}>
                <div className="space-y-3 rounded-2xl border border-border bg-card/70 p-4 text-sm">
                  <div className="flex items-center gap-2 font-medium">
                    <MapPin className="size-4" />
                    Saved address
                  </div>
                  {addressesLoading ? (
                    <div className="h-10 animate-pulse rounded-xl bg-muted" />
                  ) : addresses.length > 0 ? (
                    <div className="space-y-2">
                      <label htmlFor="saved_address" className="text-xs font-medium text-muted-foreground">
                        Choose saved address
                      </label>
                      <select
                        id="saved_address"
                        value={selectedAddressId}
                        onChange={(event) => onSelectSavedAddress(event.target.value)}
                        className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                      >
                        <option value="">Manual address</option>
                        {addresses.map((address) => (
                          <option key={address.id} value={address.id}>
                            {address.is_default ? "Default - " : ""}
                            {address.recipient_name} / {address.address_line}, {address.district}
                          </option>
                        ))}
                      </select>
                    </div>
                  ) : (
                    <p className="text-xs leading-5 text-muted-foreground">
                      No saved addresses yet. You can type manually or save one from your profile.
                    </p>
                  )}
                  {addressError ? <p className="text-xs text-destructive">{addressError}</p> : null}
                  <Button asChild variant="outline" size="sm" className="w-full">
                    <Link href="/profile/addresses">Manage saved addresses</Link>
                  </Button>
                </div>

                <div className="space-y-3 rounded-2xl border border-border bg-muted/45 p-4 text-sm">
                  <div className="flex items-center gap-2 font-medium">
                    <MapPin className="size-4" />
                    Delivery contact
                  </div>
                  <div className="grid gap-3">
                    <div className="space-y-1">
                      <label htmlFor="recipient_name" className="text-xs font-medium text-muted-foreground">
                        Recipient name
                      </label>
                      <input
                        id="recipient_name"
                        required
                        minLength={2}
                        maxLength={120}
                        value={checkoutForm.recipient_name}
                        onChange={(event) => updateCheckoutField("recipient_name", event.target.value)}
                        className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                        placeholder="Nguyen Van A"
                      />
                    </div>
                    <div className="space-y-1">
                      <label htmlFor="phone" className="text-xs font-medium text-muted-foreground">
                        Phone
                      </label>
                      <input
                        id="phone"
                        required
                        minLength={8}
                        maxLength={32}
                        value={checkoutForm.phone}
                        onChange={(event) => updateCheckoutField("phone", event.target.value)}
                        className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                        placeholder="0901234567"
                      />
                    </div>
                    <div className="space-y-1">
                      <label htmlFor="address_line" className="text-xs font-medium text-muted-foreground">
                        Address
                      </label>
                      <input
                        id="address_line"
                        required
                        minLength={5}
                        maxLength={255}
                        value={checkoutForm.address_line}
                        onChange={(event) => updateCheckoutField("address_line", event.target.value)}
                        className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                        placeholder="123 Nguyen Trai"
                      />
                    </div>
                    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
                      <div className="space-y-1">
                        <label htmlFor="city" className="text-xs font-medium text-muted-foreground">
                          City
                        </label>
                        <input
                          id="city"
                          required
                          minLength={2}
                          maxLength={120}
                          value={checkoutForm.city}
                          onChange={(event) => updateCheckoutField("city", event.target.value)}
                          className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                          placeholder="Ho Chi Minh City"
                        />
                      </div>
                      <div className="space-y-1">
                        <label htmlFor="district" className="text-xs font-medium text-muted-foreground">
                          District
                        </label>
                        <input
                          id="district"
                          required
                          minLength={2}
                          maxLength={120}
                          value={checkoutForm.district}
                          onChange={(event) => updateCheckoutField("district", event.target.value)}
                          className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                          placeholder="District 1"
                        />
                      </div>
                    </div>
                    <div className="space-y-1">
                      <label htmlFor="note" className="text-xs font-medium text-muted-foreground">
                        Delivery note
                      </label>
                      <textarea
                        id="note"
                        maxLength={1000}
                        value={checkoutForm.note ?? ""}
                        onChange={(event) => updateCheckoutField("note", event.target.value)}
                        className="min-h-20 w-full rounded-xl border border-input bg-card px-3 py-2 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                        placeholder="Call before delivery, preferred time, building note..."
                      />
                    </div>
                  </div>
                </div>

                <div className="grid gap-3 rounded-2xl border border-border bg-card/70 p-4 text-sm">
                  <div className="space-y-1">
                    <label htmlFor="shipping_method" className="text-xs font-medium text-muted-foreground">
                      Shipping method
                    </label>
                    <select
                      id="shipping_method"
                      value={checkoutForm.shipping_method}
                      onChange={(event) =>
                        updateCheckoutField("shipping_method", event.target.value as CheckoutInput["shipping_method"])
                      }
                      className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                    >
                      <option value="standard">Standard delivery</option>
                      <option value="express">Express delivery</option>
                    </select>
                  </div>
                  <div className="space-y-1">
                    <label htmlFor="payment_method" className="text-xs font-medium text-muted-foreground">
                      Payment method
                    </label>
                    <select
                      id="payment_method"
                      value={checkoutForm.payment_method}
                      onChange={(event) =>
                        updateCheckoutField("payment_method", event.target.value as CheckoutInput["payment_method"])
                      }
                      className="h-10 w-full rounded-xl border border-input bg-card px-3 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                    >
                      <option value="cod">COD - Cash on delivery</option>
                      <option value="demo_payment">Demo payment placeholder</option>
                    </select>
                  </div>
                  <p className="text-xs leading-5 text-muted-foreground">
                    No real card, bank, or wallet credentials are collected in this MVP checkout.
                  </p>
                </div>
              </form>

              <div className="space-y-3 rounded-2xl border border-border bg-muted/45 p-4 text-sm">
                <div className="flex items-center gap-2 font-medium">
                  <MapPin className="size-4" />
                  Order destination
                </div>
                <p className="leading-6 text-muted-foreground">
                  Saved on the order so customer and admin order screens can show fulfillment details.
                </p>
              </div>
              <div className="grid gap-2 rounded-2xl border border-border bg-card/70 p-4 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Items</span>
                  <span>{cart?.items.length ?? 0}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Shipping</span>
                  <span>Calculated later</span>
                </div>
                <div className="flex items-center justify-between border-t border-border pt-3">
                  <span className="font-medium">Total</span>
                  <span className="font-heading text-2xl font-semibold">{formatVND(cart?.total ?? 0)}</span>
                </div>
              </div>
              <div className="grid gap-2 text-xs text-muted-foreground">
                <p className="inline-flex items-center gap-2">
                  <ShieldCheck className="size-4" /> Stock is validated again during checkout.
                </p>
                <p className="inline-flex items-center gap-2">
                  <Truck className="size-4" /> Order status appears in your order history after success.
                </p>
              </div>
              <div className="flex items-center justify-between rounded-2xl bg-primary p-4 text-primary-foreground">
                <span className="text-sm opacity-80">Payable now</span>
                <span className="font-heading text-2xl font-semibold">
                  {formatVND(cart?.total ?? 0)}
                </span>
              </div>
              <Button form="checkout-form" type="submit" className="w-full" disabled={checkoutLoading || isEmpty}>
                {checkoutLoading ? "Processing..." : "Checkout"}
              </Button>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
