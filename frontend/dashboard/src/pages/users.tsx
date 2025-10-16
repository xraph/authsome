import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

/**
 * Users page component
 */
export function Users() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Users</h1>
        <p className="text-muted-foreground">
          Manage user accounts and permissions.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>User Management</CardTitle>
          <CardDescription>
            View and manage all user accounts in your system.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">User management interface coming soon...</p>
        </CardContent>
      </Card>
    </div>
  )
}