import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { PlusIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"
import { api, jsonBody } from "@/lib/api"
import type { Schedule, TargetGroup } from "@/lib/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { CrudDialog } from "@/components/crud-dialog"
import { PageHeader, PageLoading, formatTime } from "@/components/page"
import { ResourceTable } from "@/components/resource-table"
import { useI18n } from "@/lib/i18n"

const kinds = [
  { label: "每天", value: "daily" },
  { label: "每周", value: "weekly" },
  { label: "每月", value: "monthly" },
  { label: "每年", value: "yearly" },
  { label: "5 段 Cron", value: "cron" },
  { label: "单次", value: "once" },
]
export function SchedulesPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [groupID, setGroupID] = useState("")
  const [kind, setKind] = useState("daily")
  const [zone, setZone] = useState("")
  const [timeValue, setTimeValue] = useState("09:00")
  const [interval, setInterval] = useState(1)
  const [day, setDay] = useState(1)
  const [month, setMonth] = useState(1)
  const [cron, setCron] = useState("0 9 * * *")
  const [runAt, setRunAt] = useState("")
  const [payload, setPayload] = useState('{\n  "message": "定时提醒"\n}')
  const schedules = useQuery({
    queryKey: ["schedules"],
    queryFn: () => api<Schedule[]>("/schedules"),
  })
  const groups = useQuery({
    queryKey: ["groups"],
    queryFn: () => api<TargetGroup[]>("/target-groups"),
  })
  const recurrence = () =>
    kind === "cron"
      ? { kind, cron }
      : kind === "once"
        ? { kind, run_at: new Date(runAt).toISOString() }
        : {
            kind,
            interval,
            time: timeValue,
            ...(kind === "weekly" ? { weekdays: [1] } : {}),
            ...(kind === "monthly" ? { day } : {}),
            ...(kind === "yearly" ? { month, day } : {}),
          }
  const save = useMutation({
    mutationFn: () =>
      api("/schedules", {
        method: "POST",
        body: jsonBody({
          name,
          target_group_id: groupID,
          timezone: zone,
          enabled: true,
          recurrence: recurrence(),
          payload: JSON.parse(payload),
        }),
      }),
    onSuccess: () => {
      toast.success(t("定时任务已创建"))
      setOpen(false)
      client.invalidateQueries({ queryKey: ["schedules"] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const remove = useMutation({
    mutationFn: (id: string) => api(`/schedules/${id}`, { method: "DELETE" }),
    onSuccess: () => client.invalidateQueries({ queryKey: ["schedules"] }),
  })
  const groupItems = (groups.data ?? []).map((x) => ({
    label: x.name,
    value: x.id,
  }))
  const localizedKinds = kinds.map((item) => ({
    ...item,
    label: t(item.label),
  }))
  return (
    <>
      <PageHeader
        title="定时任务"
        description="所有时间按任务自己的 IANA 时区计算，数据库统一保存 UTC，精确到分钟。"
        action={
          <Button onClick={() => setOpen(true)}>
            <PlusIcon data-icon="inline-start" />
            {t("新建定时任务")}
          </Button>
        }
      />
      {schedules.isLoading ? (
        <PageLoading />
      ) : (
        <Card>
          <CardContent>
            <ResourceTable
              items={schedules.data ?? []}
              columns={[
                {
                  key: "name",
                  label: "名称",
                  render: (item) => (
                    <span className="font-medium">{item.name}</span>
                  ),
                },
                {
                  key: "kind",
                  label: "周期",
                  render: (item) => (
                    <Badge variant="secondary">{t(item.recurrence.kind)}</Badge>
                  ),
                },
                {
                  key: "zone",
                  label: "时区",
                  render: (item) => (
                    <code className="text-xs">
                      {item.timezone || t("系统默认")}
                    </code>
                  ),
                },
                {
                  key: "next",
                  label: "下次执行",
                  render: (item) => formatTime(item.next_run_at),
                },
                {
                  key: "status",
                  label: "状态",
                  render: (item) => (
                    <Badge variant={item.enabled ? "default" : "outline"}>
                      {item.enabled ? t("启用") : t("停用")}
                    </Badge>
                  ),
                },
                {
                  key: "actions",
                  label: "操作",
                  render: (item) => (
                    <Button
                      size="icon-sm"
                      variant="destructive"
                      onClick={() => remove.mutate(item.id)}
                      aria-label={t("删除定时任务")}
                    >
                      <Trash2Icon />
                    </Button>
                  ),
                },
              ]}
            />
          </CardContent>
        </Card>
      )}
      <CrudDialog
        open={open}
        onOpenChange={setOpen}
        title="新建定时任务"
        description="日、周、月、年提供可视化配置；高级场景可使用标准 5 段 Cron。"
      >
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="schedule-name">{t("名称")}</FieldLabel>
            <Input
              id="schedule-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel>{t("目标组")}</FieldLabel>
            <Select
              items={groupItems}
              value={groupID}
              onValueChange={(v) => setGroupID(v as string)}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("选择目标组")} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {groupItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel>{t("重复周期")}</FieldLabel>
            <Select
              items={localizedKinds}
              value={kind}
              onValueChange={(v) => setKind(v as string)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {localizedKinds.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel htmlFor="schedule-zone">{t("IANA 时区")}</FieldLabel>
            <Input
              id="schedule-zone"
              value={zone}
              onChange={(e) => setZone(e.target.value)}
            />
            <FieldDescription>
              {t(
                "留空使用后台设置的系统时区；也可填写 Asia/Shanghai、Europe/Berlin。"
              )}
            </FieldDescription>
          </Field>
          {kind === "cron" ? (
            <Field>
              <FieldLabel htmlFor="schedule-cron">Cron</FieldLabel>
              <Input
                id="schedule-cron"
                value={cron}
                onChange={(e) => setCron(e.target.value)}
              />
              <FieldDescription>
                {t("标准 5 段：分钟 小时 日 月 星期。")}
              </FieldDescription>
            </Field>
          ) : kind === "once" ? (
            <Field>
              <FieldLabel htmlFor="schedule-once">{t("执行时间")}</FieldLabel>
              <Input
                id="schedule-once"
                type="datetime-local"
                value={runAt}
                onChange={(e) => setRunAt(e.target.value)}
              />
            </Field>
          ) : (
            <>
              <Field>
                <FieldLabel htmlFor="schedule-time">{t("执行时间")}</FieldLabel>
                <Input
                  id="schedule-time"
                  type="time"
                  value={timeValue}
                  onChange={(e) => setTimeValue(e.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="schedule-interval">{t("间隔")}</FieldLabel>
                <Input
                  id="schedule-interval"
                  type="number"
                  min={1}
                  value={interval}
                  onChange={(e) => setInterval(Number(e.target.value))}
                />
              </Field>
              {kind === "monthly" || kind === "yearly" ? (
                <Field>
                  <FieldLabel htmlFor="schedule-day">{t("日期")}</FieldLabel>
                  <Input
                    id="schedule-day"
                    type="number"
                    min={1}
                    max={31}
                    value={day}
                    onChange={(e) => setDay(Number(e.target.value))}
                  />
                </Field>
              ) : null}
              {kind === "yearly" ? (
                <Field>
                  <FieldLabel htmlFor="schedule-month">{t("月份")}</FieldLabel>
                  <Input
                    id="schedule-month"
                    type="number"
                    min={1}
                    max={12}
                    value={month}
                    onChange={(e) => setMonth(Number(e.target.value))}
                  />
                </Field>
              ) : null}
            </>
          )}
          <Field>
            <FieldLabel htmlFor="schedule-payload">
              {t("事件正文 JSON")}
            </FieldLabel>
            <Textarea
              id="schedule-payload"
              className="min-h-40 font-mono text-xs"
              value={payload}
              onChange={(e) => setPayload(e.target.value)}
            />
          </Field>
          <Button disabled={!name || !groupID} onClick={() => save.mutate()}>
            {t("保存定时任务")}
          </Button>
        </FieldGroup>
      </CrudDialog>
    </>
  )
}
