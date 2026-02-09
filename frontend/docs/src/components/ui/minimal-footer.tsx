import {
	GithubIcon,
	LinkedinIcon,
	TwitterIcon,
} from 'lucide-react';

export function MinimalFooter() {
	const year = new Date().getFullYear();

	const company = [
		{
			title: 'About',
			href: 'https://xraph.com/about',
			target: '_blank',
		},
		{
			title: 'GitHub',
			href: 'https://github.com/xraph/authsome',
			target: '_blank',
		},
	];

	const resources = [
		{
			title: 'Documentation',
			href: '/docs',
		},
		{
			title: 'Getting Started',
			href: '/docs/go/getting-started',
		},
		{
			title: 'API Reference',
			href: '/docs/go/api',
		},
		{
			title: 'Examples',
			href: '/docs/go/examples',
		},
		{
			title: 'Plugins',
			href: '/docs/go/plugins',
		},
	];

	const socialLinks = [
		{
			icon: <GithubIcon className="size-4" />,
			link: 'https://github.com/xraph/authsome',
			target: '_blank',
		},
		{
			icon: <LinkedinIcon className="size-4" />,
			link: 'https://www.linkedin.com/in/rex-raphael/',
			target: '_blank',
		},
		{
			icon: <TwitterIcon className="size-4" />,
			link: 'https://twitter.com/nicerapha',
			target: '_blank',
		},
	];

	return (
		<footer className="relative">
			<div className="bg-[radial-gradient(35%_80%_at_30%_0%,--theme(--color-foreground/.1),transparent)] mx-auto max-w-7xl md:border-x">
				<div className="bg-border absolute inset-x-0 h-px w-full" />
				<div className="grid max-w-7xl grid-cols-12 gap-6 p-4">
					<div className="md:col-span-6 flex flex-col gap-5 col-span-4">
						<span className="text-lg font-semibold">AuthSome</span>
						<p className="text-muted-foreground max-w-sm font-mono text-sm text-balance">
							A comprehensive, pluggable authentication framework for Go applications.
						</p>
						<div className="flex gap-2">
							{socialLinks.map((item, i) => (
								<a
									key={i}
									className="hover:bg-accent rounded-md border p-1.5"
									target={item.target}
									href={item.link}
								>
									{item.icon}
								</a>
							))}
						</div>
					</div>
					<div className="md:col-span-3 w-full col-span-1">
						<span className="text-muted-foreground mb-1 text-xs">
							Resources
						</span>
						<div className="flex flex-col gap-1">
							{resources.map(({ href, title }, i) => (
								<a
									key={i}
									className={`w-max py-1 text-sm duration-200 hover:underline`}
									href={href}
								>
									{title}
								</a>
							))}
						</div>
					</div>
					<div className="md:col-span-3 w-full col-span-1">
						<span className="text-muted-foreground mb-1 text-xs">Company</span>
						<div className="flex flex-col gap-1">
							{company.map(({ href, title, target }, i) => (
								<a
									key={i}
									className={`w-max py-1 text-sm duration-200 hover:underline`}
									href={href}
									target={target}
								>
									{title}
								</a>
							))}
						</div>
					</div>
				</div>
				<div className="bg-border absolute inset-x-0 h-px w-full" />
				<div className="flex max-w-4xl flex-col justify-between gap-2 pt-2 pb-5 px-4">
					<p className="text-muted-foreground text-sm">
						&copy; {year} <a href="https://xraph.com" className="hover:underline">xraph</a>. All rights reserved.
					</p>
				</div>
			</div>
		</footer>
	);
}
