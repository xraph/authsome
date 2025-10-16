import * as React from "react"
import { Bell, Search, User, LogOut, Settings } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"

interface HeaderProps {
  className?: string
  title?: string
}

/**
 * Header component with search, notifications, and user menu
 * @param className - Additional CSS classes
 * @param title - Page title
 */
export function Header({ className, title }: HeaderProps) {
  const [searchQuery, setSearchQuery] = React.useState("")
  const [showUserMenu, setShowUserMenu] = React.useState(false)
  const [showNotifications, setShowNotifications] = React.useState(false)

  return (
    <header
      className={cn(
        "flex h-16 items-center justify-between border-b bg-background px-4 lg:px-6",
        className
      )}
    >
      {/* Left side - Title */}
      <div className="flex items-center gap-4">
        {title && (
          <div>
            <h1 className="text-lg font-semibold">{title}</h1>
          </div>
        )}
      </div>

      {/* Center - Search */}
      <div className="flex-1 max-w-md mx-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search users, sessions, logs..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>
      </div>

      {/* Right side - Notifications and user menu */}
      <div className="flex items-center gap-2">
        {/* Notifications */}
        <div className="relative">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setShowNotifications(!showNotifications)}
            className="relative"
          >
            <Bell className="h-5 w-5" />
            <Badge
              variant="destructive"
              className="absolute -top-1 -right-1 h-5 w-5 rounded-full p-0 text-xs"
            >
              3
            </Badge>
            <span className="sr-only">Notifications</span>
          </Button>

          {/* Notifications dropdown */}
          {showNotifications && (
            <div className="absolute right-0 top-full mt-2 w-80 rounded-md border bg-popover p-4 shadow-md z-50">
              <div className="flex items-center justify-between mb-3">
                <h3 className="font-semibold">Notifications</h3>
                <Button variant="ghost" size="sm">
                  Mark all read
                </Button>
              </div>
              <div className="space-y-3">
                <div className="flex gap-3 p-2 rounded-md hover:bg-accent">
                  <div className="h-2 w-2 rounded-full bg-blue-500 mt-2 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-sm font-medium">New user registered</p>
                    <p className="text-xs text-muted-foreground">john@example.com just signed up</p>
                    <p className="text-xs text-muted-foreground">2 minutes ago</p>
                  </div>
                </div>
                <div className="flex gap-3 p-2 rounded-md hover:bg-accent">
                  <div className="h-2 w-2 rounded-full bg-yellow-500 mt-2 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-sm font-medium">Failed login attempt</p>
                    <p className="text-xs text-muted-foreground">Multiple failed attempts from IP 192.168.1.1</p>
                    <p className="text-xs text-muted-foreground">5 minutes ago</p>
                  </div>
                </div>
                <div className="flex gap-3 p-2 rounded-md hover:bg-accent">
                  <div className="h-2 w-2 rounded-full bg-green-500 mt-2 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-sm font-medium">Plugin updated</p>
                    <p className="text-xs text-muted-foreground">Two-Factor Auth plugin updated to v2.1.0</p>
                    <p className="text-xs text-muted-foreground">1 hour ago</p>
                  </div>
                </div>
              </div>
              <div className="mt-3 pt-3 border-t">
                <Button variant="ghost" size="sm" className="w-full">
                  View all notifications
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* User menu */}
        <div className="relative">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setShowUserMenu(!showUserMenu)}
            className="relative"
          >
            <User className="h-5 w-5" />
            <span className="sr-only">User menu</span>
          </Button>

          {/* User menu dropdown */}
          {showUserMenu && (
            <div className="absolute right-0 top-full mt-2 w-56 rounded-md border bg-popover p-2 shadow-md z-50">
              <div className="px-2 py-1.5 mb-2">
                <p className="text-sm font-medium">Admin User</p>
                <p className="text-xs text-muted-foreground">admin@authsome.com</p>
              </div>
              <div className="space-y-1">
                <Button variant="ghost" size="sm" className="w-full justify-start gap-2">
                  <User className="h-4 w-4" />
                  Profile
                </Button>
                <Button variant="ghost" size="sm" className="w-full justify-start gap-2">
                  <Settings className="h-4 w-4" />
                  Settings
                </Button>
                <div className="border-t my-1" />
                <Button variant="ghost" size="sm" className="w-full justify-start gap-2 text-destructive">
                  <LogOut className="h-4 w-4" />
                  Sign out
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Click outside to close dropdowns */}
      {(showNotifications || showUserMenu) && (
        <div
          className="fixed inset-0 z-40"
          onClick={() => {
            setShowNotifications(false)
            setShowUserMenu(false)
          }}
        />
      )}
    </header>
  )
}