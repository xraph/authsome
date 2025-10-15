/**
 * Core types for AuthSome Dashboard
 */

// User types
export interface User {
  id: string
  email: string
  firstName?: string
  lastName?: string
  avatar?: string
  role: UserRole
  status: UserStatus
  emailVerified: boolean
  phoneVerified: boolean
  twoFactorEnabled: boolean
  lastLoginAt?: Date
  createdAt: Date
  updatedAt: Date
  organizationId: string
}

export type UserRole = 'admin' | 'user' | 'moderator'
export type UserStatus = 'active' | 'inactive' | 'suspended' | 'pending'

// Organization types
export interface Organization {
  id: string
  name: string
  slug: string
  domain?: string
  logo?: string
  plan: OrganizationPlan
  status: OrganizationStatus
  settings: OrganizationSettings
  memberCount: number
  createdAt: Date
  updatedAt: Date
}

export type OrganizationPlan = 'free' | 'pro' | 'enterprise'
export type OrganizationStatus = 'active' | 'suspended' | 'trial'

export interface OrganizationSettings {
  allowSignup: boolean
  requireEmailVerification: boolean
  enableTwoFactor: boolean
  sessionTimeout: number
  passwordPolicy: PasswordPolicy
  branding: BrandingSettings
}

export interface PasswordPolicy {
  minLength: number
  requireUppercase: boolean
  requireLowercase: boolean
  requireNumbers: boolean
  requireSymbols: boolean
}

export interface BrandingSettings {
  primaryColor: string
  logo?: string
  favicon?: string
  customCss?: string
}

// Session types
export interface Session {
  id: string
  userId: string
  deviceInfo: DeviceInfo
  ipAddress: string
  userAgent: string
  isActive: boolean
  lastActivityAt: Date
  expiresAt: Date
  createdAt: Date
}

export interface DeviceInfo {
  type: 'desktop' | 'mobile' | 'tablet'
  os: string
  browser: string
  location?: string
}

// Audit types
export interface AuditLog {
  id: string
  userId?: string
  organizationId: string
  action: string
  resource: string
  resourceId?: string
  metadata: Record<string, unknown>
  ipAddress: string
  userAgent: string
  timestamp: Date
}

// Analytics types
export interface DashboardStats {
  totalUsers: number
  activeUsers: number
  newUsersToday: number
  totalSessions: number
  activeSessions: number
  failedLogins: number
  userGrowth: number
  sessionGrowth: number
}

export interface ChartData {
  date: string
  value: number
  label?: string
}

export interface UserGrowthData extends ChartData {
  newUsers: number
  totalUsers: number
}

export interface LoginActivityData extends ChartData {
  successful: number
  failed: number
}

// Plugin types
export interface Plugin {
  id: string
  name: string
  description: string
  version: string
  enabled: boolean
  config: Record<string, unknown>
  dependencies: string[]
}

// API types
export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: string
  message?: string
}

export interface PaginatedResponse<T> extends ApiResponse<T[]> {
  pagination: {
    page: number
    limit: number
    total: number
    totalPages: number
  }
}

export interface ApiError {
  code: string
  message: string
  details?: Record<string, unknown>
}

// Form types
export interface FormField {
  name: string
  label: string
  type: 'text' | 'email' | 'password' | 'select' | 'checkbox' | 'textarea'
  required: boolean
  placeholder?: string
  options?: { label: string; value: string }[]
  validation?: ValidationRule[]
}

export interface ValidationRule {
  type: 'required' | 'email' | 'minLength' | 'maxLength' | 'pattern'
  value?: string | number
  message: string
}

// Navigation types
export interface NavItem {
  title: string
  href: string
  icon?: string
  badge?: string | number
  children?: NavItem[]
  disabled?: boolean
}

// Theme types
export type Theme = 'light' | 'dark' | 'system'

// Dashboard layout types
export interface DashboardConfig {
  title: string
  description?: string
  navigation: NavItem[]
  theme: Theme
  features: {
    userManagement: boolean
    analytics: boolean
    auditLogs: boolean
    organizationSettings: boolean
    pluginManagement: boolean
  }
}

// Filter and search types
export interface FilterOption {
  label: string
  value: string
  count?: number
}

export interface SearchFilters {
  query?: string
  status?: string[]
  role?: string[]
  dateRange?: {
    from: Date
    to: Date
  }
  organizationId?: string
}

export interface SortOption {
  field: string
  direction: 'asc' | 'desc'
}

// Table types
export interface TableColumn<T = unknown> {
  key: keyof T
  label: string
  sortable?: boolean
  width?: string
  render?: (value: unknown, row: T) => React.ReactNode
}

export interface TableProps<T = unknown> {
  data: T[]
  columns: TableColumn<T>[]
  loading?: boolean
  pagination?: {
    page: number
    limit: number
    total: number
    onPageChange: (page: number) => void
  }
  sorting?: {
    field: string
    direction: 'asc' | 'desc'
    onSort: (field: string, direction: 'asc' | 'desc') => void
  }
  selection?: {
    selectedIds: string[]
    onSelectionChange: (ids: string[]) => void
  }
}

// Notification types
export interface Notification {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  title: string
  message?: string
  duration?: number
  action?: {
    label: string
    onClick: () => void
  }
}

// Modal types
export interface ModalProps {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  children: React.ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl'
}

// Dashboard widget types
export interface Widget {
  id: string
  title: string
  type: 'stat' | 'chart' | 'table' | 'custom'
  size: 'sm' | 'md' | 'lg' | 'xl'
  data: unknown
  config: Record<string, unknown>
}

export interface StatWidget extends Widget {
  type: 'stat'
  data: {
    value: number | string
    label: string
    change?: number
    trend?: 'up' | 'down' | 'neutral'
    icon?: string
  }
}

export interface ChartWidget extends Widget {
  type: 'chart'
  data: {
    chartType: 'line' | 'bar' | 'pie' | 'area'
    datasets: ChartData[]
    labels?: string[]
  }
}