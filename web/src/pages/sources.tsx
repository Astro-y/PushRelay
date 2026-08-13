import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  CopyIcon,
  KeyRoundIcon,
  PlusIcon,
  RotateCwIcon,
  Trash2Icon,
} from "lucide-react"
import { toast } from "sonner"
import { api, jsonBody } from "@/lib/api"
import type { Source } from "@/lib/types"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
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
import { Switch } from "@/components/ui/switch"
import { CrudDialog } from "@/components/crud-dialog"
import { PageHeader, PageLoading, formatTime } from "@/components/page"
import { ResourceTable } from "@/components/resource-table"
import { useI18n } from "@/lib/i18n"

type SourceResult = {
  source: Source
  token?: string
  hook_url?: string
  hmac_secret?: string
}
export function SourcesPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [created, setCreated] = useState<SourceResult | null>(null)
  const [name, setName] = useState("")
  const [hmac, setHmac] = useState(false)
  const [matchMode, setMatchMode] = useState("all_match")
  const [payloadPolicy, setPayloadPolicy] = useState("redacted")
  const [cidrs, setCidrs] = useState("")
  const [sensitiveFields, setSensitiveFields] = useState("")
  const sources = useQuery({
    queryKey: ["sources"],
    queryFn: () => api<Source[]>("/sources"),
  })
  const save = useMutation({
    mutationFn: () =>
      api<SourceResult>("/sources", {
        method: "POST",
        body: jsonBody({
          name,
          hmac_enabled: hmac,
          match_mode: matchMode,
          payload_policy: payloadPolicy,
          allowed_cidrs: cidrs
            .split(",")
            .map((v) => v.trim())
            .filter(Boolean),
          custom_sensitive_fields: sensitiveFields
            .split(",")
            .map((v) => v.trim())
            .filter(Boolean),
          enabled: true,
        }),
      }),
    onSuccess: (data) => {
      setCreated(data)
      client.invalidateQueries({ queryKey: ["sources"] })
    },
    onError: (e: Error) => toast.error(e.message),
  })
  const remove = useMutation({
    mutationFn: (id: string) => api(`/sources/${id}`, { method: "DELETE" }),
    onSuccess: () => client.invalidateQueries({ queryKey: ["sources"] }),
  })
  const rotate = useMutation({
    mutationFn: (id: string) =>
      api<{ hook_url: string }>(`/sources/${id}/rotate`, { method: "POST" }),
    onSuccess: (data) => {
      navigator.clipboard.writeText(data.hook_url)
      toast.success(t("入口令牌已轮换，新地址已复制"))
    },
    onError: (e: Error) => toast.error(e.message),
  })
  return (
    <>
      <PageHeader
        title="Webhook 入口"
        description="每个入口拥有不可猜测令牌，可选 HMAC、IP 白名单、载荷留存策略和规则匹配模式。"
        action={
          <Button
            onClick={() => {
              setCreated(null)
              setOpen(true)
            }}
          >
            <PlusIcon data-icon="inline-start" />
            {t("创建入口")}
          </Button>
        }
      />
      {sources.isLoading ? (
        <PageLoading />
      ) : (
        <Card>
          <CardContent>
            <ResourceTable
              items={sources.data ?? []}
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
                  key: "token",
                  label: "令牌",
                  render: (item) => (
                    <code className="text-xs">{item.token_prefix}…</code>
                  ),
                },
                {
                  key: "mode",
                  label: "规则模式",
                  render: (item) => (
                    <Badge variant="secondary">{t(item.match_mode)}</Badge>
                  ),
                },
                {
                  key: "policy",
                  label: "载荷日志",
                  render: (item) => (
                    <Badge variant="outline">{t(item.payload_policy)}</Badge>
                  ),
                },
                {
                  key: "actions",
                  label: "操作",
                  render: (item) => (
                    <div className="flex gap-2">
                      <Button
                        size="icon-sm"
                        variant="outline"
                        onClick={() => rotate.mutate(item.id)}
                        aria-label={t("轮换令牌")}
                      >
                        <RotateCwIcon />
                      </Button>
                      <Button
                        size="icon-sm"
                        variant="destructive"
                        onClick={() => remove.mutate(item.id)}
                        aria-label={t("删除入口")}
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
        title="创建 Webhook 入口"
        description="入口 URL 和 HMAC Secret 仅在创建时完整显示，请立即保存。"
      >
        {created ? (
          <FieldGroup>
            <Alert>
              <KeyRoundIcon />
              <AlertTitle>{t("请立即保存凭据")}</AlertTitle>
              <AlertDescription>
                {t("关闭弹窗后将无法再次查看完整 Token 或 HMAC Secret。")}
              </AlertDescription>
            </Alert>
            <Field>
              <FieldLabel>{t("入口地址")}</FieldLabel>
              <Input readOnly value={created.hook_url ?? ""} />
            </Field>
            {created.hmac_secret ? (
              <Field>
                <FieldLabel>HMAC Secret</FieldLabel>
                <Input readOnly value={created.hmac_secret} />
              </Field>
            ) : null}
            <Button
              onClick={() => {
                navigator.clipboard.writeText(
                  `${created.hook_url}\n${created.hmac_secret ?? ""}`
                )
                toast.success(t("凭据已复制"))
              }}
            >
              <CopyIcon data-icon="inline-start" />
              {t("复制凭据")}
            </Button>
          </FieldGroup>
        ) : (
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="source-name">{t("名称")}</FieldLabel>
              <Input
                id="source-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </Field>
            <Field>
              <FieldLabel>{t("规则命中模式")}</FieldLabel>
              <SimpleSelect
                value={matchMode}
                onChange={setMatchMode}
                items={[
                  { label: t("执行全部命中规则"), value: "all_match" },
                  { label: t("仅执行第一条规则"), value: "first_match" },
                ]}
              />
            </Field>
            <Field>
              <FieldLabel>{t("载荷日志")}</FieldLabel>
              <SimpleSelect
                value={payloadPolicy}
                onChange={setPayloadPolicy}
                items={[
                  { label: t("脱敏后保存"), value: "redacted" },
                  { label: t("仅保存元数据"), value: "metadata" },
                  { label: t("不保存正文"), value: "none" },
                ]}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="source-cidrs">
                {t("IP / CIDR 白名单")}
              </FieldLabel>
              <Input
                id="source-cidrs"
                value={cidrs}
                onChange={(e) => setCidrs(e.target.value)}
                placeholder="10.0.0.0/8, 203.0.113.1"
              />
              <FieldDescription>
                {t("留空表示允许所有来源 IP。")}
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="source-sensitive-fields">
                {t("额外脱敏字段")}
              </FieldLabel>
              <Input
                id="source-sensitive-fields"
                value={sensitiveFields}
                onChange={(e) => setSensitiveFields(e.target.value)}
                placeholder="phone, id_card, customer_code"
              />
              <FieldDescription>
                {t("按字段名匹配，多个名称使用逗号分隔。")}
              </FieldDescription>
            </Field>
            <Field orientation="horizontal">
              <Switch
                id="source-hmac"
                checked={hmac}
                onCheckedChange={setHmac}
              />
              <FieldLabel htmlFor="source-hmac">
                {t("启用 HMAC-SHA256 签名验证")}
              </FieldLabel>
            </Field>
            <Button disabled={!name} onClick={() => save.mutate()}>
              {t("创建入口")}
            </Button>
          </FieldGroup>
        )}
      </CrudDialog>
    </>
  )
}
function SimpleSelect({
  value,
  onChange,
  items,
}: {
  value: string
  onChange: (v: string) => void
  items: { label: string; value: string }[]
}) {
  return (
    <Select
      items={items}
      value={value}
      onValueChange={(v) => onChange(v as string)}
    >
      <SelectTrigger>
        <SelectValue />
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
  )
}
