import Link from "next/link";
import { ArrowRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type InfoPageProps = {
  eyebrow: string;
  title: string;
  description: string;
  sections: Array<{
    title: string;
    body: string;
  }>;
  cta?: {
    href: string;
    label: string;
  };
};

export function InfoPage({ eyebrow, title, description, sections, cta }: InfoPageProps) {
  return (
    <div className="space-y-8">
      <section className="surface-card overflow-hidden rounded-[2.5rem] p-6 sm:p-8 lg:p-10">
        <div className="max-w-3xl">
          <p className="eyebrow">{eyebrow}</p>
          <h1 className="mt-3 text-5xl font-semibold leading-tight sm:text-6xl">{title}</h1>
          <p className="mt-5 text-base leading-7 text-muted-foreground sm:text-lg">{description}</p>
          {cta ? (
            <Button asChild className="mt-6" size="lg">
              <Link href={cta.href}>
                {cta.label} <ArrowRight className="size-4" />
              </Link>
            </Button>
          ) : null}
        </div>
      </section>

      <section className="grid gap-5 md:grid-cols-2">
        {sections.map((section) => (
          <Card key={section.title} className="surface-card rounded-[1.75rem]">
            <CardHeader>
              <CardTitle className="text-2xl">{section.title}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm leading-7 text-muted-foreground">{section.body}</p>
            </CardContent>
          </Card>
        ))}
      </section>
    </div>
  );
}
