"use client";

import * as React from "react";

import { cn } from "../lib/utils";

/**
 * A field that looks like one input but holds several controls: a prefix, a
 * suffix, a unit, a country picker.
 *
 * The group carries the border, the radius, the height and the focus ring,
 * and everything inside it is drawn borderless and transparent. That is what
 * makes a select sitting next to a text field read as one control rather
 * than two that happen to be adjacent.
 */
const InputGroup = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { invalid?: boolean }
>(({ className, invalid, ...props }, ref) => (
  <div
    ref={ref}
    data-slot="input-group"
    aria-invalid={invalid || undefined}
    className={cn(
      "flex h-[30px] w-full items-stretch overflow-hidden rounded-md border border-input bg-background text-[13px] transition-[color,box-shadow,border-color]",
      // focus-within rather than focus, because the thing being focused is
      // always a child. Matching Input's ring keeps the two consistent when
      // they sit in the same form.
      "focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/40",
      "has-disabled:cursor-not-allowed has-disabled:opacity-50",
      invalid && "border-destructive focus-within:ring-destructive/30",
      className,
    )}
    {...props}
  />
));
InputGroup.displayName = "InputGroup";

/**
 * A segment attached to one end of the group: a currency symbol, a unit, a
 * country picker, a button.
 *
 * `side` only decides which edge gets the hairline rule, so an addon knows
 * where it sits without the caller styling it.
 */
const InputAddon = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement> & { side?: "start" | "end" }
>(({ className, side = "start", ...props }, ref) => (
  <div
    ref={ref}
    data-slot="input-addon"
    className={cn(
      "flex shrink-0 items-center text-muted-foreground",
      side === "start" ? "border-r border-input" : "border-l border-input",
      className,
    )}
    {...props}
  />
));
InputAddon.displayName = "InputAddon";

/**
 * The text field inside a group. Borderless and transparent, because the
 * group draws the chrome.
 */
const InputGroupInput = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => (
  <input
    ref={ref}
    data-slot="input-group-input"
    className={cn(
      "min-w-0 flex-1 bg-transparent px-2.5 py-1 text-[13px] outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed",
      className,
    )}
    {...props}
  />
));
InputGroupInput.displayName = "InputGroupInput";

export { InputGroup, InputAddon, InputGroupInput };
