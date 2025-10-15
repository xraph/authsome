import * as React from "react"
import { Users, Activity, Shield, AlertTriangle } from "lucide-react"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import type { DashboardStats } from "@/types"

/**
 * Mock dashboard statistics
 */
const mockStats: DashboardStats = {
  totalUsers: 1247,
  activeUsers: 892,
  newUsersToday: 23,
  totalSessions: 1456,
  activeSessions: 234,
  failedLogins: 12,
  userGrowth: 12.5,
  sessionGrowth: 8.3,
}

/**
 * Stat card component
 * @param title - Card title
 * @param value - Main value to display
 * @param description - Card description
 * @param icon - Icon component
 * @param trend - Growth percentage
 * @param badge - Optional badge text
 */
function StatCard({
  title,
  value,
  description,
  icon: Icon,
  trend,
  badge,
}: {
  title: string
  value: string | number
  description: string
  icon: React.ComponentType<{ className?: string }>
  trend?: number
  badge?: string
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <div className="flex items-center gap-2">
          {badge && <Badge variant="secondary">{badge}</Badge>}
          <Icon className="h-4 w-4 text-muted-foreground" />
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value.toLocaleString()}</div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>{description}</span>
          {trend !== undefined && (
            <span className={trend >= 0 ? "text-green-600" : "text-red-600"}>
              {trend >= 0 ? "+" : ""}{trend}%
            </span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * Recent activity item component
 */
function ActivityItem({
  title,
  description,
  time,
  type,
}: {
  title: string
  description: string
  time: string
  type: "success" | "warning" | "error" | "info"
}) {
  const typeColors = {
    success: "bg-green-500",
    warning: "bg-yellow-500",
    error: "bg-red-500",
    info: "bg-blue-500",
  }

  return (
    <div className="flex gap-3 p-3 rounded-lg hover:bg-accent/50 transition-colors">
      <div className={`h-2 w-2 rounded-full mt-2 flex-shrink-0 ${typeColors[type]}`} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{title}</p>
        <p className="text-xs text-muted-foreground truncate">{description}</p>
        <p className="text-xs text-muted-foreground mt-1">{time}</p>
      </div>
    </div>
  )
}

/**
 * Dashboard page component
 */
export function Dashboard() {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Welcome to your AuthSome dashboard. Here's what's happening with your authentication system.
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Users"
          value={mockStats.totalUsers}
          description="All registered users"
          icon={Users}
          trend={mockStats.userGrowth}
        />
        <StatCard
          title="Active Users"
          value={mockStats.activeUsers}
          description="Users active in last 30 days"
          icon={Activity}
          trend={5.2}
        />
        <StatCard
          title="Active Sessions"
          value={mockStats.activeSessions}
          description="Currently logged in"
          icon={Shield}
          trend={mockStats.sessionGrowth}
        />
        <StatCard
          title="Failed Logins"
          value={mockStats.failedLogins}
          description="In the last 24 hours"
          icon={AlertTriangle}
          trend={-15.3}
          badge="24h"
        />
      </div>

      {/* Content Grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent Activity */}
        <Card>
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
            <CardDescription>
              Latest authentication events and system activities
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            <ActivityItem
              title="New user registration"
              description="john.doe@example.com registered successfully"
              time="2 minutes ago"
              type="success"
            />
            <ActivityItem
              title="Failed login attempt"
              description="Multiple failed attempts from IP 192.168.1.100"
              time="5 minutes ago"
              type="warning"
            />
            <ActivityItem
              title="Password reset"
              description="sarah.wilson@example.com requested password reset"
              time="12 minutes ago"
              type="info"
            />
            <ActivityItem
              title="Account locked"
              description="Account mike.johnson@example.com locked due to suspicious activity"
              time="25 minutes ago"
              type="error"
            />
            <ActivityItem
              title="Two-factor enabled"
              description="emma.davis@example.com enabled 2FA"
              time="1 hour ago"
              type="success"
            />
          </CardContent>
        </Card>

        {/* System Status */}
        <Card>
          <CardHeader>
            <CardTitle>System Status</CardTitle>
            <CardDescription>
              Current status of authentication services and plugins
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-green-500" />
                <span className="text-sm font-medium">Authentication Service</span>
              </div>
              <Badge variant="success">Operational</Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-green-500" />
                <span className="text-sm font-medium">Session Management</span>
              </div>
              <Badge variant="success">Operational</Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-yellow-500" />
                <span className="text-sm font-medium">Email Service</span>
              </div>
              <Badge variant="warning">Degraded</Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-green-500" />
                <span className="text-sm font-medium">Two-Factor Auth</span>
              </div>
              <Badge variant="success">Operational</Badge>
            </div>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-green-500" />
                <span className="text-sm font-medium">OAuth Providers</span>
              </div>
              <Badge variant="success">Operational</Badge>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card>
        <CardHeader>
          <CardTitle>Quick Actions</CardTitle>
          <CardDescription>
            Common tasks and shortcuts for managing your authentication system
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-accent/50 transition-colors cursor-pointer">
              <Users className="h-8 w-8 mb-2 text-primary" />
              <span className="text-sm font-medium">Manage Users</span>
              <span className="text-xs text-muted-foreground text-center">
                View and manage user accounts
              </span>
            </div>
            <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-accent/50 transition-colors cursor-pointer">
              <Shield className="h-8 w-8 mb-2 text-primary" />
              <span className="text-sm font-medium">Security Settings</span>
              <span className="text-xs text-muted-foreground text-center">
                Configure security policies
              </span>
            </div>
            <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-accent/50 transition-colors cursor-pointer">
              <Activity className="h-8 w-8 mb-2 text-primary" />
              <span className="text-sm font-medium">View Audit Logs</span>
              <span className="text-xs text-muted-foreground text-center">
                Review system activity
              </span>
            </div>
            <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-accent/50 transition-colors cursor-pointer">
              <AlertTriangle className="h-8 w-8 mb-2 text-primary" />
              <span className="text-sm font-medium">System Alerts</span>
              <span className="text-xs text-muted-foreground text-center">
                Check security alerts
              </span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}