import * as React from "react"
import { Link, useLocation } from "react-router-dom"
import { 
  LayoutDashboard, 
  Users, 
  Settings, 
  Shield, 
  Activity,
  Puzzle,
  ChevronRight
} from "lucide-react"

import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarMenuBadge,
} from "@/components/ui/sidebar"
import type { NavItem } from "@/types"

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
 * Navigation item component using Shadcn sidebar patterns
 * @param item - Navigation item data
 */
function NavItem({ item }: { item: NavItem }) {
  const location = useLocation()
  const [expanded, setExpanded] = React.useState(false)
  const isActive = location.pathname === item.href
  const hasChildren = item.children && item.children.length > 0
  const Icon = item.icon ? iconMap[item.icon as keyof typeof iconMap] : null

  // Check if any child is active
  const isChildActive = hasChildren && item.children?.some(child => 
    location.pathname === child.href
  )

  // Auto-expand if child is active
  React.useEffect(() => {
    if (isChildActive) {
      setExpanded(true)
    }
  }, [isChildActive])

  if (hasChildren) {
    return (
      <SidebarMenuItem>
        <SidebarMenuButton
          onClick={() => setExpanded(!expanded)}
          isActive={isActive || isChildActive}
          className="w-full"
        >
          {Icon && <Icon className="h-4 w-4" />}
          <span>{item.title}</span>
          {item.badge && (
            <SidebarMenuBadge>{item.badge}</SidebarMenuBadge>
          )}
          <ChevronRight 
            className={cn(
              "ml-auto h-4 w-4 transition-transform",
              expanded && "rotate-90"
            )} 
          />
        </SidebarMenuButton>
        {expanded && (
          <SidebarMenuSub>
            {item.children?.map((child) => {
              const ChildIcon = child.icon ? iconMap[child.icon as keyof typeof iconMap] : null
              const isChildItemActive = location.pathname === child.href
              
              return (
                <SidebarMenuSubItem key={child.href}>
                  <SidebarMenuSubButton 
                    asChild 
                    isActive={isChildItemActive}
                  >
                    <Link to={child.href}>
                      {ChildIcon && <ChildIcon className="h-4 w-4" />}
                      <span>{child.title}</span>
                      {child.badge && (
                        <Badge variant="secondary" className="ml-auto text-xs">
                          {child.badge}
                        </Badge>
                      )}
                    </Link>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              )
            })}
          </SidebarMenuSub>
        )}
      </SidebarMenuItem>
    )
  }

  return (
    <SidebarMenuItem>
      <SidebarMenuButton asChild isActive={isActive}>
        <Link to={item.href}>
          {Icon && <Icon className="h-4 w-4" />}
          <span>{item.title}</span>
          {item.badge && (
            <SidebarMenuBadge>{item.badge}</SidebarMenuBadge>
          )}
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

/**
 * Application sidebar component using Shadcn UI patterns
 * Maintains all existing functionality while using modern UI components
 */
export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar {...props}>
      <SidebarHeader>
        <div className="flex h-12 items-center px-4">
          <Link to="/" className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Shield className="h-4 w-4" />
            </div>
            <div className="flex flex-col">
              <span className="text-sm font-semibold">AuthSome</span>
              <span className="text-xs text-muted-foreground">Dashboard</span>
            </div>
          </Link>
        </div>
      </SidebarHeader>
      
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Navigation</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {navigationItems.map((item) => (
                <NavItem key={item.href} item={item} />
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      
      <SidebarFooter>
        <div className="flex items-center gap-2 px-4 py-2 text-xs text-muted-foreground">
          <span>v1.0.0</span>
          <span>•</span>
          <span>AuthSome</span>
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}