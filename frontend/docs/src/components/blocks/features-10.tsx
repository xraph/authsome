import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import { ShieldCheck, LayoutDashboard, LucideIcon } from 'lucide-react'
import { ReactNode } from 'react'

export function Features() {
    return (
        <section className="bg-zinc-50 py-16 md:py-32 dark:bg-transparent">
            <div className="mx-auto px-6 max-w-7xl">
                <div className="mx-auto grid gap-4 lg:grid-cols-2">
                    <FeatureCard>
                        <CardHeader className="pb-3">
                            <CardHeading
                                icon={ShieldCheck}
                                title="Consent and Compliance"
                                description="Built-in GDPR, HIPAA, and CCPA compliance with user consent management workflows."
                            />
                        </CardHeader>

                        <div className="relative mb-6 border-t border-dashed sm:mb-0">
                            <div className="absolute inset-0 [background:radial-gradient(125%_125%_at_50%_0%,transparent_40%,hsl(var(--muted)),white_125%)]"></div>
                            <div className="aspect-[76/59] p-1 px-6 flex items-center justify-center">
                                <div className="w-full max-w-sm space-y-3 py-6">
                                    <div className="rounded-lg border border-border bg-card p-4">
                                        <div className="flex items-center gap-3 mb-3">
                                            <div className="h-8 w-8 rounded-full bg-green-500/20 flex items-center justify-center">
                                                <ShieldCheck className="h-4 w-4 text-green-600 dark:text-green-400" />
                                            </div>
                                            <div>
                                                <div className="text-sm font-medium">Data Processing Consent</div>
                                                <div className="text-xs text-muted-foreground">Required for GDPR</div>
                                            </div>
                                        </div>
                                        <div className="h-2 w-full rounded-full bg-muted">
                                            <div className="h-2 w-4/5 rounded-full bg-green-500" />
                                        </div>
                                        <div className="mt-1 text-xs text-muted-foreground text-right">80% opted in</div>
                                    </div>
                                    <div className="rounded-lg border border-border bg-card p-4">
                                        <div className="flex items-center gap-3 mb-3">
                                            <div className="h-8 w-8 rounded-full bg-blue-500/20 flex items-center justify-center">
                                                <ShieldCheck className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                                            </div>
                                            <div>
                                                <div className="text-sm font-medium">Identity Verification</div>
                                                <div className="text-xs text-muted-foreground">KYC via Stripe Identity</div>
                                            </div>
                                        </div>
                                        <div className="h-2 w-full rounded-full bg-muted">
                                            <div className="h-2 w-3/5 rounded-full bg-blue-500" />
                                        </div>
                                        <div className="mt-1 text-xs text-muted-foreground text-right">Verified</div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </FeatureCard>

                    <FeatureCard>
                        <CardHeader className="pb-3">
                            <CardHeading
                                icon={LayoutDashboard}
                                title="Admin Dashboard"
                                description="Server-rendered admin UI with user, organization, and application management."
                            />
                        </CardHeader>

                        <CardContent>
                            <div className="relative mb-6 sm:mb-0">
                                <div className="absolute -inset-6 [background:radial-gradient(50%_50%_at_75%_50%,transparent,hsl(var(--background))_100%)]"></div>
                                <div className="aspect-[76/59] border rounded-lg bg-card p-4 flex flex-col">
                                    <div className="flex items-center justify-between mb-4">
                                        <div className="text-sm font-medium">Users Overview</div>
                                        <div className="flex gap-1">
                                            <div className="h-2 w-2 rounded-full bg-green-500" />
                                            <div className="text-xs text-muted-foreground">Live</div>
                                        </div>
                                    </div>
                                    <div className="grid grid-cols-3 gap-3 mb-4">
                                        <div className="rounded-md bg-muted/50 p-3 text-center">
                                            <div className="text-lg font-bold">2.4k</div>
                                            <div className="text-xs text-muted-foreground">Users</div>
                                        </div>
                                        <div className="rounded-md bg-muted/50 p-3 text-center">
                                            <div className="text-lg font-bold">48</div>
                                            <div className="text-xs text-muted-foreground">Orgs</div>
                                        </div>
                                        <div className="rounded-md bg-muted/50 p-3 text-center">
                                            <div className="text-lg font-bold">12</div>
                                            <div className="text-xs text-muted-foreground">Apps</div>
                                        </div>
                                    </div>
                                    <div className="flex-1 space-y-2">
                                        {[
                                            { name: 'Active Sessions', value: '1,284', pct: 85 },
                                            { name: 'MFA Enabled', value: '67%', pct: 67 },
                                            { name: 'Auth Success', value: '99.2%', pct: 99 },
                                        ].map((stat) => (
                                            <div key={stat.name} className="flex items-center gap-3">
                                                <div className="w-28 text-xs text-muted-foreground">{stat.name}</div>
                                                <div className="flex-1 h-1.5 rounded-full bg-muted">
                                                    <div
                                                        className="h-1.5 rounded-full bg-brand"
                                                        style={{ width: `${stat.pct}%` }}
                                                    />
                                                </div>
                                                <div className="w-12 text-xs text-right font-mono">{stat.value}</div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        </CardContent>
                    </FeatureCard>

                    <FeatureCard className="p-6 md:col-span-2">
                        <p className="mx-auto my-6 max-w-lg text-balance text-center text-2xl font-semibold">
                            Authentication should be secure, compliant, and simple to integrate -- without compromising on flexibility
                        </p>

                        <div className="flex justify-center gap-6 overflow-hidden">
                            <CircularUI
                                label="Authenticate"
                                circles={[{ pattern: 'border' }, { pattern: 'border' }]}
                            />

                            <CircularUI
                                label="Authorize"
                                circles={[{ pattern: 'none' }, { pattern: 'primary' }]}
                            />

                            <CircularUI
                                label="Protect"
                                circles={[{ pattern: 'blue' }, { pattern: 'none' }]}
                            />

                            <CircularUI
                                label="Audit"
                                circles={[{ pattern: 'primary' }, { pattern: 'none' }]}
                                className="hidden sm:block"
                            />
                        </div>
                    </FeatureCard>
                </div>
            </div>
        </section>
    )
}

interface FeatureCardProps {
    children: ReactNode
    className?: string
}

const FeatureCard = ({ children, className }: FeatureCardProps) => (
    <Card className={cn('group relative rounded-none shadow-zinc-950/5', className)}>
        <CardDecorator />
        {children}
    </Card>
)

const CardDecorator = () => (
    <>
        <span className="border-primary absolute -left-px -top-px block size-2 border-l-2 border-t-2"></span>
        <span className="border-primary absolute -right-px -top-px block size-2 border-r-2 border-t-2"></span>
        <span className="border-primary absolute -bottom-px -left-px block size-2 border-b-2 border-l-2"></span>
        <span className="border-primary absolute -bottom-px -right-px block size-2 border-b-2 border-r-2"></span>
    </>
)

interface CardHeadingProps {
    icon: LucideIcon
    title: string
    description: string
}

const CardHeading = ({ icon: Icon, title, description }: CardHeadingProps) => (
    <div className="p-6">
        <span className="text-muted-foreground flex items-center gap-2">
            <Icon className="size-4" />
            {title}
        </span>
        <p className="mt-8 text-2xl font-semibold">{description}</p>
    </div>
)

interface CircleConfig {
    pattern: 'none' | 'border' | 'primary' | 'blue'
}

interface CircularUIProps {
    label: string
    circles: CircleConfig[]
    className?: string
}

const CircularUI = ({ label, circles, className }: CircularUIProps) => (
    <div className={className}>
        <div className="bg-gradient-to-b from-border size-fit rounded-2xl to-transparent p-px">
            <div className="bg-gradient-to-b from-background to-muted/25 relative flex aspect-square w-fit items-center -space-x-4 rounded-[15px] p-4">
                {circles.map((circle, i) => (
                    <div
                        key={i}
                        className={cn('size-7 rounded-full border sm:size-8', {
                            'border-primary': circle.pattern === 'none',
                            'border-primary bg-[repeating-linear-gradient(-45deg,hsl(var(--border)),hsl(var(--border))_1px,transparent_1px,transparent_4px)]': circle.pattern === 'border',
                            'border-primary bg-background bg-[repeating-linear-gradient(-45deg,hsl(var(--primary)),hsl(var(--primary))_1px,transparent_1px,transparent_4px)]': circle.pattern === 'primary',
                            'bg-background z-1 border-blue-500 bg-[repeating-linear-gradient(-45deg,theme(colors.blue.500),theme(colors.blue.500)_1px,transparent_1px,transparent_4px)]': circle.pattern === 'blue',
                        })}></div>
                ))}
            </div>
        </div>
        <span className="text-muted-foreground mt-1.5 block text-center text-sm">{label}</span>
    </div>
)
