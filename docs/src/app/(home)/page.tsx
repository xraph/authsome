import { AuthFlowSection } from "@/components/landing/auth-flow-section";
import { AuthStrategiesSection } from "@/components/landing/auth-strategies-section";
import { CodeShowcase } from "@/components/landing/code-showcase";
import { CTA } from "@/components/landing/cta";
import { DashboardSection } from "@/components/landing/dashboard-section";
import { EnterpriseSection } from "@/components/landing/enterprise-section";
import { FeatureBento } from "@/components/landing/feature-bento";
import { Hero } from "@/components/landing/hero";
import { OrgsRbacSection } from "@/components/landing/orgs-rbac-section";
import { SdkEcosystemSection } from "@/components/landing/sdk-ecosystem-section";
import { SecuritySection } from "@/components/landing/security-section";
import { UIShowcase } from "@/components/landing/ui-showcase";

export default function HomePage() {
  return (
    <main className="flex flex-col items-center overflow-x-hidden relative">
      <Hero />
      <FeatureBento />
      <AuthStrategiesSection />
      <SecuritySection />
      <AuthFlowSection />
      <OrgsRbacSection />
      <CodeShowcase />
      <SdkEcosystemSection />
      <UIShowcase />
      <EnterpriseSection />
      <DashboardSection />
      <CTA />
    </main>
  );
}
