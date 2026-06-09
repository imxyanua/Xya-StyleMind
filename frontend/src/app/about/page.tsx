import { InfoPage } from "@/components/site/info-page";

export default function AboutPage() {
  return (
    <InfoPage
      eyebrow="About"
      title="A fashion ecommerce platform built for smarter shopping."
      description="Xya-StyleMind is a full-stack ecommerce demo that combines a production-minded Go backend with a modern Next.js storefront. It is designed to feel like a real fashion shop today and evolve into AI-assisted styling tomorrow."
      cta={{ href: "/products", label: "Browse products" }}
      sections={[
        {
          title: "What we sell",
          body: "The catalog focuses on practical fashion categories across streetwear, minimal, Korean, formal, casual, and sporty styles. Products include stock, price, rating, color, and category metadata so shoppers can filter with intent.",
        },
        {
          title: "Why it exists",
          body: "The project demonstrates a complete ecommerce foundation: authentication, catalog browsing, cart, checkout, orders, wishlist, reviews, admin operations, audit logs, observability, and API documentation.",
        },
        {
          title: "AI roadmap",
          body: "Future phases can add outfit recommendation, smart search, personalized styling prompts, and visual merchandising powered by Gemini/OpenAI/Anthropic integrations without rewriting the core commerce flow.",
        },
        {
          title: "Built for maintainability",
          body: "The frontend consumes the backend contract through OpenAPI-generated types, while the backend follows service/repository patterns, migrations, tests, JWT hardening, Redis-backed rate limiting, and audit logging.",
        },
      ]}
    />
  );
}
