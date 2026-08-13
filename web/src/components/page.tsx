import type { ReactNode } from "react"
import { Skeleton } from "@/components/ui/skeleton"
import { useI18n } from "@/lib/i18n"

export function PageHeader({
  title,
  description,
  action,
}: {
  title: string
  description: string
  action?: ReactNode
}) {
  const { t } = useI18n()
  return (
    <header className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex flex-col gap-1">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          {t(title)}
        </h1>
        <p className="max-w-2xl text-sm text-muted-foreground">
          {t(description)}
        </p>
      </div>
      {action}
    </header>
  )
}

export function PageLoading() {
  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {Array.from({ length: 6 }).map((_, index) => (
        <Skeleton className="h-32" key={index} />
      ))}
    </div>
  )
}

export function formatTime(unix?: number) {
  if (!unix) return "—"
  return new Intl.DateTimeFormat(document.documentElement.lang || "zh-CN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(unix * 1000))
}
