"use client";

import { type FormEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { MapPin, Plus, Star, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  createMyAddress,
  deleteMyAddress,
  fetchMyAddresses,
  setDefaultMyAddress,
  updateMyAddress,
  type AddressInput,
  type UserAddress,
} from "@/lib/api";
import { getToken } from "@/lib/auth";
import { ApiError } from "@/types/api";

const EMPTY_FORM: AddressInput = {
  recipient_name: "",
  phone: "",
  address_line: "",
  city: "",
  district: "",
  note: "",
  is_default: false,
};

function addressLabel(address: UserAddress) {
  return [address.address_line, address.district, address.city].filter(Boolean).join(", ");
}

function toForm(address: UserAddress): AddressInput {
  return {
    recipient_name: address.recipient_name ?? "",
    phone: address.phone ?? "",
    address_line: address.address_line ?? "",
    city: address.city ?? "",
    district: address.district ?? "",
    note: address.note ?? "",
    is_default: Boolean(address.is_default),
  };
}

export default function AddressBookPage() {
  const router = useRouter();
  const [addresses, setAddresses] = useState<UserAddress[]>([]);
  const [form, setForm] = useState<AddressInput>(EMPTY_FORM);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login?redirect=/profile/addresses");
      return;
    }

    let cancelled = false;
    async function loadAddresses() {
      setLoading(true);
      setError(null);
      try {
        const response = await fetchMyAddresses();
        if (!cancelled) {
          setAddresses(response.data ?? []);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        if (err instanceof ApiError && err.status === 401) {
          router.replace("/login?redirect=/profile/addresses");
          return;
        }
        setError(err instanceof Error ? err.message : "Failed to load saved addresses");
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadAddresses();
    return () => {
      cancelled = true;
    };
  }, [router]);

  const sortedAddresses = useMemo(() => {
    return [...addresses].sort((a, b) => Number(Boolean(b.is_default)) - Number(Boolean(a.is_default)));
  }, [addresses]);

  function updateField<K extends keyof AddressInput>(key: K, value: AddressInput[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function resetForm() {
    setForm(EMPTY_FORM);
    setEditingId(null);
  }

  async function reloadAddresses() {
    const response = await fetchMyAddresses();
    setAddresses(response.data ?? []);
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError(null);
    setMessage(null);
    try {
      if (editingId) {
        await updateMyAddress(editingId, form);
        setMessage("Address updated.");
      } else {
        await createMyAddress(form);
        setMessage("Address saved.");
      }
      resetForm();
      await reloadAddresses();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/profile/addresses");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to save address");
    } finally {
      setSaving(false);
    }
  }

  function onEdit(address: UserAddress) {
    if (!address.id) {
      return;
    }
    setEditingId(address.id);
    setForm(toForm(address));
    setMessage(null);
    setError(null);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  async function onDelete(address: UserAddress) {
    if (!address.id) {
      return;
    }
    const confirmed = window.confirm("Delete this saved address?");
    if (!confirmed) {
      return;
    }

    setBusyId(address.id);
    setError(null);
    setMessage(null);
    try {
      await deleteMyAddress(address.id);
      if (editingId === address.id) {
        resetForm();
      }
      setMessage("Address deleted.");
      await reloadAddresses();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/profile/addresses");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to delete address");
    } finally {
      setBusyId(null);
    }
  }

  async function onSetDefault(address: UserAddress) {
    if (!address.id) {
      return;
    }
    setBusyId(address.id);
    setError(null);
    setMessage(null);
    try {
      await setDefaultMyAddress(address.id);
      setMessage("Default address updated.");
      await reloadAddresses();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        router.replace("/login?redirect=/profile/addresses");
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to update default address");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div className="space-y-7">
      <section className="surface-card rounded-[2rem] p-6 sm:p-8">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="eyebrow">Account center</p>
            <h1 className="mt-2 text-4xl font-semibold sm:text-5xl">Address Book</h1>
            <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground">
              Save delivery addresses once, then reuse them during checkout without retyping every field.
            </p>
          </div>
          <Button asChild variant="outline">
            <Link href="/profile">Back to profile</Link>
          </Button>
        </div>
      </section>

      {error ? (
        <p className="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}
      {message ? (
        <p className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700 dark:text-emerald-300">
          {message}
        </p>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[420px_minmax(0,1fr)]">
        <Card className="surface-card h-fit rounded-[1.75rem] lg:sticky lg:top-28">
          <CardHeader>
            <p className="eyebrow">{editingId ? "Edit address" : "New address"}</p>
            <CardTitle className="flex items-center gap-2 text-3xl">
              <Plus className="size-6" /> {editingId ? "Update delivery spot" : "Save a delivery spot"}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={onSubmit}>
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
                    value={form.recipient_name}
                    onChange={(event) => updateField("recipient_name", event.target.value)}
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
                    value={form.phone}
                    onChange={(event) => updateField("phone", event.target.value)}
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
                    value={form.address_line}
                    onChange={(event) => updateField("address_line", event.target.value)}
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
                      value={form.city}
                      onChange={(event) => updateField("city", event.target.value)}
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
                      value={form.district}
                      onChange={(event) => updateField("district", event.target.value)}
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
                    value={form.note ?? ""}
                    onChange={(event) => updateField("note", event.target.value)}
                    className="min-h-24 w-full rounded-xl border border-input bg-card px-3 py-2 text-sm outline-none focus:border-ring focus:ring-2 focus:ring-ring/30"
                    placeholder="Call before delivery, gate code, preferred time..."
                  />
                </div>
                <label className="flex items-center gap-2 rounded-2xl border border-border bg-muted/50 p-3 text-sm">
                  <input
                    type="checkbox"
                    checked={Boolean(form.is_default)}
                    onChange={(event) => updateField("is_default", event.target.checked)}
                    className="size-4 rounded border-input"
                  />
                  <span>Default address</span>
                </label>
              </div>
              <div className="flex flex-col gap-2 sm:flex-row">
                <Button type="submit" disabled={saving} className="sm:flex-1">
                  {saving ? "Saving..." : editingId ? "Update address" : "Save address"}
                </Button>
                {editingId ? (
                  <Button type="button" variant="outline" onClick={resetForm}>
                    Cancel edit
                  </Button>
                ) : null}
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="surface-card rounded-[1.75rem]">
          <CardHeader>
            <p className="eyebrow">Saved locations</p>
            <CardTitle className="text-3xl">Your addresses</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="grid gap-3">
                {Array.from({ length: 3 }).map((_, index) => (
                  <div key={index} className="h-32 animate-pulse rounded-3xl bg-muted/80" />
                ))}
              </div>
            ) : sortedAddresses.length === 0 ? (
              <div className="state-panel min-h-72">
                <MapPin className="size-10 text-muted-foreground" />
                <p className="text-xl font-semibold">No saved addresses yet.</p>
                <p className="max-w-md text-sm text-muted-foreground">
                  Add one now, then checkout can auto-fill delivery contact and location details.
                </p>
              </div>
            ) : (
              <div className="grid gap-3">
                {sortedAddresses.map((address) => (
                  <article key={address.id} className="rounded-3xl border border-border bg-card/70 p-4">
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <h2 className="break-words text-xl font-semibold">{address.recipient_name}</h2>
                          {address.is_default ? <Badge>Default</Badge> : null}
                        </div>
                        <p className="mt-1 text-sm text-muted-foreground">{address.phone}</p>
                        <p className="mt-3 break-words text-sm leading-6">{addressLabel(address)}</p>
                        {address.note ? (
                          <p className="mt-2 break-words rounded-2xl bg-muted/60 px-3 py-2 text-xs text-muted-foreground">
                            {address.note}
                          </p>
                        ) : null}
                      </div>
                      <div className="flex flex-wrap gap-2 sm:justify-end">
                        <Button variant="outline" size="sm" onClick={() => onEdit(address)}>
                          Edit
                        </Button>
                        {!address.is_default ? (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => onSetDefault(address)}
                            disabled={busyId === address.id}
                          >
                            <Star className="mr-1 size-4" /> Set default
                          </Button>
                        ) : null}
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => onDelete(address)}
                          disabled={busyId === address.id}
                        >
                          <Trash2 className="mr-1 size-4" /> Delete
                        </Button>
                      </div>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
