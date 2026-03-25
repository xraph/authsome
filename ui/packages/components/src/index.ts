/** @authsome/ui-components — Styled authentication UI components built on shadcn/ui. */

// Primitives
export { Button, type ButtonProps, buttonVariants } from "./primitives/button";
export {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from "./primitives/card";
export { Input, type InputProps } from "./primitives/input";
export { Label } from "./primitives/label";
export { Alert, AlertDescription } from "./primitives/alert";
export {
  Avatar,
  AvatarImage,
  AvatarFallback,
} from "./primitives/avatar";
export {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "./primitives/dropdown-menu";
export { Separator } from "./primitives/separator";
export { Badge, type BadgeProps, badgeVariants } from "./primitives/badge";
export { Skeleton } from "./primitives/skeleton";
export { Checkbox } from "./primitives/checkbox";
export {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
  InputOTPSeparator,
} from "./primitives/otp-input";
export {
  Dialog,
  DialogPortal,
  DialogOverlay,
  DialogClose,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from "./primitives/dialog";
export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectLabel,
  SelectItem,
  SelectSeparator,
} from "./primitives/select";
export { Tabs, TabsList, TabsTrigger, TabsContent } from "./primitives/tabs";
export {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
} from "./primitives/tooltip";
export { Switch } from "./primitives/switch";
export {
  Sheet,
  SheetPortal,
  SheetOverlay,
  SheetTrigger,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
} from "./primitives/sheet";

// Shared components
export { AuthCard, type AuthCardProps, type AuthCardAlign, type AuthCardVariant } from "./components/auth-card";
export { ErrorDisplay, type ErrorDisplayProps } from "./components/error-display";
export { LoadingSpinner, type LoadingSpinnerProps } from "./components/loading-spinner";
export { PasswordInput } from "./components/password-input";
export { SocialButtons, OrDivider, type SocialButtonsProps, type SocialProvider, type SocialButtonLayout } from "./components/social-buttons";
export { openOAuthPopup, handleSocialLogin } from "./lib/social-login";

// Self-routing auth components (Clerk-style)
export { SignIn, type SignInProps } from "./components/sign-in";
export { SignUp, type SignUpProps } from "./components/sign-up";

// Re-export control components from @authsome/ui-react for convenience
export { SignedIn, SignedOut, Protect } from "@authsome/ui-react";
export type { SignedInProps, SignedOutProps, ProtectProps } from "@authsome/ui-react";

// Auth form components (lower-level)
export { SignInForm, type SignInFormComponentProps } from "./components/sign-in-form";
export { SignUpForm, type SignUpFormComponentProps } from "./components/sign-up-form";
export { ForgotPasswordForm, type ForgotPasswordFormProps } from "./components/forgot-password-form";
export { ResetPasswordForm, type ResetPasswordFormProps } from "./components/reset-password-form";
export { MagicLinkForm, type MagicLinkFormProps } from "./components/magic-link-form";
export { WaitlistForm, type WaitlistFormProps } from "./components/waitlist-form";
export { MFAChallengeForm as MFAChallengeFormStyled, type MFAChallengeFormStyledProps } from "./components/mfa-challenge-form";
export { ChangePasswordForm, type ChangePasswordFormProps } from "./components/change-password-form";
export { EmailVerificationForm, type EmailVerificationFormProps } from "./components/email-verification-form";

// Passkey components
export { PasskeyLoginButton, type PasskeyLoginButtonProps } from "./components/passkey-login-button";
export { PasskeyRegisterButton, type PasskeyRegisterButtonProps } from "./components/passkey-register-button";
export { PasskeyList, type PasskeyListProps } from "./components/passkey-list";

// Device & session management components
export { DeviceList, type DeviceListProps } from "./components/device-list";
export { SessionList, type SessionListProps } from "./components/session-list";
export { DeviceAuthorizationForm, type DeviceAuthorizationFormProps } from "./components/device-authorization-form";

// User components
export { UserAvatar, type UserAvatarProps } from "./components/user-avatar";
export { UserButton, type UserButtonProps, type UserButtonMenuItem } from "./components/user-button";
export { UserProfileCard, type UserProfileCardProps } from "./components/user-profile-card";
export { OrgSwitcher, type OrgSwitcherProps } from "./components/org-switcher";
export { StyledAuthGuard, type StyledAuthGuardProps } from "./components/styled-auth-guard";

// Utilities
export { cn } from "./lib/utils";
