import { Outlet } from "react-router-dom";
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";
import { AppSidebar } from "./app-sidebar";
import { Header } from "./header";

interface DashboardLayoutProps {
  children?: React.ReactNode;
}

/**
 * Main dashboard layout component that provides the overall structure
 * for the dashboard pages including sidebar, header, and main content area.
 * 
 * Features:
 * - Responsive sidebar with collapsible functionality
 * - Header with navigation and user controls
 * - Main content area for page content
 * - Consistent spacing and layout across all dashboard pages
 */
export function DashboardLayout({}: DashboardLayoutProps) {
  return (
    <SidebarProvider>
      <AppSidebar />
      {/* <div className={cn("flex min-h-screen w-full", className)}>
        
      </div> */}
      <SidebarInset>
        {/* Header with sidebar trigger */}
        <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <div className="flex-1">
            <Header />
          </div>
        </header>

        <main className="flex flex-1 flex-col">
          <div className="@container/main flex flex-1 flex-col gap-2">
            <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
              <Outlet />
            </div>
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
