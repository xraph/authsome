import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

/**
 * Plugins page component
 */
export function Plugins() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Plugins</h1>
        <p className="text-muted-foreground">
          Manage authentication plugins and extensions.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Plugin Management</CardTitle>
          <CardDescription>
            Configure and manage authentication plugins.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">Plugin management interface coming soon...</p>
        </CardContent>
      </Card>
    </div>
  )
}