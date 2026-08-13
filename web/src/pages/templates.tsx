import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { BracesIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"
import { api, jsonBody } from "@/lib/api"
import type { ChannelType, MessageTemplate } from "@/lib/types"
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

const defaultBody = JSON.stringify(
  {
    msgtype: "text",
    text: { content: '{{ get "body.message" | default "收到一条新消息" }}' },
  },
  null,
  2
)
export function TemplatesPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [type, setType] = useState("wecom")
  const [body, setBody] = useState(defaultBody)
  const templates = useQuery({
    queryKey: ["templates"],
    queryFn: () => api<MessageTemplate[]>("/templates"),
  })
  const types = useQuery({
    queryKey: ["channel-types"],
    queryFn: () => api<ChannelType[]>("/channel-types"),
  })
  const save = useMutation({
    mutationFn: () =>
      api("/templates", {
        method: "POST",
        body: jsonBody({ name, channel_type: type, body: JSON.parse(body) }),
      }),
    onSuccess: () => {
      toast.success(t("模板已创建"))
      setOpen(false)
      client.invalidateQueries({ queryKey: ["templates"] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const preview = useMutation({
    mutationFn: () =>
      api<{ rendered: unknown }>("/preview", {
        method: "POST",
        body: jsonBody({ body: JSON.parse(body) }),
      }),
    onSuccess: (data) => toast.success(JSON.stringify(data.rendered, null, 2)),
    onError: (e: Error) => toast.error(e.message),
  })
  const remove = useMutation({
    mutationFn: (id: string) => api(`/templates/${id}`, { method: "DELETE" }),
    onSuccess: () => client.invalidateQueries({ queryKey: ["templates"] }),
  })
  const selectItems = (types.data ?? []).map((item) => ({
    label: t(item.label),
    value: item.type,
  }))
  return (
    <>
      <PageHeader
        title="消息模板"
        description="逐字段渲染安全模板，把统一事件上下文转换为各个平台所需的消息结构。"
        action={
          <Button onClick={() => setOpen(true)}>
            <PlusIcon data-icon="inline-start" />
            {t("新建模板")}
          </Button>
        }
      />
      {templates.isLoading ? (
        <PageLoading />
      ) : (
        <Card>
          <CardContent>
            <ResourceTable
              items={templates.data ?? []}
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
                  label: "渠道类型",
                  render: (item) => (
                    <Badge variant="secondary">{item.channel_type}</Badge>
                  ),
                },
                {
                  key: "body",
                  label: "模板",
                  render: (item) => (
                    <code className="block max-w-xs truncate text-xs text-muted-foreground">
                      {JSON.stringify(item.body)}
                    </code>
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
                      aria-label={t("删除模板")}
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
        title="新建消息模板"
        description="模板字符串支持 get、default、toJSON、truncate、dateInZone 和 urlquery。"
      >
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="template-name">{t("名称")}</FieldLabel>
            <Input
              id="template-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel>{t("渠道类型")}</FieldLabel>
            <Select
              items={selectItems}
              value={type}
              onValueChange={(value) => setType(value as string)}
            >
              <SelectTrigger>
                <SelectValue />
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
          <Field>
            <FieldLabel htmlFor="template-body">{t("模板 JSON")}</FieldLabel>
            <Textarea
              id="template-body"
              className="min-h-64 font-mono text-xs"
              value={body}
              onChange={(e) => setBody(e.target.value)}
            />
            <FieldDescription>
              {t("所有字段在渲染后才会统一序列化，避免手工 JSON 转义问题。")}
            </FieldDescription>
          </Field>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => preview.mutate()}>
              <BracesIcon data-icon="inline-start" />
              {t("预览")}
            </Button>
            <Button onClick={() => save.mutate()}>{t("保存模板")}</Button>
          </div>
        </FieldGroup>
      </CrudDialog>
    </>
  )
}
