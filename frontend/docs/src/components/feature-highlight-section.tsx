'use client';
import { Zap, Fingerprint, Shield, Users, Lock, Server } from 'lucide-react';
import { motion, useReducedMotion } from 'motion/react';
import { FeatureCard } from '@/components/blocks/grid-feature-cards';

const features = [
	{
		icon: Shield,
		title: 'Enterprise Security',
		description:
			'Rate limiting, device tracking, and audit logging with exporters for Datadog, Splunk, and Syslog. Adaptive risk-based authentication out of the box.',
	},
	{
		icon: Users,
		title: 'Multi-Tenancy',
		description:
			'Standalone and SaaS modes with Clerk.js-style organizations. Org-scoped config overrides, user management, and seamless tenant isolation.',
	},
	{
		icon: Fingerprint,
		title: '30+ Plugins',
		description:
			'Passwordless, MFA, Social OAuth, Enterprise SSO, API keys, SCIM, and more via an extensible plugin system. Build custom plugins in minutes.',
	},
	{
		icon: Server,
		title: 'Built for Forge',
		description:
			'Native integration with the Forge framework. Mount AuthSome on your app, configure via YAML, and leverage Forge middleware and config management.',
	},
	{
		icon: Lock,
		title: 'RBAC & Permissions',
		description:
			'Role-based access control with a policy engine for fine-grained, org-scoped permissions. Define custom roles and enforce them at every layer.',
	},
	{
		icon: Zap,
		title: 'High Performance',
		description:
			'Session caching with Redis, connection pooling, and optimized queries. Supports PostgreSQL, MySQL, and SQLite with Bun ORM.',
	},
];

export default function FeatureHighlightSection() {
	return (
		<section className="pt-16 md:pt-32">
			<div className="mx-auto w-full max-w-7xl space-y-8 px-4">
				<AnimatedContainer className="mx-auto max-w-3xl text-center">
					<h2 className="text-3xl font-bold tracking-wide text-balance md:text-4xl lg:text-5xl xl:font-extrabold">
						Everything you need for authentication
					</h2>
					<p className="text-muted-foreground mt-4 text-sm tracking-wide text-balance md:text-base">
						From simple email/password to enterprise SSO and compliance -- AuthSome ships with 30+ plugins covering every authentication scenario.
					</p>
				</AnimatedContainer>

				<AnimatedContainer
					delay={0.4}
					className="grid grid-cols-1 divide-x divide-y divide-dashed border border-dashed sm:grid-cols-2 md:grid-cols-3"
				>
					{features.map((feature, i) => (
						<FeatureCard key={i} feature={feature} />
					))}
				</AnimatedContainer>
			</div>
		</section>
	);
}

type ViewAnimationProps = {
	delay?: number;
	className?: React.ComponentProps<typeof motion.div>['className'];
	children: React.ReactNode;
};

function AnimatedContainer({ className, delay = 0.1, children }: ViewAnimationProps) {
	const shouldReduceMotion = useReducedMotion();

	if (shouldReduceMotion) {
		return children;
	}

	return (
		<motion.div
			initial={{ filter: 'blur(4px)', translateY: -8, opacity: 0 }}
			whileInView={{ filter: 'blur(0px)', translateY: 0, opacity: 1 }}
			viewport={{ once: true }}
			transition={{ delay, duration: 0.8 }}
			className={className}
		>
			{children}
		</motion.div>
	);
}
