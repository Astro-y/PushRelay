import { cn } from "@/lib/utils"

export function BrandMark({
  className,
  decorative = true,
}: {
  className?: string
  decorative?: boolean
}) {
  return (
    <img
      src="/brand/pushrelay-app-icon.png"
      alt={decorative ? "" : "PushRelay"}
      aria-hidden={decorative || undefined}
      className={cn("shrink-0", className)}
    />
  )
}
