import { InfoPage } from "@/components/site/info-page";

export default function ReturnsPage() {
  return (
    <InfoPage
      eyebrow="Returns"
      title="A fair returns flow keeps fashion shopping low-friction."
      description="This demo page documents practical return expectations for apparel ecommerce while leaving refund automation for a later production phase."
      cta={{ href: "/orders", label: "Check orders" }}
      sections={[
        {
          title: "Return window",
          body: "Eligible items would be returnable within 7-14 days after delivery, provided they are unused, unworn, and returned with original packaging and tags.",
        },
        {
          title: "Condition checks",
          body: "Fashion returns should verify stains, damage, fragrance, missing accessories, and tag condition before approving a refund or exchange.",
        },
        {
          title: "Refund handling",
          body: "Production refunds should return funds to the original payment method after warehouse inspection. The MVP does not process real payments or refunds.",
        },
        {
          title: "Exchanges",
          body: "Size and color exchanges can be supported later by creating replacement orders or inventory reservations once address and shipment modules exist.",
        },
      ]}
    />
  );
}
