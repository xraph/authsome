import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

/**
 * Two-Factor Authentication page component
 */
export function TwoFactorAuth() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Two-Factor Authentication</h1>
        <p className="text-muted-foreground">
          Configure and manage 2FA settings.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>2FA Configuration</CardTitle>
          <CardDescription>
            Set up and manage two-factor authentication for enhanced security.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">2FA configuration interface coming soon...</p>
        </CardContent>
      </Card>
    </div>
  )
}