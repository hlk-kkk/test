import { cn } from "@/lib/utils";
import { getAspectRatioCss } from "./aspectRatio";

export function ImageSkeleton({ aspectRatio, className = "" }) {
  const cssRatio = getAspectRatioCss(aspectRatio);

  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-xl bg-secondary/50",
        // Amber 边框脉动效果
        "border border-primary/20 animate-pulse-glow",
        className
      )}
      style={{ aspectRatio: cssRatio }}
    >
      {/* 水平方向 shimmer 动画，更亮的渐变 */}
      <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/15 to-transparent -translate-x-full animate-[shimmer_1.5s_infinite]" />

      {/* 内部微弱发光效果 */}
      <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-transparent" />
    </div>
  );
}
