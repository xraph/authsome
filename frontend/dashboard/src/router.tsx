import * as React from "react"
import { createBrowserRouter, Navigate } from "react-router-dom"

import { DashboardLayout } from "@/components/layout/dashboard-layout"
import { SuspenseWrapper } from "@/components/suspense-wrapper"
import { Dashboard } from "@/pages/dashboard"

// Lazy load pages for better performance
const Users = React.lazy(() => import("@/pages/users").then(m => ({ default: m.Users })))
const Sessions = React.lazy(() => import("@/pages/sessions").then(m => ({ default: m.Sessions })))
const Security = React.lazy(() => import("@/pages/security").then(m => ({ default: m.Security })))
const AuditLogs = React.lazy(() => import("@/pages/security/audit").then(m => ({ default: m.AuditLogs })))
const TwoFactorAuth = React.lazy(() => import("@/pages/security/2fa").then(m => ({ default: m.TwoFactorAuth })))
const Plugins = React.lazy(() => import("@/pages/plugins").then(m => ({ default: m.Plugins })))
const Settings = React.lazy(() => import("@/pages/settings").then(m => ({ default: m.Settings })))

/**
 * Router configuration for the dashboard
 */
export const router = createBrowserRouter([
  {
    path: "/",
    element: <DashboardLayout />,
    children: [
      {
        index: true,
        element: <Dashboard />,
      },
      {
        path: "users",
        element: (
          <SuspenseWrapper>
            <Users />
          </SuspenseWrapper>
        ),
      },
      {
        path: "sessions",
        element: (
          <SuspenseWrapper>
            <Sessions />
          </SuspenseWrapper>
        ),
      },
      {
        path: "security",
        element: (
          <SuspenseWrapper>
            <Security />
          </SuspenseWrapper>
        ),
        children: [
          {
            index: true,
            element: <Navigate to="/security/audit" replace />,
          },
          {
            path: "audit",
            element: (
              <SuspenseWrapper>
                <AuditLogs />
              </SuspenseWrapper>
            ),
          },
          {
            path: "2fa",
            element: (
              <SuspenseWrapper>
                <TwoFactorAuth />
              </SuspenseWrapper>
            ),
          },
        ],
      },
      {
        path: "plugins",
        element: (
          <SuspenseWrapper>
            <Plugins />
          </SuspenseWrapper>
        ),
      },
      {
        path: "settings",
        element: (
          <SuspenseWrapper>
            <Settings />
          </SuspenseWrapper>
        ),
      },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/" replace />,
  },
])