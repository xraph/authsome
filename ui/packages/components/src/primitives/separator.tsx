import * as React from "react";
import * as SeparatorPrimitive from "@radix-ui/react-separator";

import { cn } from "../lib/utils";

const Separator = React.forwardRef<
  React.ComponentRef<typeof SeparatorPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof SeparatorPrimitive.Root>
>(
  (
    { className, orientation = "horizontal", decorative = true, ...props },
    ref,
  ) => (
    <SeparatorPrimitive.Root
      ref={ref}
      decorative={decorative}
      orientation={orientation}
      className={cn(
        "shrink-0 bg-border",
        orientation === "horizontal" ? "h-[1px] w-full" : "h-full w-[1px]",
        className,
      )}
      {...props}
    />
  ),
);
Separator.displayName = SeparatorPrimitive.Root.displayName;

export interface LabelledSeparatorProps
  extends React.HTMLAttributes<HTMLDivElement> {
  /** The text sitting on the rule, e.g. "or with email". */
  children: React.ReactNode;
}

/**
 * A rule with a label centred on it.
 *
 * The label is flanked by two flex-grown rules rather than laid over one
 * full-width rule with an opaque background behind the text. The overlay
 * version has to know the colour it sits on, which is why the old divider
 * hardcoded `bg-card` and broke the moment a form was rendered borderless on
 * a page background. This version never needs to know.
 */
const LabelledSeparator = React.forwardRef<
  HTMLDivElement,
  LabelledSeparatorProps
>(({ className, children, ...props }, ref) => (
  <div
    ref={ref}
    className={cn("flex items-center gap-3", className)}
    {...props}
  >
    <Separator className="flex-1" />
    <span className="shrink-0 text-[13px] text-muted-foreground">
      {children}
    </span>
    <Separator className="flex-1" />
  </div>
));
LabelledSeparator.displayName = "LabelledSeparator";

export { Separator, LabelledSeparator };
