import { useEffect, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Navigate, useNavigate, useSearchParams } from "react-router-dom"
import { KeyRoundIcon, LoaderCircleIcon, UserRoundPlusIcon } from "lucide-react"
import { toast } from "sonner"
import { api, apiEndpoint, jsonBody, setCSRF } from "@/lib/api"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
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
import { Separator } from "@/components/ui/separator"
import { useI18n } from "@/lib/i18n"
import { BrandMark } from "@/components/brand-mark"

export function AuthGate({ children }: { children: React.ReactNode }) {
  const { t } = useI18n()
  const setup = useQuery({
    queryKey: ["setup-status"],
    queryFn: () => api<{ required: boolean }>("/setup/status"),
  })
  const me = useQuery({
    queryKey: ["me"],
    queryFn: () => api<{ username: string; csrf_token: string }>("/auth/me"),
    enabled: setup.data?.required === false,
    retry: false,
  })
  useEffect(() => {
    if (me.data?.csrf_token) setCSRF(me.data.csrf_token)
  }, [me.data])
  if (setup.isLoading || (setup.data?.required === false && me.isLoading))
    return (
      <AuthFrame>
        <p className="text-sm text-muted-foreground">{t("正在连接服务…")}</p>
      </AuthFrame>
    )
  if (setup.data?.required) return <Navigate to="/setup" replace />
  if (me.isError) return <Navigate to="/login" replace />
  return children
}

export function SetupPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [token, setToken] = useState("")
  const [username, setUsername] = useState("admin")
  const [password, setPassword] = useState("")
  const mutation = useMutation({
    mutationFn: () =>
      api("/setup", {
        method: "POST",
        headers: { "X-Setup-Token": token },
        body: jsonBody({ username, password }),
      }),
    onSuccess: () => {
      client.setQueryData(["setup-status"], { required: false })
      client.removeQueries({ queryKey: ["me"] })
      toast.success(t("管理员创建成功"))
      navigate("/login")
    },
    onError: (e: Error) => toast.error(e.message),
  })
  return (
    <AuthFrame>
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t("初始化 PushRelay")}</CardTitle>
          <CardDescription>
            {t(
              "这是唯一一次创建管理员的机会。Setup Token 会显示在后端启动日志中。"
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="setup-token">Setup Token</FieldLabel>
              <Input
                id="setup-token"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                autoComplete="off"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="setup-user">{t("管理员账号")}</FieldLabel>
              <Input
                id="setup-user"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
              />
            </Field>
            <Field>
              <FieldLabel htmlFor="setup-password">
                {t("管理员密码")}
              </FieldLabel>
              <Input
                id="setup-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="new-password"
              />
              <FieldDescription>
                {t("至少 12 个字符，将使用 Argon2id 加密保存。")}
              </FieldDescription>
            </Field>
          </FieldGroup>
        </CardContent>
        <CardFooter>
          <Button
            className="w-full"
            disabled={mutation.isPending || !token || password.length < 12}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? (
              <LoaderCircleIcon data-icon="inline-start" />
            ) : (
              <UserRoundPlusIcon data-icon="inline-start" />
            )}
            {t("创建管理员")}
          </Button>
        </CardFooter>
      </Card>
    </AuthFrame>
  )
}

export function LoginPage() {
  const { t } = useI18n()
  const client = useQueryClient()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [otp, setOTP] = useState("")
  const [twoFactorRequired, setTwoFactorRequired] = useState(false)
  const pocketID = useQuery({
    queryKey: ["pocketid-status"],
    queryFn: () => api<{ enabled: boolean }>("/auth/pocketid/status"),
    retry: false,
  })
  useEffect(() => {
    const oauthError = searchParams.get("oauth_error")
    if (oauthError) toast.error(oauthError)
  }, [searchParams])
  const mutation = useMutation({
    mutationFn: () =>
      api<{
        username?: string
        csrf_token?: string
        two_factor_required?: boolean
      }>("/auth/login", {
        method: "POST",
        body: jsonBody({ username, password, otp }),
      }),
    onSuccess: (data) => {
      if (data.two_factor_required) {
        setTwoFactorRequired(true)
        setOTP("")
        return
      }
      if (!data.csrf_token || !data.username) return
      setCSRF(data.csrf_token)
      client.setQueryData(["setup-status"], { required: false })
      client.setQueryData(["me"], data)
      navigate("/")
    },
    onError: (e: Error) => toast.error(e.message),
  })
  return (
    <AuthFrame>
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>{t("欢迎回来")}</CardTitle>
          <CardDescription>{t("登录你的自托管消息中转后台。")}</CardDescription>
        </CardHeader>
        <CardContent>
          <FieldGroup>
            {twoFactorRequired ? (
              <Field>
                <FieldLabel htmlFor="login-otp">
                  {t("验证码或恢复码")}
                </FieldLabel>
                <Input
                  id="login-otp"
                  value={otp}
                  onChange={(event) => setOTP(event.target.value)}
                  autoComplete="one-time-code"
                  autoFocus
                />
                <FieldDescription>
                  {t("输入验证器中的 6 位验证码，或一个尚未使用的恢复码。")}
                </FieldDescription>
              </Field>
            ) : (
              <>
                <Field>
                  <FieldLabel htmlFor="login-user">{t("账号")}</FieldLabel>
                  <Input
                    id="login-user"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    autoComplete="username"
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="login-password">{t("密码")}</FieldLabel>
                  <Input
                    id="login-password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="current-password"
                  />
                </Field>
              </>
            )}
          </FieldGroup>
          {mutation.isError ? (
            <Alert variant="destructive" className="mt-4">
              <AlertTitle>{t("登录失败")}</AlertTitle>
              <AlertDescription>{mutation.error.message}</AlertDescription>
            </Alert>
          ) : null}
        </CardContent>
        <CardFooter className="flex flex-col gap-3">
          <Button
            className="w-full"
            disabled={
              mutation.isPending ||
              !username ||
              !password ||
              (twoFactorRequired && !otp)
            }
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? (
              <LoaderCircleIcon data-icon="inline-start" />
            ) : null}
            {t(twoFactorRequired ? "验证并登录" : "登录")}
          </Button>
          {twoFactorRequired ? (
            <Button
              variant="ghost"
              className="w-full"
              onClick={() => {
                setTwoFactorRequired(false)
                setOTP("")
              }}
            >
              {t("返回账号密码")}
            </Button>
          ) : pocketID.data?.enabled ? (
            <>
              <div className="flex w-full items-center gap-3 text-xs text-muted-foreground">
                <Separator className="flex-1" />
                {t("或者")}
                <Separator className="flex-1" />
              </div>
              <Button
                variant="outline"
                className="w-full"
                onClick={() =>
                  window.location.assign(apiEndpoint("/auth/pocketid/start"))
                }
              >
                <KeyRoundIcon data-icon="inline-start" />
                {t("使用 Pocket ID 登录")}
              </Button>
            </>
          ) : null}
        </CardFooter>
      </Card>
    </AuthFrame>
  )
}

function AuthFrame({ children }: { children: React.ReactNode }) {
  const { t } = useI18n()
  return (
    <main className="grid min-h-svh place-items-center bg-muted/40 p-6">
      <div className="flex w-full flex-col items-center gap-6">
        <div className="flex items-center gap-3">
          <BrandMark className="size-10" />
          <div>
            <p className="font-heading font-semibold">PushRelay</p>
            <p className="text-xs text-muted-foreground">
              {t("自托管消息路由器")}
            </p>
          </div>
        </div>
        {children}
      </div>
    </main>
  )
}
