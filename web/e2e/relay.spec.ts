import { expect, request as createRequest, test } from "@playwright/test"

const apiURL = "http://127.0.0.1:18081"
const webURL = "http://127.0.0.1:5174"
const password = "correct horse battery staple"

test("setup, relay a webhook, and execute a one-time schedule", async ({
  page,
}) => {
  const api = await createRequest.newContext({
    baseURL: apiURL,
    extraHTTPHeaders: { Origin: webURL },
  })
  const setupStatus = await api.get("/api/v1/setup/status")
  if ((await setupStatus.json()).required) {
    await page.goto("/setup")
    await page.getByLabel("Setup Token").fill("playwright-setup-token")
    await page.getByLabel("管理员账号").fill("admin")
    await page.getByLabel("管理员密码").fill(password)
    await page.getByRole("button", { name: "创建管理员" }).click()
    await expect(page).toHaveURL(/\/login$/)
  }

  await page.goto("/login")
  await page.getByLabel("账号").fill("admin")
  await page.getByLabel("密码").fill(password)
  await page.getByRole("button", { name: "登录" }).click()
  await expect(page.getByRole("heading", { name: "消息中枢" })).toBeVisible()

  await page.goto("/account")
  await page.getByRole("button", { name: "修改用户名" }).click()
  await expect(
    page.getByRole("dialog").getByRole("heading", { name: "修改用户名" })
  ).toBeVisible()
  await page.getByRole("dialog").getByRole("button", { name: "取消" }).click()

  await page.getByRole("button", { name: "修改密码" }).click()
  await expect(
    page.getByRole("dialog").getByRole("heading", { name: "修改密码" })
  ).toBeVisible()
  await page.getByRole("dialog").getByRole("button", { name: "取消" }).click()

  await page.getByRole("button", { name: "设置两步验证" }).click()
  const twoFactorDialog = page.getByRole("dialog")
  await expect(
    twoFactorDialog.getByRole("heading", { name: "启用两步验证" })
  ).toBeVisible()
  await twoFactorDialog.getByLabel("当前密码").fill(password)
  await twoFactorDialog.getByRole("button", { name: "验证并继续" }).click()
  await expect(
    twoFactorDialog.getByRole("heading", { name: "扫描二维码" })
  ).toBeVisible()
  await expect(twoFactorDialog.getByAltText("两步验证二维码")).toBeVisible()
  await expect(
    twoFactorDialog.getByRole("button", { name: "复制密钥" })
  ).toBeVisible()
  await twoFactorDialog.getByRole("button", { name: "取消" }).click()

  await page.goto("/settings")
  await page.getByLabel("系统默认时区").fill("Berlin")
  await page.getByRole("option", { name: "Europe/Berlin" }).click()
  await page.getByLabel("消息正文保留天数").fill("7")
  await page.getByLabel("事件元数据保留天数").fill("30")
  const privateTargets = page.getByRole("switch", {
    name: "允许通用 Webhook 访问私网",
  })
  if ((await privateTargets.getAttribute("aria-checked")) !== "true") {
    await privateTargets.click()
  }
  await page.getByRole("button", { name: "保存设置" }).click()
  await expect(page.getByText("系统设置已保存并立即生效")).toBeVisible()

  await page.getByRole("button", { name: "主题" }).click()
  await page.getByRole("menuitemradio", { name: "深色" }).click()
  await expect(page.locator("html")).toHaveClass(/dark/)

  await page.getByRole("button", { name: "语言" }).click()
  await page.getByRole("menuitemradio", { name: "English" }).click()
  await expect(
    page.getByRole("heading", { name: "System settings" })
  ).toBeVisible()

  for (const route of [
    "/",
    "/channels",
    "/templates",
    "/groups",
    "/sources",
    "/rules",
    "/schedules",
    "/activity",
    "/account",
    "/settings",
  ]) {
    await page.goto(route)
    await expect(page.locator('[data-slot="sidebar-wrapper"]')).toBeVisible()
    expect(await page.locator("body").innerText()).not.toMatch(
      /[\u4e00-\u9fff]/
    )
  }

  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto("/")
  await page.locator('[data-sidebar="trigger"]').click()
  const mobileSidebar = page.locator('[data-mobile="true"]')
  await expect(mobileSidebar).toBeVisible()
  await mobileSidebar
    .getByRole("link", { name: "Notification channels" })
    .click()
  await expect(page).toHaveURL(/\/channels$/)
  await expect(mobileSidebar).toBeHidden()

  const login = await api.post("/api/v1/auth/login", {
    data: { username: "admin", password },
  })
  expect(login.ok()).toBeTruthy()
  const { csrf_token: csrf } = await login.json()
  const headers = { "X-CSRF-Token": csrf }

  const channel = await api.post("/api/v1/channels", {
    headers,
    data: {
      name: "E2E receiver",
      type: "webhook",
      enabled: true,
      config: { webhook: "http://127.0.0.1:19090/notify" },
    },
  })
  const channelBody = await channel.json()
  const template = await api.post("/api/v1/templates", {
    headers,
    data: {
      name: "E2E template",
      channel_type: "webhook",
      body: { method: "POST", body: { text: '{{ get "body.message" }}' } },
    },
  })
  const templateBody = await template.json()
  const group = await api.post("/api/v1/target-groups", {
    headers,
    data: {
      name: "E2E targets",
      bindings: [
        {
          channel_id: channelBody.id,
          template_id: templateBody.id,
          enabled: true,
        },
      ],
    },
  })
  const groupBody = await group.json()
  const source = await api.post("/api/v1/sources", {
    headers,
    data: {
      name: "E2E source",
      hmac_enabled: false,
      allowed_cidrs: [],
      custom_sensitive_fields: ["customer_code"],
      match_mode: "all_match",
      payload_policy: "redacted",
      enabled: true,
    },
  })
  const sourceBody = await source.json()
  await api.post("/api/v1/rules", {
    headers,
    data: {
      name: "Errors",
      source_id: sourceBody.source.id,
      target_group_id: groupBody.id,
      priority: 10,
      enabled: true,
      condition: { path: "body.status", operator: "eq", value: "error" },
    },
  })

  const accepted = await api.post(sourceBody.hook_url, {
    headers: { "Idempotency-Key": "playwright-webhook" },
    data: { status: "error", message: "relay me", customer_code: "secret" },
  })
  expect(accepted.status()).toBe(202)
  await expect
    .poll(
      async () =>
        (await (await api.get("/api/v1/deliveries", { headers })).json())[0]
          ?.status,
      { timeout: 20_000 }
    )
    .toBe("success")

  const runAt = new Date(Date.now() + 5_000).toISOString()
  const scheduled = await api.post("/api/v1/schedules", {
    headers,
    data: {
      name: "E2E once",
      recurrence: { kind: "once", run_at: runAt },
      timezone: "Asia/Shanghai",
      payload: { message: "scheduled" },
      target_group_id: groupBody.id,
      enabled: true,
    },
  })
  expect(scheduled.status()).toBe(201)
  await expect
    .poll(
      async () =>
        (
          await (await api.get("/api/v1/events?limit=20", { headers })).json()
        ).some(
          (event: { trigger_type: string }) => event.trigger_type === "schedule"
        ),
      { timeout: 25_000 }
    )
    .toBeTruthy()
  await api.dispose()
})
