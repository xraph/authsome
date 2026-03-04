import { AuthFlowSection } from "@/components/landing/auth-flow-section";
import { CodeShowcase } from "@/components/landing/code-showcase";
import { CTA } from "@/components/landing/cta";
import { FeatureBento } from "@/components/landing/feature-bento";
import { Footer } from "@/components/landing/footer";
import { Hero } from "@/components/landing/hero";
import { UIShowcase } from "@/components/landing/ui-showcase";

export default function HomePage() {
  return (
    <main className="flex flex-col items-center overflow-x-hidden relative">
      <Hero />
      <FeatureBento />
      <AuthFlowSection />
      <CodeShowcase />
      <UIShowcase />
      <CTA />
      <Footer />
    </main>
  );
}
