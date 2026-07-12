/** web-wlsg 独立 API 客户端，负责 JWT、错误和 401 会话失效处理。 */
import type { ErrorResponse } from './types'

export interface ApiClientOptions {
  baseUrl: string
  getToken: () => string | null
  onUnauthorized: () => void
  fetcher?: typeof fetch
}

export class ApiError extends Error {
  readonly status: number
  readonly body: unknown

  /** 创建包含 HTTP 状态和原始响应体的 API 错误。 */
  constructor(status: number, body: unknown, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

const errorMessages: Record<string, string> = {
  'invalid username or password': '用户名或密码错误',
  'account not found': '账号不存在或已失效',
  'account service unavailable': '账号服务暂不可用，请稍后重试',
  'authentication is not configured': '服务器登录功能尚未配置',
  'insufficient resources': '资源不足，无法执行操作',
  'building is already upgrading': '该建筑正在建造中',
  'building is at max level': '该建筑已达最高等级',
  'building not found': '建筑不存在',
  'building status blocks this action': '当前建筑状态无法升级',
  'unit not found': '当前阵营不存在该兵种',
  'invalid recruit amount': '征兵数量必须是 1 至 100000 的整数',
  'recruit queue is full': '征兵队列已满',
  'insufficient city gold': '城金不足',
}

/** 从后端响应中提取适合展示的中文错误。 */
function resolveErrorMessage(status: number, body: unknown): string {
  const response = body && typeof body === 'object' ? body as ErrorResponse : null
  const raw = response?.error ?? response?.message
  if (raw) return errorMessages[raw] ?? raw
  if (status === 401) return '登录已失效，请重新登录'
  if (status === 403) return '当前账号无权访问该内容'
  if (status === 404) return '请求的内容不存在'
  if (status >= 500) return '服务器异常，请稍后重试'
  return `请求失败（${status}）`
}

/** 创建统一 GET、POST 请求客户端。 */
export function createApiClient(options: ApiClientOptions) {
  const fetcher = options.fetcher ?? fetch

  /** 发送请求并统一解析 JSON 与错误。 */
  async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers)
    headers.set('Accept', 'application/json')
    if (init.body !== undefined) headers.set('Content-Type', 'application/json')
    const token = options.getToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)

    let response: Response
    try {
      response = await fetcher(`${options.baseUrl}${path}`, { ...init, headers })
    } catch {
      throw new ApiError(0, null, '网络连接失败，请检查后端服务或 API 地址')
    }

    const body = await response.json().catch(() => null) as unknown
    if (!response.ok) {
      if (response.status === 401 && token) options.onUnauthorized()
      throw new ApiError(response.status, body, resolveErrorMessage(response.status, body))
    }
    return body as T
  }

  return {
    /** 发送 GET 请求。 */
    get<T>(path: string, init: RequestInit = {}) { return request<T>(path, init) },
    /** 发送 JSON POST 请求。 */
    post<T>(path: string, body: unknown) {
      return request<T>(path, { method: 'POST', body: JSON.stringify(body) })
    },
  }
}

export type ApiClient = ReturnType<typeof createApiClient>
