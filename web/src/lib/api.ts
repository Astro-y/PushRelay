const API_BASE = import.meta.env.VITE_API_URL ?? ""
let csrfToken = sessionStorage.getItem("pushrelay_csrf") ?? ""

export function setCSRF(value: string) {
  csrfToken = value
  if (value) sessionStorage.setItem("pushrelay_csrf", value)
  else sessionStorage.removeItem("pushrelay_csrf")
}

export class APIError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !headers.has("Content-Type"))
    headers.set("Content-Type", "application/json")
  if (init.method && !["GET", "HEAD"].includes(init.method) && csrfToken)
    headers.set("X-CSRF-Token", csrfToken)
  const response = await fetch(`${API_BASE}/api/v1${path}`, {
    ...init,
    headers,
    credentials: "include",
  })
  if (response.status === 204) return undefined as T
  const data = await response.json().catch(() => ({}))
  if (!response.ok)
    throw new APIError(
      response.status,
      data.code ?? "request_failed",
      data.message ??
        (document.documentElement.lang === "en" ? "Request failed" : "请求失败")
    )
  return data as T
}

export const jsonBody = (value: unknown) => JSON.stringify(value)

export const apiEndpoint = (path: string) =>
  new URL(`${API_BASE}/api/v1${path}`, window.location.origin).toString()
