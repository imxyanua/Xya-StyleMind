import { InfoPage } from "@/components/site/info-page";

export default function TermsPage() {
  return (
    <InfoPage
      eyebrow="Terms"
      title="Demo terms for using Xya-StyleMind."
      description="These terms are professional placeholder copy for a public portfolio project. Production legal terms should be reviewed by qualified counsel before launch."
      sections={[
        {
          title: "Demo use",
          body: "Xya-StyleMind is presented as a software engineering and ecommerce experience demo. Product listings, prices, contacts, and policies may be illustrative unless connected to a real merchant account.",
        },
        {
          title: "Accounts",
          body: "Users are responsible for keeping credentials secure. Admin functions are restricted by role and should only be used in controlled testing environments.",
        },
        {
          title: "Orders and payments",
          body: "The MVP order flow validates cart, inventory, and checkout behavior but does not process real payments. Production payment terms must define authorization, settlement, cancellation, and refund handling.",
        },
        {
          title: "Limitations",
          body: "The platform is provided as-is for demonstration. Production deployment requires final security review, monitoring, backups, domain configuration, payment integration, and operational policies.",
        },
      ]}
    />
  );
}
