import { Children } from "react";
import { cn } from "@/lib/utils";

export function MasonryGrid({ children, className = "", itemClassName = "" }) {
  const items = Children.toArray(children);

  return (
    <div className={cn("columns-1 gap-4 sm:columns-2 lg:columns-3", className)}>
      {items.map((child) => (
        <div key={child.key ?? undefined} className={cn("mb-4 [break-inside:avoid]", itemClassName)}>
          {child}
        </div>
      ))}
    </div>
  );
}
