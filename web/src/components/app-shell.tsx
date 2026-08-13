import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom"
import { useMutation } from "@tanstack/react-query"
import { flushSync } from "react-dom"
import {
  ActivityIcon,
  BellRingIcon,
  BoxesIcon,
  CalendarClockIcon,
  GaugeIcon,
  LanguagesIcon,
  LaptopIcon,
  LogOutIcon,
  MoonIcon,
  RouteIcon,
  SendIcon,
  SettingsIcon,
  SunIcon,
  SunMoonIcon,
  UserRoundCogIcon,
  WebhookIcon,
} from "lucide-react"
import { api, setCSRF } from "@/lib/api"
import { BrandMark } from "@/components/brand-mark"
import { useI18n, type Language } from "@/lib/i18n"
import { useTheme } from "@/components/theme-provider"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Separator } from "@/components/ui/separator"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar"

const primary = [
  { to: "/", label: "概览", icon: GaugeIcon },
  { to: "/channels", label: "通知渠道", icon: SendIcon },
  { to: "/templates", label: "消息模板", icon: BellRingIcon },
  { to: "/groups", label: "目标组", icon: BoxesIcon },
  { to: "/sources", label: "Webhook 入口", icon: WebhookIcon },
  { to: "/rules", label: "路由规则", icon: RouteIcon },
  { to: "/schedules", label: "定时任务", icon: CalendarClockIcon },
]
const secondary = [
  { to: "/activity", label: "执行记录", icon: ActivityIcon },
  { to: "/account", label: "个人设置", icon: UserRoundCogIcon },
  { to: "/settings", label: "系统设置", icon: SettingsIcon },
]

export function AppShell() {
  return (
    <SidebarProvider>
      <ShellContent />
    </SidebarProvider>
  )
}

function ShellContent() {
  const location = useLocation()
  const navigate = useNavigate()
  const { t } = useI18n()
  const logout = useMutation({
    mutationFn: () => api<void>("/auth/logout", { method: "POST" }),
    onSuccess: () => {
      setCSRF("")
      navigate("/login")
    },
  })

  return (
    <>
      <Sidebar collapsible="icon">
        <SidebarHeader>
          <div className="flex items-center gap-3 px-2 py-1">
            <BrandMark className="size-8" />
            <div className="grid min-w-0 flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
              <span className="truncate font-semibold">PushRelay</span>
              <span className="truncate text-xs text-muted-foreground">
                {t("消息中转中心")}
              </span>
            </div>
          </div>
        </SidebarHeader>
        <SidebarContent>
          <NavGroup label="工作台" items={primary} path={location.pathname} />
          <NavGroup label="运维" items={secondary} path={location.pathname} />
        </SidebarContent>
        <SidebarFooter>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton onClick={() => logout.mutate()}>
                <LogOutIcon />
                <span>{t("退出登录")}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
      <SidebarInset>
        <header className="flex h-14 shrink-0 items-center gap-3 px-4">
          <SidebarTrigger aria-label={t("切换菜单")} />
          <Separator orientation="vertical" className="h-4" />
          <span className="hidden truncate text-sm text-muted-foreground sm:block">
            {t("可靠地把每一次事件送到正确的位置")}
          </span>
          <div className="ml-auto flex items-center gap-1">
            <ThemeMenu />
            <LanguageMenu />
          </div>
        </header>
        <main className="flex flex-1 flex-col gap-6 p-4 pt-2 md:p-6 md:pt-3">
          <Outlet />
        </main>
      </SidebarInset>
    </>
  )
}

function NavGroup({
  label,
  items,
  path,
}: {
  label: string
  items: typeof primary
  path: string
}) {
  const { isMobile, setOpenMobile } = useSidebar()
  const { t } = useI18n()
  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t(label)}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {items.map(({ to, label: itemLabel, icon: Icon }) => (
            <SidebarMenuItem key={to}>
              <SidebarMenuButton
                render={
                  <NavLink
                    to={to}
                    onClick={() => {
                      if (isMobile) setOpenMobile(false)
                    }}
                  />
                }
                isActive={to === "/" ? path === "/" : path.startsWith(to)}
              >
                <Icon />
                <span>{t(itemLabel)}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}

function ThemeMenu() {
  const { theme, setTheme } = useTheme()
  const { t } = useI18n()

  function changeTheme(nextTheme: string) {
    if (nextTheme === theme) return

    const reduceMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)"
    ).matches
    if (reduceMotion || typeof document.startViewTransition !== "function") {
      setTheme(nextTheme)
      return
    }

    document.startViewTransition(() => {
      flushSync(() => setTheme(nextTheme))
    })
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button variant="ghost" size="icon-sm" />}
        aria-label={t("主题")}
      >
        <SunMoonIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t("主题")}</DropdownMenuLabel>
          <DropdownMenuRadioGroup value={theme} onValueChange={changeTheme}>
            <DropdownMenuRadioItem value="light" closeOnClick>
              <SunIcon />
              {t("浅色")}
            </DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="dark" closeOnClick>
              <MoonIcon />
              {t("深色")}
            </DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="system" closeOnClick>
              <LaptopIcon />
              {t("跟随系统")}
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function LanguageMenu() {
  const { language, setLanguage, t } = useI18n()
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button variant="ghost" size="icon-sm" />}
        aria-label={t("语言")}
      >
        <LanguagesIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t("语言")}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuRadioGroup
            value={language}
            onValueChange={(value) => setLanguage(value as Language)}
          >
            <DropdownMenuRadioItem value="zh-CN" closeOnClick>
              简体中文
            </DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="en" closeOnClick>
              English
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
