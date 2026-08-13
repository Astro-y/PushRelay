import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { CopyIcon, SaveIcon } from "lucide-react"
import { toast } from "sonner"
import { api, apiEndpoint, jsonBody } from "@/lib/api"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { PageHeader } from "@/components/page"
import { TimezoneCombobox } from "@/components/timezone-combobox"
import { useI18n } from "@/lib/i18n"

type Settings = {
  default_timezone: string
  allow_private_webhook_targets: boolean
  payload_retention_days: number
  metadata_retention_days: number
  pocketid_enabled: boolean
  pocketid_issuer_url: string
  pocketid_client_id: string
  pocketid_client_secret_configured: boolean
  pocketid_allowed_identity: string
}

type SettingsFormValue = Settings & {
  pocketid_client_secret: string
  pocketid_clear_client_secret: boolean
}

export function SettingsPage() {
  const query = useQuery({
    queryKey: ["settings"],
    queryFn: () => api<Settings>("/settings"),
  })
  return (
    <>
      <PageHeader
        title="系统设置"
        description="管理默认时区、数据保留、安全策略和 Pocket ID 登录。"
      />
      {query.isLoading || !query.data ? (
        <Skeleton className="h-96" />
      ) : (
        <SettingsForm initial={query.data} />
      )}
    </>
  )
}

function SettingsForm({ initial }: { initial: Settings }) {
  const { t } = useI18n()
  const client = useQueryClient()
  const [form, setForm] = useState<SettingsFormValue>({
    ...initial,
    pocketid_client_secret: "",
    pocketid_clear_client_secret: false,
  })
  const save = useMutation({
    mutationFn: (value: SettingsFormValue) => {
      const {
        pocketid_client_secret_configured: _secretConfigured,
        ...payload
      } = value
      void _secretConfigured
      return api<Settings>("/settings", {
        method: "PUT",
        body: jsonBody(payload),
      })
    },
    onSuccess: (value) => {
      setForm({
        ...value,
        pocketid_client_secret: "",
        pocketid_clear_client_secret: false,
      })
      client.setQueryData(["settings"], value)
      toast.success(t("系统设置已保存并立即生效"))
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <div className="grid w-full max-w-7xl items-start gap-6 xl:grid-cols-[minmax(20rem,0.85fr)_minmax(0,1.15fr)]">
      <Card>
        <CardHeader>
          <CardTitle>{t("运行与安全")}</CardTitle>
          <CardDescription>
            {t("调度、日志保留和出站 Webhook 的全局运行策略。")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <FieldGroup>
          <Field>
            <FieldLabel htmlFor="default-timezone">
              {t("系统默认时区")}
            </FieldLabel>
            <TimezoneCombobox
              id="default-timezone"
              value={form.default_timezone}
              onValueChange={(value) =>
                setForm({ ...form, default_timezone: value })
              }
            />
            <FieldDescription>
              {t("输入城市或区域名称搜索，例如 Shanghai、Berlin、New_York。")}
            </FieldDescription>
          </Field>
          <div className="grid gap-5 sm:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="payload-retention">
                {t("消息正文保留天数")}
              </FieldLabel>
              <Input
                id="payload-retention"
                type="number"
                min={0}
                max={3650}
                value={form.payload_retention_days}
                onChange={(event) =>
                  setForm({
                    ...form,
                    payload_retention_days: Number(event.target.value),
                  })
                }
              />
              <FieldDescription>
                {t("设为 0 表示投递完成后不长期保留正文。")}
              </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="metadata-retention">
                {t("事件元数据保留天数")}
              </FieldLabel>
              <Input
                id="metadata-retention"
                type="number"
                min={1}
                max={3650}
                value={form.metadata_retention_days}
                onChange={(event) =>
                  setForm({
                    ...form,
                    metadata_retention_days: Number(event.target.value),
                  })
                }
              />
              <FieldDescription>
                {t("正文保留天数不能超过元数据保留天数。")}
              </FieldDescription>
            </Field>
          </div>
          <Field orientation="horizontal">
            <div className="flex-1">
              <FieldLabel htmlFor="allow-private-targets">
                {t("允许通用 Webhook 访问私网")}
              </FieldLabel>
              <FieldDescription>
                {t("允许环回、局域网和链路本地地址，仅建议用于可信内网。")}
              </FieldDescription>
            </div>
            <Switch
              id="allow-private-targets"
              checked={form.allow_private_webhook_targets}
              onCheckedChange={(checked) =>
                setForm({ ...form, allow_private_webhook_targets: checked })
              }
            />
          </Field>
          {form.allow_private_webhook_targets ? (
            <Alert variant="destructive">
              <AlertTitle>{t("私网访问已开启")}</AlertTitle>
              <AlertDescription>
                {t(
                  "通用出站 Webhook 可以连接本机和内网服务，请只向可信管理员开放后台。"
                )}
              </AlertDescription>
            </Alert>
          ) : null}
          </FieldGroup>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Pocket ID OAuth / OIDC</CardTitle>
          <CardDescription>
            {t(
              "使用 Pocket ID 的 OIDC Discovery、授权码流程和 PKCE 登录管理员账号。"
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <FieldGroup>
          <Field orientation="horizontal">
            <div className="flex-1">
              <FieldLabel htmlFor="pocketid-enabled">
                {t("启用 Pocket ID 登录")}
              </FieldLabel>
              <FieldDescription>
                {t(
                  "启用前必须填写下面所有配置，并在 Pocket ID 中登记回调地址。"
                )}
              </FieldDescription>
            </div>
            <Switch
              id="pocketid-enabled"
              checked={form.pocketid_enabled}
              onCheckedChange={(checked) =>
                setForm({ ...form, pocketid_enabled: checked })
              }
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="pocketid-callback">{t("回调地址")}</FieldLabel>
            <div className="flex gap-2">
              <Input
                id="pocketid-callback"
                value={apiEndpoint("/auth/pocketid/callback")}
                readOnly
              />
              <Button
                variant="outline"
                size="icon"
                aria-label={t("复制回调地址")}
                onClick={() => {
                  void navigator.clipboard.writeText(
                    apiEndpoint("/auth/pocketid/callback")
                  )
                  toast.success(t("回调地址已复制"))
                }}
              >
                <CopyIcon />
              </Button>
            </div>
            <FieldDescription>
              {t(
                "把这个完整地址添加到 Pocket ID OIDC Client 的 Callback URLs。"
              )}
            </FieldDescription>
          </Field>
          <Field>
            <FieldLabel htmlFor="pocketid-issuer">
              {t("Issuer / Discovery URL")}
            </FieldLabel>
            <Input
              id="pocketid-issuer"
              type="url"
              value={form.pocketid_issuer_url}
              onChange={(event) =>
                setForm({ ...form, pocketid_issuer_url: event.target.value })
              }
              placeholder="https://id.example.com"
              autoComplete="url"
            />
            <FieldDescription>
              {t("填写 Pocket ID 实例地址；也可以粘贴其 OIDC Discovery URL。")}
            </FieldDescription>
          </Field>
          <div className="grid gap-5 sm:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="pocketid-client-id">Client ID</FieldLabel>
              <Input
                id="pocketid-client-id"
                value={form.pocketid_client_id}
                onChange={(event) =>
                  setForm({ ...form, pocketid_client_id: event.target.value })
                }
                autoComplete="off"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="pocketid-client-secret">
                Client Secret
              </FieldLabel>
              <Input
                id="pocketid-client-secret"
                type="password"
                value={form.pocketid_client_secret}
                onChange={(event) =>
                  setForm({
                    ...form,
                    pocketid_client_secret: event.target.value,
                    pocketid_clear_client_secret: false,
                  })
                }
                placeholder={
                  form.pocketid_client_secret_configured
                    ? t("已安全保存，留空表示不修改")
                    : t("输入 Pocket ID Client Secret")
                }
                autoComplete="new-password"
              />
            </Field>
          </div>
          <Field>
            <FieldLabel htmlFor="pocketid-identity">
              {t("允许绑定的用户名或邮箱")}
            </FieldLabel>
            <Input
              id="pocketid-identity"
              value={form.pocketid_allowed_identity}
              onChange={(event) =>
                setForm({
                  ...form,
                  pocketid_allowed_identity: event.target.value,
                })
              }
              placeholder="admin@example.com"
              autoComplete="off"
            />
            <FieldDescription>
              {t(
                "首次登录必须与 Pocket ID 的 preferred_username 或已验证邮箱完全匹配，之后固定绑定其 subject。"
              )}
            </FieldDescription>
          </Field>
          </FieldGroup>
        </CardContent>
      </Card>

      <div className="flex justify-end xl:col-span-2">
        <Button
          className="w-full sm:w-auto"
          disabled={
            save.isPending ||
            !form.default_timezone.trim() ||
            form.payload_retention_days < 0 ||
            form.metadata_retention_days < 1 ||
            form.payload_retention_days > form.metadata_retention_days ||
            (form.pocketid_enabled &&
              (!form.pocketid_issuer_url.trim() ||
                !form.pocketid_client_id.trim() ||
                (!form.pocketid_client_secret_configured &&
                  !form.pocketid_client_secret.trim()) ||
                !form.pocketid_allowed_identity.trim()))
          }
          onClick={() => save.mutate(form)}
        >
          <SaveIcon data-icon="inline-start" />
          {save.isPending ? t("保存中…") : t("保存设置")}
        </Button>
      </div>
    </div>
  )
}
