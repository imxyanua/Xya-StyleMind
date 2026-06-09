import { InfoPage } from "@/components/site/info-page";

export default function ContactPage() {
  return (
    <InfoPage
      eyebrow="Contact"
      title="Need help with an order, product, or demo walkthrough?"
      description="This public demo uses placeholder contact channels, but the page is structured like a real customer support surface for ecommerce operations."
      cta={{ href: "/products", label: "Continue shopping" }}
      sections={[
        {
          title: "Customer support",
          body: "Email hello@xya-stylemind.demo for order questions, sizing help, return guidance, or account support. In a production setup this would connect to a support inbox or ticketing tool.",
        },
        {
          title: "Business inquiries",
          body: "For collaboration, portfolio review, or technical walkthroughs, include your preferred meeting time and the area you want to discuss: backend, frontend, DevOps, security, or AI roadmap.",
        },
        {
          title: "Response expectations",
          body: "Demo support copy targets a one-business-day response window. Urgent payment or delivery issues would be prioritized in a real ecommerce environment.",
        },
        {
          title: "Demo note",
          body: "Do not send real secrets, payment data, or private customer information to demo contact channels. Production deployments should use verified support tooling and access controls.",
        },
      ]}
    />
  );
}
