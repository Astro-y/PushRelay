import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { PlusIcon, Trash2Icon } from "lucide-react"
import { toast } from "sonner"
import { api, jsonBody } from "@/lib/api"
import type { Rule, Source, TargetGroup } from "@/lib/types"
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
import { PageHeader, PageLoading } from "@/components/page"
import { ResourceTable } from "@/components/resource-table"
import { useI18n } from "@/lib/i18n"

export function RulesPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState("")
  const [sourceID, setSourceID] = useState("")
  const [groupID, setGroupID] = useState("")
  const [path, setPath] = useState("body.status")
  const [operator, setOperator] = useState("eq")
  const [value, setValue] = useState("error")
  const [priority, setPriority] = useState(100)
  const rules = useQuery({
    queryKey: ["rules"],
    queryFn: () => api<Rule[]>("/rules"),
  })
  const sources = useQuery({
    queryKey: ["sources"],
    queryFn: () => api<Source[]>("/sources"),
  })
  const groups = useQuery({
    queryKey: ["groups"],
    queryFn: () => api<TargetGroup[]>("/target-groups"),
  })
  const save = useMutation({
    mutationFn: () =>
      api("/rules", {
        method: "POST",
        body: jsonBody({
          name,
          source_id: sourceID,
          target_group_id: groupID,
          priority,
          enabled: true,
          condition: { path, operator, value },
        }),
      }),
    onSuccess: () => {
      toast.success(t("规则已创建"))
      setOpen(false)
      client.invalidateQueries({ queryKey: ["rules"] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const remove = useMutation({
    mutationFn: (id: string) => api(`/rules/${id}`, { method: "DELETE" }),
    onSuccess: () => client.invalidateQueries({ queryKey: ["rules"] }),
  })
  return (
    <>
      <PageHeader
        title="路由规则"
        description="使用受限条件树匹配正文、Header、查询参数和元数据，不执行任意脚本。"
        action={
          <Button onClick={() => setOpen(true)}>
            <PlusIcon data-icon="inline-start" />
            {t("新建规则")}
          </Button>
        }
      />
      {rules.isLoading ? (
        <PageLoading />
      ) : (
        <Card>
          <CardContent>
            <ResourceTable
              items={rules.data ?? []}
              columns={[
                {
                  key: "name",
                  label: "名称",
                  render: (item) => (
                    <span className="font-medium">{item.name}</span>
                  ),
                },
                {
                  key: "source",
                  label: "入口",
                  render: (item) =>
                    sources.data?.find((x) => x.id === item.source_id)?.name ??
                    item.source_id,
                },
                {
                  key: "condition",
                  label: "条件",
                  render: (item) => (
                    <code className="block max-w-sm truncate text-xs">
                      {JSON.stringify(item.condition)}
                    </code>
                  ),
                },
                {
                  key: "priority",
                  label: "优先级",
                  render: (item) => (
                    <Badge variant="secondary">{item.priority}</Badge>
                  ),
                },
                {
                  key: "target",
                  label: "目标组",
                  render: (item) =>
                    groups.data?.find((x) => x.id === item.target_group_id)
                      ?.name ?? item.target_group_id,
                },
                {
                  key: "actions",
                  label: "操作",
                  render: (item) => (
                    <Button
                      size="icon-sm"
                      variant="destructive"
                      onClick={() => remove.mutate(item.id)}
                      aria-label={t("删除规则")}
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
        title="新建路由规则"
        description="当前表单创建单条件规则；管理 API 支持最多 5 层 AND/OR 嵌套条件树。"
      >
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="rule-name">{t("名称")}</FieldLabel>
            <Input
              id="rule-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </Field>
          <SelectField
            label="Webhook 入口"
            value={sourceID}
            onChange={setSourceID}
            items={(sources.data ?? []).map((x) => ({
              label: x.name,
              value: x.id,
            }))}
          />
          <SelectField
            label="目标组"
            value={groupID}
            onChange={setGroupID}
            items={(groups.data ?? []).map((x) => ({
              label: x.name,
              value: x.id,
            }))}
          />
          <Field>
            <FieldLabel htmlFor="rule-path">{t("字段路径")}</FieldLabel>
            <Input
              id="rule-path"
              value={path}
              onChange={(e) => setPath(e.target.value)}
            />
            <FieldDescription>
              {t("例如 body.status、headers.x-event-type、query.environment。")}
            </FieldDescription>
          </Field>
          <SelectField
            label="运算符"
            value={operator}
            onChange={setOperator}
            items={[
              "eq",
              "ne",
              "contains",
              "not_contains",
              "regex",
              "gt",
              "gte",
              "lt",
              "lte",
              "in",
              "not_in",
              "exists",
              "not_exists",
            ].map((x) => ({ label: x, value: x }))}
          />
          <Field>
            <FieldLabel htmlFor="rule-value">{t("比较值")}</FieldLabel>
            <Input
              id="rule-value"
              value={value}
              onChange={(e) => setValue(e.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="rule-priority">{t("优先级")}</FieldLabel>
            <Input
              id="rule-priority"
              type="number"
              value={priority}
              onChange={(e) => setPriority(Number(e.target.value))}
            />
          </Field>
          <Button
            disabled={!name || !sourceID || !groupID}
            onClick={() => save.mutate()}
          >
            {t("保存规则")}
          </Button>
        </FieldGroup>
      </CrudDialog>
    </>
  )
}
function SelectField({
  label,
  value,
  onChange,
  items,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  items: { label: string; value: string }[]
}) {
  const { t } = useI18n()
  return (
    <Field>
      <FieldLabel>{t(label)}</FieldLabel>
      <Select
        items={items}
        value={value}
        onValueChange={(v) => onChange(v as string)}
      >
        <SelectTrigger>
          <SelectValue placeholder={`${t("选择")}${t(label)}`} />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </Field>
  )
}
