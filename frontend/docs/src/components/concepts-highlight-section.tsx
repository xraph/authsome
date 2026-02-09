import { BentoGrid, type BentoItem } from "@/components/ui/bento-grid"
import {
    Users,
    Fingerprint,
    ShieldCheck,
    Code,
} from "lucide-react";

const items: BentoItem[] = [
    {
        title: "Multi-Tenancy",
        meta: "Standalone & SaaS",
        description:
            "Standalone mode for single-tenant apps, SaaS mode for full multi-tenancy with Clerk.js-style organizations and org-scoped configuration overrides.",
        icon: <Users className="w-4 h-4 text-blue-500" />,
        status: "Production",
        tags: ["Organizations", "Tenant Isolation", "Config Overrides"],
        colSpan: 2,
        hasPersistentHover: true,
    },
    {
        title: "Authentication Methods",
        meta: "30+ plugins",
        description: "Passwordless login, MFA, Social OAuth with 17+ providers, Enterprise SSO, API keys, and more -- all via a unified plugin system.",
        icon: <Fingerprint className="w-4 h-4 text-emerald-500" />,
        status: "Extensible",
        tags: ["Passkey", "Magic Link", "SAML", "OAuth"],
    },
    {
        title: "Security & Compliance",
        meta: "Enterprise-grade",
        description: "Adaptive MFA, device tracking, risk-based authentication, audit logging with Datadog/Splunk exporters, GDPR/HIPAA compliance, and geofencing.",
        icon: <ShieldCheck className="w-4 h-4 text-purple-500" />,
        tags: ["Audit", "Compliance", "Device Tracking"],
        colSpan: 2,
    },
    {
        title: "Developer Experience",
        meta: "Clean Architecture",
        description: "Service-oriented design with repository pattern, extensible plugin system, Smartform integration for dynamic forms, and comprehensive API.",
        icon: <Code className="w-4 h-4 text-sky-500" />,
        status: "Documented",
        tags: ["Plugins", "Forge", "Type-safe"],
    },
];

function ConceptsHighlightSection() {
    return (
        <section className="py-16 md:py-24">
            <div className="mx-auto max-w-7xl px-4">
                <div className="mx-auto max-w-2xl text-center mb-12">
                    <h2 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
                        Core Concepts
                    </h2>
                    <p className="mt-4 text-lg text-muted-foreground">
                        The building blocks behind AuthSome&apos;s architecture.
                    </p>
                </div>
                <BentoGrid items={items} />
            </div>
        </section>
    );
}

export default ConceptsHighlightSection;
