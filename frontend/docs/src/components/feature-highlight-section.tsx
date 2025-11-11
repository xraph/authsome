'use client';
import { Zap, Cpu, Fingerprint, Pencil, Settings2, Sparkles, Code, Globe, Shield, Users, Lock } from 'lucide-react';
import { motion, useReducedMotion } from 'motion/react';
import { FeatureCard } from '@/components/blocks/grid-feature-cards';

// const features = [
// 	{
// 		title: 'Faaast',
// 		icon: Zap,
// 		description: 'It supports an entire helping developers and innovate.',
// 	},
// 	{
// 		title: 'Powerful',
// 		icon: Cpu,
// 		description: 'It supports an entire helping developers and businesses.',
// 	},
// 	{
// 		title: 'Security',
// 		icon: Fingerprint,
// 		description: 'It supports an helping developers businesses.',
// 	},
// 	{
// 		title: 'Customization',
// 		icon: Pencil,
// 		description: 'It supports helping developers and businesses innovate.',
// 	},
// 	{
// 		title: 'Control',
// 		icon: Settings2,
// 		description: 'It supports helping developers and businesses innovate.',
// 	},
// 	{
// 		title: 'Built for AI',
// 		icon: Sparkles,
// 		description: 'It supports helping developers and businesses innovate.',
// 	},
// ];


const features = [
    {
      icon: Shield,
      title: "Enterprise Security",
      description:
        "Built-in security features including rate limiting, device tracking, and audit logging with comprehensive threat detection.",
      className: "md:col-span-2",
      header: (
        <div className="flex h-full min-h-[6rem] w-full flex-1 rounded-xl bg-gradient-to-br from-blue-500/10 via-purple-500/10 to-pink-500/10 dark:from-blue-500/20 dark:via-purple-500/20 dark:to-pink-500/20" />
      ),
    },
    {
      icon: Users,
      title: "Multi-Tenancy",
      description:
        "Organization-scoped configurations and user management with seamless tenant isolation.",
      className: "md:col-span-1",
      header: (
        <div className="flex h-full min-h-[6rem] w-full flex-1 rounded-xl bg-gradient-to-br from-green-500/10 via-emerald-500/10 to-teal-500/10 dark:from-green-500/20 dark:via-emerald-500/20 dark:to-teal-500/20" />
      ),
    },
    {
      icon: Code,
      title: "Plugin Architecture",
      description:
        "12+ authentication methods via extensible plugin system. Add custom auth flows easily.",
      className: "md:col-span-1",
      header: (
        <div className="flex h-full min-h-[6rem] w-full flex-1 rounded-xl bg-gradient-to-br from-orange-500/10 via-red-500/10 to-pink-500/10 dark:from-orange-500/20 dark:via-red-500/20 dark:to-pink-500/20" />
      ),
    },
    {
      icon: Globe,
      title: "Framework Agnostic",
      description:
        "Mounts on Forge framework but designed to work with any Go web framework.",
      className: "md:col-span-2",
      header: (
        <div className="flex h-full min-h-[6rem] w-full flex-1 rounded-xl bg-gradient-to-br from-cyan-500/10 via-blue-500/10 to-indigo-500/10 dark:from-cyan-500/20 dark:via-blue-500/20 dark:to-indigo-500/20" />
      ),
    },
    {
      icon: Lock,
      title: "RBAC & Policies",
      description:
        "Role-based access control with policy language for fine-grained permissions.",
      className: "md:col-span-1",
      header: (
        <div className="flex h-full min-h-[6rem] w-full flex-1 rounded-xl bg-gradient-to-br from-violet-500/10 via-purple-500/10 to-fuchsia-500/10 dark:from-violet-500/20 dark:via-purple-500/20 dark:to-fuchsia-500/20" />
      ),
    },
    {
      icon: Zap,
      title: "High Performance",
      description:
        "Session caching with Redis, connection pooling, and optimized database queries for lightning-fast responses.",
      className: "md:col-span-2",
      header: (
        <div className="flex h-full min-h-[6rem] w-full flex-1 rounded-xl bg-gradient-to-br from-yellow-500/10 via-orange-500/10 to-red-500/10 dark:from-yellow-500/20 dark:via-orange-500/20 dark:to-red-500/20" />
      ),
    },
  ];

export default function FeatureHighlightSection() {
	return (
		<section className="pt-16 md:pt-32">
			<div className="mx-auto w-full max-w-7xl space-y-8 px-4">
				<AnimatedContainer className="mx-auto max-w-3xl text-center">
					<h2 className="text-3xl font-bold tracking-wide text-balance md:text-4xl lg:text-5xl xl:font-extrabold">
						Flexible. Speed. Pluggable.
					</h2>
					<p className="text-muted-foreground mt-4 text-sm tracking-wide text-balance md:text-base">
                      From simple username/password to enterprise SSO, AuthSome provides all the tools you need.
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
