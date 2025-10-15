import * as React from "react"
import { Link, useLocation } from "react-router-dom"
import { 
  LayoutDashboard, 
  Users, 
  Settings, 
  Shield, 
  Activity,
  Puzzle,
  ChevronDown,
  ChevronRight
} from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import type { NavItem } from "@/types"

interface SidebarProps {
  className?: string
  collapsed?: boolean
}

/**
 * Navigation items for the sidebar
 */
const navigationItems: NavItem[] = [
  {
    title: "Dashboard",
    href: "/",
    icon: "LayoutDashboard",
  },
  {
    title: "Users",
    href: "/users",
    icon: "Users",
    badge: "12",
  },
  {
    title: "Sessions",
    href: "/sessions",
    icon: "Activity",
  },
  {
    title: "Security",
    href: "/security",
    icon: "Shield",
    children: [
      {
        title: "Audit Logs",
        href: "/security/audit",
        icon: "Activity",
      },
      {
        title: "Two-Factor Auth",
        href: "/security/2fa",
        icon: "Shield",
      },
    ],
  },
  {
    title: "Plugins",
    href: "/plugins",
    icon: "Puzzle",
  },
  {
    title: "Settings",
    href: "/settings",
    icon: "Settings",
  },
]

/**
 * Icon component mapper
 */
const iconMap = {
  LayoutDashboard,
  Users,
  Settings,
  Shield,
  Activity,
  Puzzle,
}

/**
 * Navigation item component
 * @param item - Navigation item data
 * @param collapsed - Whether sidebar is collapsed
 * @param level - Nesting level for indentation
 */
function NavItem({ 
  item, 
  collapsed = false, 
  level = 0 
}: { 
  item: NavItem
  collapsed?: boolean
  level?: number
}) {
  const location = useLocation()
  const [expanded, setExpanded] = React.useState(false)
  const isActive = location.pathname === item.href
  const hasChildren = item.children && item.children.length > 0
  const Icon = item.icon ? iconMap[item.icon as keyof typeof iconMap] : null

  const handleClick = () => {
    if (hasChildren) {
      setExpanded(!expanded)
    }
  }

  return (
    <div>
      <Button
        variant={isActive ? "secondary" : "ghost"}
        className={cn(
          "w-full justify-start gap-2 h-9",
          level > 0 && "ml-4 w-[calc(100%-1rem)]",
          collapsed && "px-2"
        )}
        onClick={handleClick}
        asChild={!hasChildren}
      >
        {hasChildren ? (
          <div className="flex items-center justify-between w-full">
            <div className="flex items-center gap-2">
              {Icon && <Icon className="h-4 w-4" />}
              {!collapsed && <span>{item.title}</span>}
            </div>
            {!collapsed && hasChildren && (
              <div className="flex items-center gap-1">
                {item.badge && (
                  <Badge variant="secondary" className="text-xs">
                    {item.badge}
                  </Badge>
                )}
                {expanded ? (
                  <ChevronDown className="h-3 w-3" />
                ) : (
                  <ChevronRight className="h-3 w-3" />
                )}
              </div>
            )}
          </div>
        ) : (
          <Link to={item.href} className="flex items-center justify-between w-full">
            <div className="flex items-center gap-2">
              {Icon && <Icon className="h-4 w-4" />}
              {!collapsed && <span>{item.title}</span>}
            </div>
            {!collapsed && item.badge && (
              <Badge variant="secondary" className="text-xs">
                {item.badge}
              </Badge>
            )}
          </Link>
        )}
      </Button>
      
      {hasChildren && expanded && !collapsed && (
        <div className="mt-1 space-y-1">
          {item.children?.map((child) => (
            <NavItem
              key={child.href}
              item={child}
              collapsed={collapsed}
              level={level + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

/**
 * Sidebar component for navigation
 * @param className - Additional CSS classes
 * @param collapsed - Whether sidebar is collapsed
 */
export function Sidebar({ className, collapsed = false }: SidebarProps) {
  return (
    <div
      className={cn(
        "flex flex-col border-r bg-background",
        collapsed ? "w-16" : "w-64",
        "transition-all duration-300 ease-in-out",
        className
      )}
    >
      {/* Logo/Brand */}
      <div className="flex h-16 items-center border-b px-4">
        <Link to="/" className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Shield className="h-4 w-4" />
          </div>
          {!collapsed && (
            <div className="flex flex-col">
              <span className="text-sm font-semibold">AuthSome</span>
              <span className="text-xs text-muted-foreground">Dashboard</span>
            </div>
          )}
        </Link>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 p-4">
        {navigationItems.map((item) => (
          <NavItem
            key={item.href}
            item={item}
            collapsed={collapsed}
          />
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t p-4">
        <div className={cn(
          "flex items-center gap-2 text-xs text-muted-foreground",
          collapsed && "justify-center"
        )}>
          {!collapsed && (
            <>
              <span>v1.0.0</span>
              <span>•</span>
              <span>AuthSome</span>
            </>
          )}
        </div>
      </div>
    </div>
  )
}