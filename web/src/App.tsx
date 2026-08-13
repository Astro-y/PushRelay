import { lazy, Suspense } from "react"
import { Route, Routes } from "react-router-dom"
import { AppShell } from "@/components/app-shell"
import { AuthGate, LoginPage, SetupPage } from "@/pages/auth"
import { PageLoading } from "@/components/page"

const ActivityPage = lazy(() =>
  import("@/pages/activity").then((module) => ({
    default: module.ActivityPage,
  }))
)
const AccountPage = lazy(() =>
  import("@/pages/account").then((module) => ({
    default: module.AccountPage,
  }))
)
const ChannelsPage = lazy(() =>
  import("@/pages/channels").then((module) => ({
    default: module.ChannelsPage,
  }))
)
const DashboardPage = lazy(() =>
  import("@/pages/dashboard").then((module) => ({
    default: module.DashboardPage,
  }))
)
const GroupsPage = lazy(() =>
  import("@/pages/groups").then((module) => ({ default: module.GroupsPage }))
)
const RulesPage = lazy(() =>
  import("@/pages/rules").then((module) => ({ default: module.RulesPage }))
)
const SchedulesPage = lazy(() =>
  import("@/pages/schedules").then((module) => ({
    default: module.SchedulesPage,
  }))
)
const SettingsPage = lazy(() =>
  import("@/pages/settings").then((module) => ({
    default: module.SettingsPage,
  }))
)
const SourcesPage = lazy(() =>
  import("@/pages/sources").then((module) => ({ default: module.SourcesPage }))
)
const TemplatesPage = lazy(() =>
  import("@/pages/templates").then((module) => ({
    default: module.TemplatesPage,
  }))
)

export default function App() {
  return (
    <Suspense
      fallback={
        <main className="p-6">
          <PageLoading />
        </main>
      }
    >
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/setup" element={<SetupPage />} />
        <Route
          element={
            <AuthGate>
              <AppShell />
            </AuthGate>
          }
        >
          <Route index element={<DashboardPage />} />
          <Route path="channels" element={<ChannelsPage />} />
          <Route path="templates" element={<TemplatesPage />} />
          <Route path="groups" element={<GroupsPage />} />
          <Route path="sources" element={<SourcesPage />} />
          <Route path="rules" element={<RulesPage />} />
          <Route path="schedules" element={<SchedulesPage />} />
          <Route path="activity" element={<ActivityPage />} />
          <Route path="account" element={<AccountPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </Suspense>
  )
}
