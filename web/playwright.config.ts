import { defineConfig } from "@playwright/test"
import { fileURLToPath } from "node:url"

const apiURL = "http://127.0.0.1:18081"
const webURL = "http://127.0.0.1:5174"
const goCache = fileURLToPath(
  new URL("../.cache/playwright-go", import.meta.url)
)

export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  fullyParallel: false,
  workers: 1,
  reporter: process.env.CI ? "github" : "list",
  use: { baseURL: webURL, trace: "retain-on-failure" },
  webServer: [
    {
      command: "go run ../cmd/server",
      url: `${apiURL}/healthz`,
      timeout: 120_000,
      reuseExistingServer: !process.env.CI,
      env: {
        APP_ENCRYPTION_KEY: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
        SETUP_TOKEN: "playwright-setup-token",
        HTTP_ADDR: ":18081",
        DATABASE_PATH: "../data/playwright.db",
        WEB_ORIGIN: webURL,
        WORKER_CONCURRENCY: "2",
        GOPATH: `${goCache}/gopath`,
        GOMODCACHE: `${goCache}/gomod`,
        GOCACHE: `${goCache}/build`,
      },
    },
    {
      command: "node e2e/mock-server.mjs",
      url: "http://127.0.0.1:19090/healthz",
      timeout: 120_000,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: "pnpm run dev --host 127.0.0.1 --port 5174",
      url: webURL,
      timeout: 120_000,
      reuseExistingServer: !process.env.CI,
      env: { VITE_API_URL: apiURL },
    },
  ],
})
