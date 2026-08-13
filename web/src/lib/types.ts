export type Dashboard = {
  channels: number
  sources: number
  schedules: number
  pending: number
  failed: number
  events_today: number
}
export type Channel = {
  id: string
  name: string
  type: string
  config: Record<string, unknown>
  enabled: boolean
  created_at: number
}
export type ChannelType = {
  type: string
  label: string
  required: string[]
  optional?: string[]
}
export type MessageTemplate = {
  id: string
  name: string
  channel_type: string
  body: unknown
  created_at: number
}
export type Binding = {
  id?: string
  channel_id: string
  template_id: string
  enabled: boolean
  channel_name?: string
  template_name?: string
}
export type TargetGroup = {
  id: string
  name: string
  bindings: Binding[]
  created_at: number
}
export type Source = {
  id: string
  name: string
  token_prefix: string
  allowed_cidrs: string[]
  custom_sensitive_fields: string[]
  match_mode: string
  payload_policy: string
  enabled: boolean
  created_at: number
}
export type Rule = {
  id: string
  source_id: string
  name: string
  priority: number
  condition: ConditionNode
  target_group_id: string
  enabled: boolean
  created_at: number
}
export type ConditionNode = {
  op?: "and" | "or"
  path?: string
  operator?: string
  value?: unknown
  children?: ConditionNode[]
}
export type Schedule = {
  id: string
  name: string
  recurrence: Recurrence
  timezone: string
  payload: unknown
  target_group_id: string
  enabled: boolean
  next_run_at?: number
  last_run_at?: number
  created_at: number
}
export type Recurrence = {
  kind: string
  interval?: number
  time?: string
  weekdays?: number[]
  day?: number
  month?: number
  cron?: string
  run_at?: string
  start_at?: string
  end_at?: string
}
export type EventItem = {
  id: string
  trigger_type: string
  method: string
  content_type: string
  payload_policy: string
  matched_rules: number
  created_at: number
}
export type Delivery = {
  id: string
  event_id: string
  status: string
  attempts: number
  run_after: number
  last_error?: string
  channel_name: string
  template_name: string
  created_at: number
}
