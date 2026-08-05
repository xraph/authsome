import * as React from "react";

import { cn } from "../lib/utils";

// A type alias rather than an empty interface: an interface that adds no
// members is just a second name for its supertype, and declaration merging
// would let an unrelated file silently widen it.
export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-[30px] w-full rounded-md border border-input bg-background px-2.5 py-1 text-[13px] ring-offset-background file:border-0 file:bg-transparent file:text-[13px] file:font-medium file:text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Input.displayName = "Input";

export { Input };
