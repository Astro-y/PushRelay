import { useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  CopyIcon,
  KeyRoundIcon,
  LoaderCircleIcon,
  ShieldCheckIcon,
  ShieldOffIcon,
  UserRoundIcon,
} from "lucide-react"
import { toast } from "sonner"
import { api, jsonBody } from "@/lib/api"
import { useI18n } from "@/lib/i18n"
import { PageHeader, PageLoading } from "@/components/page"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
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
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group"

type Me = {
  id: string
  username: string
  csrf_token: string
  two_factor_enabled: boolean
  pocketid_linked: boolean
}

type TOTPSetup = {
  secret: string
  provisioning_uri: string
  qr_code_data_url: string
}

export function AccountPage() {
  const { t } = useI18n()
  const me = useQuery({
    queryKey: ["me"],
    queryFn: () => api<Me>("/auth/me"),
  })

  return (
    <>
      <PageHeader
        title="个人设置"
        description="管理管理员账号、本地登录密码和两步验证。敏感操作需要再次验证当前密码。"
      />
      {me.isLoading || !me.data ? (
        <PageLoading />
      ) : (
        <div className="flex w-full max-w-7xl flex-col gap-6">
          <div className="grid gap-6 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
            <UsernameCard me={me.data} />
            <PasswordCard me={me.data} />
          </div>
          <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1.65fr)_minmax(18rem,0.75fr)]">
            <TwoFactorCard me={me.data} />
            <Card>
              <CardHeader>
                <CardTitle>{t("Pocket ID 账号")}</CardTitle>
                <CardDescription>
                  {t(
                    "首次通过 Pocket ID 成功登录后会把外部身份固定绑定到当前管理员。"
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Badge
                  variant={me.data.pocketid_linked ? "default" : "secondary"}
                >
                  {t(me.data.pocketid_linked ? "已绑定" : "尚未绑定")}
                </Badge>
              </CardContent>
            </Card>
          </div>
        </div>
      )}
    </>
  )
}

function UsernameCard({ me }: { me: Me }) {
  const { t } = useI18n()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [username, setUsername] = useState(me.username)
  const [password, setPassword] = useState("")
  const update = useMutation({
    mutationFn: () =>
      api<{ username: string }>("/account/username", {
        method: "PUT",
        body: jsonBody({ username, current_password: password }),
      }),
    onSuccess: (value) => {
      client.setQueryData<Me>(["me"], (current) =>
        current ? { ...current, username: value.username } : current
      )
      setPassword("")
      setOpen(false)
      toast.success(t("用户名已更新"))
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("账号资料")}</CardTitle>
        <CardDescription>
          {t("修改用于本地登录的管理员用户名。")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <p className="font-medium">{me.username}</p>
      </CardContent>
      <CardFooter className="mt-auto">
        <Dialog
          open={open}
          onOpenChange={(nextOpen) => {
            setOpen(nextOpen)
            if (nextOpen) {
              setUsername(me.username)
            } else {
              setPassword("")
            }
          }}
        >
          <DialogTrigger render={<Button variant="outline" />}>
            <UserRoundIcon data-icon="inline-start" />
            {t("修改用户名")}
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("修改用户名")}</DialogTitle>
              <DialogDescription>
                {t("修改用于本地登录的管理员用户名。")}
              </DialogDescription>
            </DialogHeader>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="account-username">
                  {t("用户名")}
                </FieldLabel>
                <Input
                  id="account-username"
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  autoComplete="username"
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="username-password">
                  {t("当前密码")}
                </FieldLabel>
                <Input
                  id="username-password"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="current-password"
                />
              </Field>
            </FieldGroup>
            <DialogFooter>
              <DialogClose render={<Button variant="outline" />}>
                {t("取消")}
              </DialogClose>
              <Button
                disabled={
                  update.isPending || username.trim().length < 3 || !password
                }
                onClick={() => update.mutate()}
              >
                {update.isPending ? (
                  <LoaderCircleIcon data-icon="inline-start" />
                ) : (
                  <UserRoundIcon data-icon="inline-start" />
                )}
                {t("保存用户名")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </CardFooter>
    </Card>
  )
}

function PasswordCard({ me }: { me: Me }) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [otp, setOTP] = useState("")
  const update = useMutation({
    mutationFn: () =>
      api<void>("/account/password", {
        method: "PUT",
        body: jsonBody({
          current_password: currentPassword,
          new_password: newPassword,
          otp,
        }),
      }),
    onSuccess: () => {
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
      setOTP("")
      setOpen(false)
      toast.success(t("密码已更新，其他登录会话已退出"))
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("修改密码")}</CardTitle>
        <CardDescription>
          {t("新密码至少 8 个字符，保存后会退出其他设备上的会话。")}
        </CardDescription>
      </CardHeader>
      <CardFooter className="mt-auto">
        <Dialog
          open={open}
          onOpenChange={(nextOpen) => {
            setOpen(nextOpen)
            if (!nextOpen) {
              setCurrentPassword("")
              setNewPassword("")
              setConfirmPassword("")
              setOTP("")
            }
          }}
        >
          <DialogTrigger render={<Button variant="outline" />}>
            <KeyRoundIcon data-icon="inline-start" />
            {t("修改密码")}
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{t("修改密码")}</DialogTitle>
              <DialogDescription>
                {t("新密码至少 8 个字符，保存后会退出其他设备上的会话。")}
              </DialogDescription>
            </DialogHeader>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="current-password">
                  {t("当前密码")}
                </FieldLabel>
                <Input
                  id="current-password"
                  type="password"
                  value={currentPassword}
                  onChange={(event) => setCurrentPassword(event.target.value)}
                  autoComplete="current-password"
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="new-password">{t("新密码")}</FieldLabel>
                <Input
                  id="new-password"
                  type="password"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  autoComplete="new-password"
                  minLength={8}
                />
              </Field>
              <Field
                data-invalid={Boolean(
                  confirmPassword && confirmPassword !== newPassword
                )}
              >
                <FieldLabel htmlFor="confirm-password">
                  {t("确认新密码")}
                </FieldLabel>
                <Input
                  id="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  autoComplete="new-password"
                  minLength={8}
                  aria-invalid={Boolean(
                    confirmPassword && confirmPassword !== newPassword
                  )}
                />
              </Field>
              {me.two_factor_enabled ? (
                <Field>
                  <FieldLabel htmlFor="password-otp">
                    {t("验证码或恢复码")}
                  </FieldLabel>
                  <Input
                    id="password-otp"
                    value={otp}
                    onChange={(event) => setOTP(event.target.value)}
                    inputMode="numeric"
                    autoComplete="one-time-code"
                  />
                </Field>
              ) : null}
            </FieldGroup>
            <DialogFooter>
              <DialogClose render={<Button variant="outline" />}>
                {t("取消")}
              </DialogClose>
              <Button
                disabled={
                  update.isPending ||
                  !currentPassword ||
                  Array.from(newPassword).length < 8 ||
                  newPassword !== confirmPassword ||
                  (me.two_factor_enabled && !otp)
                }
                onClick={() => update.mutate()}
              >
                {update.isPending ? (
                  <LoaderCircleIcon data-icon="inline-start" />
                ) : (
                  <KeyRoundIcon data-icon="inline-start" />
                )}
                {t("更新密码")}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </CardFooter>
    </Card>
  )
}

function TwoFactorCard({ me }: { me: Me }) {
  const { t } = useI18n()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState("")
  const [code, setCode] = useState("")
  const [setup, setSetup] = useState<TOTPSetup | null>(null)
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])

  const resetDialog = () => {
    setPassword("")
    setCode("")
    setSetup(null)
    setRecoveryCodes([])
  }

  const begin = useMutation({
    mutationFn: () =>
      api<TOTPSetup>("/account/2fa/setup", {
        method: "POST",
        body: jsonBody({ current_password: password }),
      }),
    onSuccess: (value) => {
      setPassword("")
      setSetup(value)
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const enable = useMutation({
    mutationFn: () =>
      api<{ two_factor_enabled: boolean; recovery_codes: string[] }>(
        "/account/2fa/enable",
        { method: "POST", body: jsonBody({ code }) }
      ),
    onSuccess: (value) => {
      setRecoveryCodes(value.recovery_codes)
      setSetup(null)
      setCode("")
      client.setQueryData<Me>(["me"], (current) =>
        current ? { ...current, two_factor_enabled: true } : current
      )
      toast.success(t("两步验证已启用"))
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const disable = useMutation({
    mutationFn: () =>
      api<void>("/account/2fa/disable", {
        method: "POST",
        body: jsonBody({ current_password: password, code }),
      }),
    onSuccess: () => {
      client.setQueryData<Me>(["me"], (current) =>
        current ? { ...current, two_factor_enabled: false } : current
      )
      setOpen(false)
      resetDialog()
      toast.success(t("两步验证已关闭"))
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-4">
          <div className="flex flex-col gap-1.5">
            <CardTitle>{t("两步验证（2FA）")}</CardTitle>
            <CardDescription>
              {t("使用任意兼容 TOTP 的验证器，在本地密码登录时增加一层保护。")}
            </CardDescription>
          </div>
          <Badge variant={me.two_factor_enabled ? "default" : "secondary"}>
            {t(me.two_factor_enabled ? "已启用" : "未启用")}
          </Badge>
        </div>
      </CardHeader>
      <CardFooter className="mt-auto">
        <Dialog
          open={open}
          onOpenChange={(nextOpen) => {
            setOpen(nextOpen)
            if (!nextOpen) resetDialog()
          }}
        >
          <DialogTrigger
            render={
              <Button variant={me.two_factor_enabled ? "outline" : "default"} />
            }
          >
            {me.two_factor_enabled ? (
              <ShieldOffIcon data-icon="inline-start" />
            ) : (
              <ShieldCheckIcon data-icon="inline-start" />
            )}
            {t(
              me.two_factor_enabled ? "关闭两步验证" : "设置两步验证"
            )}
          </DialogTrigger>
          <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-xl">
            <DialogHeader>
              <DialogTitle>
                {t(
                  recoveryCodes.length > 0
                    ? "保存恢复码"
                    : setup
                      ? "扫描二维码"
                      : me.two_factor_enabled
                        ? "关闭两步验证"
                        : "启用两步验证"
                )}
              </DialogTitle>
              <DialogDescription>
                {t(
                  recoveryCodes.length > 0
                    ? "每个恢复码只能使用一次，关闭弹窗后将无法再次查看。"
                    : setup
                      ? "扫描二维码或复制密钥，然后输入验证器显示的 6 位验证码。"
                      : me.two_factor_enabled
                        ? "输入当前密码和验证器验证码以关闭两步验证。"
                        : "先输入当前密码验证身份。"
                )}
              </DialogDescription>
            </DialogHeader>

            {recoveryCodes.length > 0 ? (
              <Alert>
                <ShieldCheckIcon />
                <AlertTitle>{t("立即保存恢复码")}</AlertTitle>
                <AlertDescription className="flex flex-col gap-3">
                  <code className="grid gap-2 font-mono sm:grid-cols-2">
                    {recoveryCodes.map((recoveryCode) => (
                      <span key={recoveryCode}>{recoveryCode}</span>
                    ))}
                  </code>
                  <Button
                    variant="outline"
                    className="w-fit"
                    onClick={() => {
                      void navigator.clipboard.writeText(
                        recoveryCodes.join("\n")
                      )
                      toast.success(t("恢复码已复制"))
                    }}
                  >
                    <CopyIcon data-icon="inline-start" />
                    {t("复制恢复码")}
                  </Button>
                </AlertDescription>
              </Alert>
            ) : setup ? (
              <div className="grid items-start gap-6 sm:grid-cols-[12rem_minmax(0,1fr)]">
                <img
                  src={setup.qr_code_data_url}
                  alt={t("两步验证二维码")}
                  className="mx-auto size-48 rounded-lg border bg-white p-2"
                />
                <FieldGroup>
                  <Field>
                    <FieldLabel htmlFor="two-factor-secret">
                      {t("手动输入密钥")}
                    </FieldLabel>
                    <InputGroup>
                      <InputGroupInput
                        id="two-factor-secret"
                        value={setup.secret}
                        readOnly
                        className="font-mono"
                      />
                      <InputGroupAddon align="inline-end">
                        <InputGroupButton
                          size="icon-xs"
                          aria-label={t("复制密钥")}
                          onClick={() => {
                            void navigator.clipboard.writeText(setup.secret)
                            toast.success(t("密钥已复制"))
                          }}
                        >
                          <CopyIcon />
                        </InputGroupButton>
                      </InputGroupAddon>
                    </InputGroup>
                  </Field>
                  <Field>
                    <FieldLabel htmlFor="enable-otp">
                      {t("6 位验证码")}
                    </FieldLabel>
                    <Input
                      id="enable-otp"
                      value={code}
                      onChange={(event) => setCode(event.target.value)}
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      maxLength={6}
                    />
                  </Field>
                </FieldGroup>
              </div>
            ) : (
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="two-factor-password">
                    {t("当前密码")}
                  </FieldLabel>
                  <Input
                    id="two-factor-password"
                    type="password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    autoComplete="current-password"
                  />
                </Field>
                {me.two_factor_enabled ? (
                  <Field>
                    <FieldLabel htmlFor="disable-otp">
                      {t("验证码或恢复码")}
                    </FieldLabel>
                    <Input
                      id="disable-otp"
                      value={code}
                      onChange={(event) => setCode(event.target.value)}
                      autoComplete="one-time-code"
                    />
                  </Field>
                ) : null}
              </FieldGroup>
            )}

            <DialogFooter>
              {recoveryCodes.length > 0 ? (
                <DialogClose render={<Button />}>
                  {t("完成")}
                </DialogClose>
              ) : (
                <>
                  <DialogClose render={<Button variant="outline" />}>
                    {t("取消")}
                  </DialogClose>
                  {setup ? (
                    <Button
                      disabled={enable.isPending || code.length !== 6}
                      onClick={() => enable.mutate()}
                    >
                      {enable.isPending ? (
                        <LoaderCircleIcon data-icon="inline-start" />
                      ) : (
                        <ShieldCheckIcon data-icon="inline-start" />
                      )}
                      {t("确认并启用")}
                    </Button>
                  ) : me.two_factor_enabled ? (
                    <Button
                      variant="destructive"
                      disabled={disable.isPending || !password || !code}
                      onClick={() => disable.mutate()}
                    >
                      {disable.isPending ? (
                        <LoaderCircleIcon data-icon="inline-start" />
                      ) : (
                        <ShieldOffIcon data-icon="inline-start" />
                      )}
                      {t("确认关闭")}
                    </Button>
                  ) : (
                    <Button
                      disabled={begin.isPending || !password}
                      onClick={() => begin.mutate()}
                    >
                      {begin.isPending ? (
                        <LoaderCircleIcon data-icon="inline-start" />
                      ) : (
                        <KeyRoundIcon data-icon="inline-start" />
                      )}
                      {t("验证并继续")}
                    </Button>
                  )}
                </>
              )}
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </CardFooter>
    </Card>
  )
}
