import { InfoPage } from "@/components/site/info-page";

export default function ShippingPage() {
  return (
    <InfoPage
      eyebrow="Shipping"
      title="Shipping policy designed for a clear checkout experience."
      description="The current backend checkout creates orders without collecting delivery addresses yet, so this page documents the intended production policy and customer expectations."
      cta={{ href: "/cart", label: "View cart" }}
      sections={[
        {
          title: "Processing time",
          body: "Orders would normally be reviewed and packed within 1-2 business days after payment confirmation. Admin order status reflects pending, paid, shipping, completed, or cancelled states.",
        },
        {
          title: "Delivery areas",
          body: "The demo storefront is prepared for Vietnam-first delivery copy, with room to expand to international shipping rules, address validation, and carrier integrations.",
        },
        {
          title: "Tracking",
          body: "Once an order moves to shipping, production systems would send tracking information by email and display it in the order detail page. The current MVP focuses on order status history.",
        },
        {
          title: "Shipping fees",
          body: "Shipping costs are not charged in the MVP checkout. A production checkout can add fee calculation by address, carrier, order value, or promotional threshold.",
        },
      ]}
    />
  );
}
