import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { PlusIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"
import { api, jsonBody } from "@/lib/api"
import type { Channel, MessageTemplate, TargetGroup } from "@/lib/types"
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
import { CrudDialog } from "@/components/crud-dialog"
import { PageHeader, PageLoading, formatTime } from "@/components/page"
import { ResourceTable } from "@/components/resource-table"
import { useI18n } from "@/lib/i18n"

export function GroupsPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [channelID, setChannelID] = useState("")
  const [templateID, setTemplateID] = useState("")
  const groups = useQuery({
    queryKey: ["groups"],
    queryFn: () => api<TargetGroup[]>("/target-groups"),
  })
  const channels = useQuery({
    queryKey: ["channels"],
    queryFn: () => api<Channel[]>("/channels"),
  })
  const templates = useQuery({
    queryKey: ["templates"],
    queryFn: () => api<MessageTemplate[]>("/templates"),
  })
  const save = useMutation({
    mutationFn: () =>
      api("/target-groups", {
        method: "POST",
        body: jsonBody({
          name,
          bindings: [
            { channel_id: channelID, template_id: templateID, enabled: true },
          ],
        }),
      }),
    onSuccess: () => {
      toast.success(t("目标组已创建"))
      setOpen(false)
      client.invalidateQueries({ queryKey: ["groups"] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const remove = useMutation({
    mutationFn: (id: string) =>
      api(`/target-groups/${id}`, { method: "DELETE" }),
    onSuccess: () => client.invalidateQueries({ queryKey: ["groups"] }),
    onError: (e: Error) => toast.error(e.message),
  })
  const channelItems = (channels.data ?? []).map((item) => ({
    label: item.name,
    value: item.id,
  }))
  const compatible = (templates.data ?? []).filter(
    (item) =>
      !channelID ||
      item.channel_type ===
        channels.data?.find((channel) => channel.id === channelID)?.type
  )
  const templateItems = compatible.map((item) => ({
    label: item.name,
    value: item.id,
  }))
  return (
    <>
      <PageHeader
        title="目标组"
        description="将“渠道 + 模板”组合为可复用的投递目标，规则和定时任务只需引用一次。"
        action={
          <Button onClick={() => setOpen(true)}>
            <PlusIcon data-icon="inline-start" />
            {t("新建目标组")}
          </Button>
        }
      />
      {groups.isLoading ? (
        <PageLoading />
      ) : (
        <Card>
          <CardContent>
            <ResourceTable
              items={groups.data ?? []}
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
                  key: "bindings",
                  label: "投递绑定",
                  render: (item) => (
                    <div className="flex flex-wrap gap-1">
                      {item.bindings.map((binding) => (
                        <Badge variant="secondary" key={binding.id}>
                          {binding.channel_name} · {binding.template_name}
                        </Badge>
                      ))}
                    </div>
                  ),
                },
                {
                  key: "count",
                  label: "数量",
                  render: (item) => (
                    <span className="tabular-nums">{item.bindings.length}</span>
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
                      aria-label={t("删除目标组")}
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
        title="新建目标组"
        description="首个版本可在创建时添加一个绑定，之后可通过更新 API 提交多个绑定。"
      >
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="group-name">{t("名称")}</FieldLabel>
            <Input
              id="group-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel>{t("渠道")}</FieldLabel>
            <Select
              items={channelItems}
              value={channelID}
              onValueChange={(value) => {
                setChannelID(value as string)
                setTemplateID("")
              }}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("选择渠道")} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {channelItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel>{t("消息模板")}</FieldLabel>
            <Select
              items={templateItems}
              value={templateID}
              onValueChange={(value) => setTemplateID(value as string)}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("选择相同渠道类型的模板")} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {templateItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>
              {t("模板类型必须与所选渠道一致。")}
            </FieldDescription>
          </Field>
          <Button
            disabled={!name || !channelID || !templateID}
            onClick={() => save.mutate()}
          >
            {t("保存目标组")}
          </Button>
        </FieldGroup>
      </CrudDialog>
    </>
  )
}
