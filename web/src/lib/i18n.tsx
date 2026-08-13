import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"

export type Language = "zh-CN" | "en"

const english: Record<string, string> = {
  消息中转中心: "Message relay center",
  工作台: "Workspace",
  运维: "Operations",
  概览: "Overview",
  通知渠道: "Notification channels",
  消息模板: "Message templates",
  目标组: "Target groups",
  "Webhook 入口": "Webhook sources",
  路由规则: "Routing rules",
  定时任务: "Schedules",
  执行记录: "Activity",
  系统设置: "System settings",
  退出登录: "Sign out",
  可靠地把每一次事件送到正确的位置:
    "Deliver every event reliably to the right destination",
  主题: "Theme",
  浅色: "Light",
  深色: "Dark",
  跟随系统: "System",
  语言: "Language",
  简体中文: "Simplified Chinese",
  英语: "English",
  切换菜单: "Toggle menu",
  暂无数据: "No data yet",
  "使用页面右上角的按钮创建第一条记录。":
    "Use the button in the top-right corner to create the first record.",
  正在连接服务: "Connecting to the service…",
  "正在连接服务…": "Connecting to the service…",
  "初始化 PushRelay": "Set up PushRelay",
  "这是唯一一次创建管理员的机会。Setup Token 会显示在后端启动日志中。":
    "This is the only opportunity to create the administrator. The Setup Token is shown in the backend startup logs.",
  管理员账号: "Administrator username",
  管理员密码: "Administrator password",
  "至少 12 个字符，将使用 Argon2id 加密保存。":
    "At least 12 characters. It will be stored using Argon2id.",
  创建管理员: "Create administrator",
  管理员创建成功: "Administrator created",
  欢迎回来: "Welcome back",
  "登录你的自托管消息中转后台。":
    "Sign in to your self-hosted message relay console.",
  账号: "Username",
  密码: "Password",
  登录: "Sign in",
  登录失败: "Sign-in failed",
  自托管消息路由器: "Self-hosted message router",
  消息中枢: "Message hub",
  "从入口、规则到最终投递，一眼掌握当前运行状态。":
    "See the current state from source and routing through final delivery.",
  今日事件: "Events today",
  今天接收和触发的事件: "Events received and triggered today",
  等待投递: "Pending deliveries",
  "已入队、等待 Worker 处理": "Queued and waiting for a worker",
  死信任务: "Dead-letter jobs",
  达到最大重试次数: "Maximum retry attempts reached",
  已配置的渠道连接: "Configured channel connections",
  可接收事件的公开入口: "Public sources that can receive events",
  启用定时: "Active schedules",
  正在运行的定时任务: "Currently active schedules",
  名称: "Name",
  类型: "Type",
  状态: "Status",
  创建时间: "Created",
  操作: "Actions",
  启用: "Enabled",
  停用: "Disabled",
  删除: "Delete",
  保存: "Save",
  取消: "Cancel",
  预览: "Preview",
  渠道: "Channel",
  渠道类型: "Channel type",
  模板: "Template",
  目标: "Target",
  数量: "Count",
  条件: "Condition",
  优先级: "Priority",
  入口: "Source",
  事件: "Events",
  方法: "Method",
  尝试: "Attempts",
  最近错误: "Latest error",
  接收时间: "Received",
  触发方式: "Trigger",
  命中规则: "Matched rules",
  载荷策略: "Payload policy",
  投递任务: "Delivery jobs",
  重投: "Retry",
  已重新加入队列: "Added back to the queue",
  运行设置: "Runtime settings",
  运行与安全: "Runtime and security",
  "管理默认时区、数据保留、安全策略和 Pocket ID 登录。":
    "Manage the default timezone, data retention, security policies, and Pocket ID sign-in.",
  "这些配置会被调度器、清理任务和出站 Webhook Worker 动态读取。":
    "The scheduler, cleanup task, and outbound Webhook workers read these settings dynamically.",
  "调度、日志保留和出站 Webhook 的全局运行策略。":
    "Global policies for scheduling, retention, and outbound Webhooks.",
  系统默认时区: "Default system timezone",
  "输入城市或区域名称搜索，例如 Shanghai、Berlin、New_York。":
    "Search by city or region, such as Shanghai, Berlin, or New_York.",
  消息正文保留天数: "Payload retention days",
  "设为 0 表示投递完成后不长期保留正文。":
    "Set to 0 to avoid retaining payloads after delivery completes.",
  事件元数据保留天数: "Metadata retention days",
  "正文保留天数不能超过元数据保留天数。":
    "Payload retention cannot exceed metadata retention.",
  "允许通用 Webhook 访问私网":
    "Allow generic Webhooks to access private networks",
  "允许环回、局域网和链路本地地址，仅建议用于可信内网。":
    "Allows loopback, LAN, and link-local addresses. Use only in trusted networks.",
  私网访问已开启: "Private-network access is enabled",
  "通用出站 Webhook 可以连接本机和内网服务，请只向可信管理员开放后台。":
    "Generic outbound Webhooks can reach local and private services. Restrict console access to trusted administrators.",
  保存设置: "Save settings",
  "保存中…": "Saving…",
  系统设置已保存并立即生效: "Settings saved and applied",
  "搜索或选择 IANA 时区": "Search or select an IANA timezone",
  未找到匹配的时区: "No matching timezone found",
  系统默认: "System default",
  "追踪每个事件的规则命中、渠道投递、重试次数和最终状态。":
    "Track rule matches, channel deliveries, retries, and final status for every event.",
  "事件 ID": "Event ID",
  success: "Success",
  dead: "Dead",
  pending: "Pending",
  processing: "Processing",
  retry: "Retrying",
  webhook: "Webhook",
  schedule: "Schedule",
  redacted: "Redacted",
  metadata: "Metadata only",
  none: "No payload",
  all_match: "All matches",
  first_match: "First match",
  "集中保管机器人、应用和通用 Webhook 的连接凭据。Secret 只会在服务端解密使用。":
    "Manage credentials for bots, applications, and generic Webhooks. Secrets are decrypted only on the server.",
  添加渠道: "Add channel",
  添加通知渠道: "Add notification channel",
  "选择渠道类型并填写其官方 API 所需凭据。":
    "Choose a channel type and enter the credentials required by its official API.",
  渠道已创建: "Channel created",
  测试消息已发送: "Test message sent",
  测试: "Test",
  删除渠道: "Delete channel",
  生产告警群: "Production alerts",
  选择渠道: "Select a channel",
  创建后立即启用: "Enable immediately after creation",
  保存渠道: "Save channel",
  "PushRelay 渠道连接测试": "PushRelay channel connection test",
  钉钉群机器人: "DingTalk group bot",
  企业微信群机器人: "WeCom group bot",
  企业微信应用: "WeCom application",
  飞书群机器人: "Feishu group bot",
  "通用 Webhook": "Generic Webhook",
  "将“渠道 + 模板”组合为可复用的投递目标，规则和定时任务只需引用一次。":
    "Combine a channel and template into reusable delivery targets for rules and schedules.",
  新建目标组: "New target group",
  投递绑定: "Delivery bindings",
  删除目标组: "Delete target group",
  "首个版本可在创建时添加一个绑定，之后可通过更新 API 提交多个绑定。":
    "Add one binding during creation; additional bindings can be submitted through the update API.",
  选择相同渠道类型的模板: "Select a template for the same channel type",
  "模板类型必须与所选渠道一致。":
    "The template type must match the selected channel.",
  保存目标组: "Save target group",
  目标组已创建: "Target group created",
  "使用受限条件树匹配正文、Header、查询参数和元数据，不执行任意脚本。":
    "Match body, headers, query parameters, and metadata with a restricted condition tree. No scripts are executed.",
  新建规则: "New rule",
  新建路由规则: "New routing rule",
  删除规则: "Delete rule",
  "当前表单创建单条件规则；管理 API 支持最多 5 层 AND/OR 嵌套条件树。":
    "This form creates a single-condition rule. The management API supports AND/OR trees up to five levels deep.",
  字段路径: "Field path",
  "例如 body.status、headers.x-event-type、query.environment。":
    "For example: body.status, headers.x-event-type, or query.environment.",
  运算符: "Operator",
  比较值: "Comparison value",
  保存规则: "Save rule",
  规则已创建: "Rule created",
  选择: "Select ",
  "所有时间按任务自己的 IANA 时区计算，数据库统一保存 UTC，精确到分钟。":
    "Times are evaluated in each task's IANA timezone and stored in UTC with minute precision.",
  新建定时任务: "New schedule",
  周期: "Recurrence",
  时区: "Timezone",
  下次执行: "Next run",
  删除定时任务: "Delete schedule",
  "日、周、月、年提供可视化配置；高级场景可使用标准 5 段 Cron。":
    "Configure daily, weekly, monthly, and yearly schedules visually, or use a standard five-field Cron expression.",
  选择目标组: "Select a target group",
  重复周期: "Recurrence",
  每天: "Daily",
  每周: "Weekly",
  每月: "Monthly",
  每年: "Yearly",
  "5 段 Cron": "5-field Cron",
  单次: "Once",
  daily: "Daily",
  weekly: "Weekly",
  monthly: "Monthly",
  yearly: "Yearly",
  cron: "Cron",
  once: "Once",
  "IANA 时区": "IANA timezone",
  "留空使用后台设置的系统时区；也可填写 Asia/Shanghai、Europe/Berlin。":
    "Leave blank to use the system timezone, or enter Asia/Shanghai or Europe/Berlin.",
  "标准 5 段：分钟 小时 日 月 星期。":
    "Standard five fields: minute, hour, day, month, weekday.",
  执行时间: "Run time",
  间隔: "Interval",
  日期: "Day",
  月份: "Month",
  "事件正文 JSON": "Event payload JSON",
  保存定时任务: "Save schedule",
  定时任务已创建: "Schedule created",
  "每个入口拥有不可猜测令牌，可选 HMAC、IP 白名单、载荷留存策略和规则匹配模式。":
    "Each source has an unguessable token with optional HMAC, IP allowlists, payload retention, and rule matching modes.",
  创建入口: "Create source",
  令牌: "Token",
  规则模式: "Rule mode",
  载荷日志: "Payload logging",
  轮换令牌: "Rotate token",
  删除入口: "Delete source",
  "创建 Webhook 入口": "Create Webhook source",
  "入口 URL 和 HMAC Secret 仅在创建时完整显示，请立即保存。":
    "The source URL and HMAC Secret are shown in full only once. Save them now.",
  请立即保存凭据: "Save these credentials now",
  "关闭弹窗后将无法再次查看完整 Token 或 HMAC Secret。":
    "After closing this dialog, the full Token and HMAC Secret cannot be viewed again.",
  入口地址: "Source URL",
  凭据已复制: "Credentials copied",
  复制凭据: "Copy credentials",
  规则命中模式: "Rule matching mode",
  执行全部命中规则: "Run all matching rules",
  仅执行第一条规则: "Run only the first matching rule",
  脱敏后保存: "Store after redaction",
  仅保存元数据: "Store metadata only",
  不保存正文: "Do not store payloads",
  "IP / CIDR 白名单": "IP / CIDR allowlist",
  "留空表示允许所有来源 IP。": "Leave blank to allow all source IPs.",
  额外脱敏字段: "Additional sensitive fields",
  "按字段名匹配，多个名称使用逗号分隔。":
    "Match by field name; separate multiple names with commas.",
  "启用 HMAC-SHA256 签名验证": "Enable HMAC-SHA256 signature verification",
  "入口令牌已轮换，新地址已复制": "Source token rotated and the new URL copied",
  "逐字段渲染安全模板，把统一事件上下文转换为各个平台所需的消息结构。":
    "Safely render each field and transform the unified event context into each platform's message format.",
  新建模板: "New template",
  新建消息模板: "New message template",
  删除模板: "Delete template",
  "模板字符串支持 get、default、toJSON、truncate、dateInZone 和 urlquery。":
    "Template strings support get, default, toJSON, truncate, dateInZone, and urlquery.",
  "模板 JSON": "Template JSON",
  "所有字段在渲染后才会统一序列化，避免手工 JSON 转义问题。":
    "Fields are serialized only after rendering to avoid manual JSON escaping issues.",
  保存模板: "Save template",
  模板已创建: "Template created",
  个人设置: "Personal settings",
  "管理管理员账号、本地登录密码和两步验证。敏感操作需要再次验证当前密码。":
    "Manage the administrator account, local password, and two-factor authentication. Sensitive changes require your current password.",
  账号资料: "Account profile",
  "修改用于本地登录的管理员用户名。":
    "Change the administrator username used for local sign-in.",
  用户名: "Username",
  当前密码: "Current password",
  修改用户名: "Change username",
  保存用户名: "Save username",
  用户名已更新: "Username updated",
  修改密码: "Change password",
  "新密码至少 12 个字符，保存后会退出其他设备上的会话。":
    "The new password must contain at least 12 characters. Other device sessions will be signed out.",
  新密码: "New password",
  确认新密码: "Confirm new password",
  验证码或恢复码: "Verification or recovery code",
  更新密码: "Update password",
  "密码已更新，其他登录会话已退出":
    "Password updated and other sessions signed out",
  "两步验证（2FA）": "Two-factor authentication (2FA)",
  "使用任意兼容 TOTP 的验证器，在本地密码登录时增加一层保护。":
    "Use any TOTP-compatible authenticator to add protection to local password sign-in.",
  已启用: "Enabled",
  未启用: "Not enabled",
  设置两步验证: "Set up two-factor authentication",
  启用两步验证: "Enable two-factor authentication",
  两步验证已启用: "Two-factor authentication enabled",
  两步验证已关闭: "Two-factor authentication disabled",
  关闭两步验证: "Disable two-factor authentication",
  手动输入密钥: "Manual setup key",
  "6 位验证码": "6-digit verification code",
  "扫描二维码后，输入验证器当前显示的验证码以完成启用。":
    "Scan the QR code, then enter the current code from your authenticator to finish setup.",
  确认并启用: "Confirm and enable",
  确认关闭: "Disable",
  验证并继续: "Verify and continue",
  两步验证二维码: "Two-factor authentication QR code",
  扫描二维码: "Scan QR code",
  "扫描二维码或复制密钥，然后输入验证器显示的 6 位验证码。":
    "Scan the QR code or copy the setup key, then enter the 6-digit code from your authenticator.",
  "先输入当前密码验证身份。":
    "Enter your current password to verify your identity.",
  "输入当前密码和验证器验证码以关闭两步验证。":
    "Enter your current password and authenticator code to disable two-factor authentication.",
  复制密钥: "Copy setup key",
  密钥已复制: "Setup key copied",
  立即保存恢复码: "Save your recovery codes now",
  保存恢复码: "Save recovery codes",
  "每个恢复码只能使用一次，离开此页面后将无法再次查看。":
    "Each recovery code can be used once. You cannot view them again after leaving this page.",
  "每个恢复码只能使用一次，关闭弹窗后将无法再次查看。":
    "Each recovery code can be used once. You cannot view them again after closing this dialog.",
  复制恢复码: "Copy recovery codes",
  恢复码已复制: "Recovery codes copied",
  完成: "Done",
  "Pocket ID 账号": "Pocket ID account",
  "首次通过 Pocket ID 成功登录后会把外部身份固定绑定到当前管理员。":
    "The first successful Pocket ID sign-in permanently links the external identity to this administrator.",
  已绑定: "Linked",
  尚未绑定: "Not linked",
  "使用 Pocket ID 的 OIDC Discovery、授权码流程和 PKCE 登录管理员账号。":
    "Use Pocket ID OIDC discovery, authorization code flow, and PKCE to sign in as the administrator.",
  "启用 Pocket ID 登录": "Enable Pocket ID sign-in",
  "启用前必须填写下面所有配置，并在 Pocket ID 中登记回调地址。":
    "Complete every field below and register the callback URL in Pocket ID before enabling it.",
  回调地址: "Callback URL",
  复制回调地址: "Copy callback URL",
  回调地址已复制: "Callback URL copied",
  "把这个完整地址添加到 Pocket ID OIDC Client 的 Callback URLs。":
    "Add this complete URL to the Pocket ID OIDC client's callback URLs.",
  "Issuer / Discovery URL": "Issuer / Discovery URL",
  "填写 Pocket ID 实例地址；也可以粘贴其 OIDC Discovery URL。":
    "Enter the Pocket ID instance URL or paste its OIDC discovery URL.",
  "已安全保存，留空表示不修改":
    "Stored securely; leave blank to keep it unchanged",
  "输入 Pocket ID Client Secret": "Enter the Pocket ID client secret",
  允许绑定的用户名或邮箱: "Allowed username or email",
  "首次登录必须与 Pocket ID 的 preferred_username 或已验证邮箱完全匹配，之后固定绑定其 subject。":
    "The first sign-in must exactly match the Pocket ID preferred_username or verified email. Its subject is then permanently linked.",
  验证并登录: "Verify and sign in",
  返回账号密码: "Back to username and password",
  或者: "or",
  "使用 Pocket ID 登录": "Sign in with Pocket ID",
  "输入验证器中的 6 位验证码，或一个尚未使用的恢复码。":
    "Enter the 6-digit code from your authenticator or an unused recovery code.",
}

type I18nContextValue = {
  language: Language
  setLanguage: (language: Language) => void
  t: (source: string) => string
}

const I18nContext = createContext<I18nContextValue | null>(null)
const storageKey = "pushrelay-language"

function initialLanguage(): Language {
  return localStorage.getItem(storageKey) === "en" ? "en" : "zh-CN"
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>(initialLanguage)

  const setLanguage = useCallback((nextLanguage: Language) => {
    localStorage.setItem(storageKey, nextLanguage)
    document.documentElement.lang = nextLanguage
    setLanguageState(nextLanguage)
  }, [])

  useEffect(() => {
    document.documentElement.lang = language
  }, [language])

  const t = useCallback(
    (source: string) =>
      language === "en" ? (english[source] ?? source) : source,
    [language]
  )

  const value = useMemo(
    () => ({ language, setLanguage, t }),
    [language, setLanguage, t]
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const context = useContext(I18nContext)
  if (!context) throw new Error("useI18n must be used inside I18nProvider")
  return context
}
