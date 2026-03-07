import * as React from "react";
import { LogOut, Settings, User as UserIcon } from "lucide-react";
import { useAuth, useUser } from "@authsome/ui-react";
import { cn } from "../lib/utils";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "../primitives/dropdown-menu";
import { UserAvatar } from "./user-avatar";

/** A simple menu-item descriptor rendered as a link inside the dropdown. */
export interface UserButtonMenuItem {
  /** Display label for the menu item. */
  label: string;
  /** URL the menu item links to. */
  href: string;
}

export interface UserButtonProps {
  /** URL to redirect after sign out. */
  afterSignOutUrl?: string;
  /** URL for the manage account / profile page. */
  profileUrl?: string;
  /** URL for the settings page. */
  settingsUrl?: string;
  /**
   * Additional menu items rendered before the sign-out action.
   * Pass an array of `{ label, href }` objects for simple links,
   * or any valid `ReactNode` for full customisation.
   */
  menuItems?: UserButtonMenuItem[] | React.ReactNode;
  /** Additional CSS class names. */
  className?: string;
}

export function UserButton({
  afterSignOutUrl,
  profileUrl,
  settingsUrl,
  menuItems,
  className,
}: UserButtonProps) {
  const { signOut } = useAuth();
  const { user } = useUser();

  const handleSignOut = React.useCallback(async () => {
    await signOut();
    if (afterSignOutUrl) {
      window.location.href = afterSignOutUrl;
    }
  }, [signOut, afterSignOutUrl]);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            "rounded-full outline-none transition-opacity hover:opacity-80 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
            className,
          )}
        >
          <UserAvatar />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-60">
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col space-y-1">
            {user?.name && (
              <p className="text-sm font-medium leading-none">{user.name}</p>
            )}
            {user?.email && (
              <p className="text-xs text-muted-foreground">{user.email}</p>
            )}
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {profileUrl && (
          <DropdownMenuItem asChild>
            <a href={profileUrl}>
              <UserIcon className="mr-2 h-4 w-4" />
              Profile
            </a>
          </DropdownMenuItem>
        )}
        {settingsUrl && (
          <DropdownMenuItem asChild>
            <a href={settingsUrl}>
              <Settings className="mr-2 h-4 w-4" />
              Settings
            </a>
          </DropdownMenuItem>
        )}
        {Array.isArray(menuItems)
          ? menuItems.map((item) =>
              typeof item === "object" &&
              item !== null &&
              "label" in item &&
              "href" in item ? (
                <DropdownMenuItem key={(item as UserButtonMenuItem).href} asChild>
                  <a href={(item as UserButtonMenuItem).href}>
                    {(item as UserButtonMenuItem).label}
                  </a>
                </DropdownMenuItem>
              ) : (
                item
              ),
            )
          : menuItems}
        {(profileUrl || settingsUrl || menuItems) && (
          <DropdownMenuSeparator />
        )}
        <DropdownMenuItem
          onClick={() => {
            void handleSignOut();
          }}
        >
          <LogOut className="mr-2 h-4 w-4" />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
