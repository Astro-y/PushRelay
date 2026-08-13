import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { PlusIcon, SendIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"
import { api, jsonBody } from "@/lib/api"
import type { Channel, ChannelType } from "@/lib/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { CrudDialog } from "@/components/crud-dialog"
import { PageHeader, PageLoading, formatTime } from "@/components/page"
import { ResourceTable } from "@/components/resource-table"
import { useI18n } from "@/lib/i18n"

export function ChannelsPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [type, setType] = useState("wecom")
  const [enabled, setEnabled] = useState(true)
  const [config, setConfig] = useState<Record<string, string>>({})
  const channels = useQuery({
    queryKey: ["channels"],
    queryFn: () => api<Channel[]>("/channels"),
  })
  const types = useQuery({
    queryKey: ["channel-types"],
    queryFn: () => api<ChannelType[]>("/channel-types"),
  })
  const save = useMutation({
    mutationFn: () =>
      api("/channels", {
        method: "POST",
        body: jsonBody({ name, type, enabled, config }),
      }),
    onSuccess: () => {
      toast.success(t("渠道已创建"))
      setOpen(false)
      client.invalidateQueries({ queryKey: ["channels"] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const remove = useMutation({
    mutationFn: (id: string) => api(`/channels/${id}`, { method: "DELETE" }),
    onSuccess: () => client.invalidateQueries({ queryKey: ["channels"] }),
    onError: (e: Error) => toast.error(e.message),
  })
  const test = useMutation({
    mutationFn: (channel: Channel) =>
      api(`/channels/${channel.id}/test`, {
        method: "POST",
        body: jsonBody({
          body: testMessage(channel.type, t("PushRelay 渠道连接测试")),
        }),
      }),
    onSuccess: () => toast.success(t("测试消息已发送")),
    onError: (e: Error) => toast.error(e.message),
  })
  const current = types.data?.find((item) => item.type === type)
  const selectItems = (types.data ?? []).map((item) => ({
    label: t(item.label),
    value: item.type,
  }))
  return (
    <>
      <PageHeader
        title="通知渠道"
        description="集中保管机器人、应用和通用 Webhook 的连接凭据。Secret 只会在服务端解密使用。"
        action={
          <Button onClick={() => setOpen(true)}>
            <PlusIcon data-icon="inline-start" />
            {t("添加渠道")}
          </Button>
        }
      />
      {channels.isLoading ? (
        <PageLoading />
      ) : (
        <Card>
          <CardContent>
            <ResourceTable
              items={channels.data ?? []}
              columns={[
                {
                  key: "name",
                  label: "名称",
                  render: (item) => (
                    <div className="flex flex-col gap-1">
                      <span className="font-medium">{item.name}</span>
                      <span className="text-xs text-muted-foreground">
                        {formatTime(item.created_at)}
                      </span>
                    </div>
                  ),
                },
                {
                  key: "type",
                  label: "类型",
                  render: (item) => (
                    <Badge variant="secondary">
                      {t(
                        types.data?.find((x) => x.type === item.type)?.label ??
                          item.type
                      )}
                    </Badge>
                  ),
                },
                {
                  key: "enabled",
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
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => test.mutate(item)}
                      >
                        <SendIcon data-icon="inline-start" />
                        {t("测试")}
                      </Button>
                      <Button
                        size="icon-sm"
                        variant="destructive"
                        onClick={() => remove.mutate(item.id)}
                        aria-label={t("删除渠道")}
                      >
                        <Trash2Icon />
                      </Button>
                    </div>
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
        title="添加通知渠道"
        description="选择渠道类型并填写其官方 API 所需凭据。"
      >
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="channel-name">{t("名称")}</FieldLabel>
            <Input
              id="channel-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("生产告警群")}
            />
          </Field>
          <Field>
            <FieldLabel>{t("渠道类型")}</FieldLabel>
            <Select
              items={selectItems}
              value={type}
              onValueChange={(value) => {
                setType(value as string)
                setConfig({})
              }}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("选择渠道")} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {selectItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          {[...(current?.required ?? []), ...(current?.optional ?? [])].map(
            (key) => (
              <Field key={key}>
                <FieldLabel htmlFor={`config-${key}`}>
                  {key}
                  {current?.required.includes(key) ? " *" : ""}
                </FieldLabel>
                <Input
                  id={`config-${key}`}
                  type={
                    key.includes("secret") || key.includes("token")
                      ? "password"
                      : "text"
                  }
                  value={config[key] ?? ""}
                  onChange={(e) =>
                    setConfig((value) => ({ ...value, [key]: e.target.value }))
                  }
                />
              </Field>
            )
          )}
          <Field orientation="horizontal">
            <Switch
              id="channel-enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
            />
            <FieldLabel htmlFor="channel-enabled">
              {t("创建后立即启用")}
            </FieldLabel>
          </Field>
          <Button
            disabled={save.isPending || !name}
            onClick={() => save.mutate()}
          >
            {t("保存渠道")}
          </Button>
        </FieldGroup>
      </CrudDialog>
    </>
  )
}

function testMessage(type: string, text: string): Record<string, unknown> {
  switch (type) {
    case "dingtalk":
      return { msgtype: "text", text: { content: text } }
    case "wecom":
      return { msgtype: "text", text: { content: text } }
    case "wecom_app":
      return { msgtype: "text", text: { content: text }, touser: "@all" }
    case "feishu":
      return { msg_type: "text", content: { text } }
    case "telegram":
      return { text }
    case "discord":
      return { content: text }
    case "bark":
      return { title: "PushRelay", body: text }
    default:
      return {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: { text },
      }
  }
}
