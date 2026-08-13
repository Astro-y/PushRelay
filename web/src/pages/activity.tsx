import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { RotateCcwIcon } from "lucide-react"
import { toast } from "sonner"
import { api } from "@/lib/api"
import type { Delivery, EventItem } from "@/lib/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { PageHeader, PageLoading, formatTime } from "@/components/page"
import { ResourceTable } from "@/components/resource-table"
import { useI18n } from "@/lib/i18n"

export function ActivityPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const events = useQuery({
    queryKey: ["events"],
    queryFn: () => api<EventItem[]>("/events?limit=100"),
    refetchInterval: 10000,
  })
  const deliveries = useQuery({
    queryKey: ["deliveries"],
    queryFn: () => api<Delivery[]>("/deliveries?limit=100"),
    refetchInterval: 5000,
  })
  const retry = useMutation({
    mutationFn: (id: string) =>
      api(`/deliveries/${id}/retry`, { method: "POST" }),
    onSuccess: () => {
      toast.success(t("已重新加入队列"))
      client.invalidateQueries({ queryKey: ["deliveries"] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  return (
    <>
      <PageHeader
        title="执行记录"
        description="追踪每个事件的规则命中、渠道投递、重试次数和最终状态。"
      />
      <Tabs defaultValue="deliveries">
        <TabsList>
          <TabsTrigger value="deliveries">{t("投递任务")}</TabsTrigger>
          <TabsTrigger value="events">{t("事件")}</TabsTrigger>
        </TabsList>
        <TabsContent value="deliveries">
          {deliveries.isLoading ? (
            <PageLoading />
          ) : (
            <Card>
              <CardContent>
                <ResourceTable
                  items={deliveries.data ?? []}
                  columns={[
                    {
                      key: "target",
                      label: "目标",
                      render: (item) => (
                        <div className="flex flex-col gap-1">
                          <span className="font-medium">
                            {item.channel_name}
                          </span>
                          <span className="text-xs text-muted-foreground">
                            {item.template_name}
                          </span>
                        </div>
                      ),
                    },
                    {
                      key: "status",
                      label: "状态",
                      render: (item) => (
                        <Badge
                          variant={
                            item.status === "success"
                              ? "default"
                              : item.status === "dead"
                                ? "destructive"
                                : "secondary"
                          }
                        >
                          {t(item.status)}
                        </Badge>
                      ),
                    },
                    {
                      key: "attempts",
                      label: "尝试",
                      render: (item) => (
                        <span className="tabular-nums">
                          {item.attempts} / 5
                        </span>
                      ),
                    },
                    {
                      key: "time",
                      label: "创建时间",
                      render: (item) => formatTime(item.created_at),
                    },
                    {
                      key: "error",
                      label: "最近错误",
                      render: (item) => (
                        <span className="block max-w-sm truncate text-xs text-destructive">
                          {item.last_error ?? "—"}
                        </span>
                      ),
                    },
                    {
                      key: "actions",
                      label: "操作",
                      render: (item) => (
                        <Button
                          size="sm"
                          variant="outline"
                          disabled={item.status !== "dead"}
                          onClick={() => retry.mutate(item.id)}
                        >
                          <RotateCcwIcon data-icon="inline-start" />
                          {t("重投")}
                        </Button>
                      ),
                    },
                  ]}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>
        <TabsContent value="events">
          {events.isLoading ? (
            <PageLoading />
          ) : (
            <Card>
              <CardContent>
                <ResourceTable
                  items={events.data ?? []}
                  columns={[
                    {
                      key: "id",
                      label: "事件 ID",
                      render: (item) => (
                        <code className="text-xs">{item.id.slice(0, 12)}…</code>
                      ),
                    },
                    {
                      key: "trigger",
                      label: "触发方式",
                      render: (item) => (
                        <Badge variant="secondary">
                          {t(item.trigger_type)}
                        </Badge>
                      ),
                    },
                    {
                      key: "method",
                      label: "方法",
                      render: (item) => item.method,
                    },
                    {
                      key: "rules",
                      label: "命中规则",
                      render: (item) => (
                        <span className="tabular-nums">
                          {item.matched_rules}
                        </span>
                      ),
                    },
                    {
                      key: "policy",
                      label: "载荷策略",
                      render: (item) => (
                        <Badge variant="outline">
                          {t(item.payload_policy)}
                        </Badge>
                      ),
                    },
                    {
                      key: "time",
                      label: "接收时间",
                      render: (item) => formatTime(item.created_at),
                    },
                  ]}
                />
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </>
  )
}
