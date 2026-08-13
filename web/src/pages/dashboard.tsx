import { useQuery } from "@tanstack/react-query"
import {
  ActivityIcon,
  BellRingIcon,
  CalendarClockIcon,
  SendIcon,
  TriangleAlertIcon,
  WebhookIcon,
} from "lucide-react"
import { api } from "@/lib/api"
import type { Dashboard } from "@/lib/types"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { PageHeader, PageLoading } from "@/components/page"
import { useI18n } from "@/lib/i18n"

const metrics = [
  {
    key: "events_today",
    label: "今日事件",
    description: "今天接收和触发的事件",
    icon: ActivityIcon,
  },
  {
    key: "pending",
    label: "等待投递",
    description: "已入队、等待 Worker 处理",
    icon: BellRingIcon,
  },
  {
    key: "failed",
    label: "死信任务",
    description: "达到最大重试次数",
    icon: TriangleAlertIcon,
  },
  {
    key: "channels",
    label: "通知渠道",
    description: "已配置的渠道连接",
    icon: SendIcon,
  },
  {
    key: "sources",
    label: "Webhook 入口",
    description: "可接收事件的公开入口",
    icon: WebhookIcon,
  },
  {
    key: "schedules",
    label: "启用定时",
    description: "正在运行的定时任务",
    icon: CalendarClockIcon,
  },
] as const
export function DashboardPage() {
  const { t } = useI18n()
  const query = useQuery({
    queryKey: ["dashboard"],
    queryFn: () => api<Dashboard>("/dashboard"),
    refetchInterval: 15000,
  })
  return (
    <>
      <PageHeader
        title="消息中枢"
        description="从入口、规则到最终投递，一眼掌握当前运行状态。"
      />
      {query.isLoading ? (
        <PageLoading />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {metrics.map(({ key, label, description, icon: Icon }) => (
            <Card key={key}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>{t(label)}</CardTitle>
                  <Icon className="size-4 text-muted-foreground" />
                </div>
                <CardDescription>{t(description)}</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="font-heading text-3xl font-semibold tabular-nums">
                  {query.data?.[key] ?? 0}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </>
  )
}
