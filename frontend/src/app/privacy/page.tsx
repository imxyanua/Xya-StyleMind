import { InfoPage } from "@/components/site/info-page";

export default function PrivacyPage() {
  return (
    <InfoPage
      eyebrow="Privacy"
      title="Privacy principles for a public ecommerce demo."
      description="Xya-StyleMind avoids committing real secrets or customer data. This page explains the privacy posture expected before turning the demo into a production deployment."
      sections={[
        {
          title: "Data collected",
          body: "The application stores account email, full name, role, status, cart/order/review/wishlist records, and audit events required for ecommerce workflows. Passwords must be stored only as secure hashes.",
        },
        {
          title: "Sensitive data",
          body: "Raw tokens, passwords, API keys, payment card data, and production secrets should never be logged, exposed in metrics, or committed to source control.",
        },
        {
          title: "Public repository note",
          body: "Migration schema files are safe to commit, but .env files, database dumps, backups, private keys, and real user data must remain ignored and rotated if leaked.",
        },
        {
          title: "User rights",
          body: "A production version should support account access, correction, deletion requests, retention policies, and operational audit controls appropriate to its market.",
        },
      ]}
    />
  );
}
