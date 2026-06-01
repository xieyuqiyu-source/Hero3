/* 统一 API 请求封装 */

import { toast } from '@/components/ui'
import { useAccountStore } from '@/store/accountStore'

const BASE_URL = import.meta.env.VITE_API_BASE ?? '/api/v1'

export class ApiError extends Error {
  status: number
  body: unknown

  constructor(status: number, body: unknown) {
    super(`API Error ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

/** 后端错误消息中文映射 */
const ERROR_MESSAGES: Record<string, string> = {
  'insufficient resources': '资源不足',
  'building not found': '建筑不存在',
  'building is already upgrading': '该建筑正在升级中',
  'building is at max level': '已达最高等级',
  'player not found': '玩家不存在',
  'account not found': '账号不存在',
  'account already exists': '账号已存在',
  'invalid username or password': '用户名或密码错误',
  'invalid json body': '请求数据格式错误',
  'unit not found': '兵种不存在',
  'invalid recruit amount': '征兵数量无效',
  'recruit queue is full': '征兵队列已满',
  'invalid general for faction': '将领不属于该阵营',
  'npc city not found': 'NPC城池不存在',
  'no units selected for dispatch': '未选择出征兵力',
  'insufficient army for dispatch': '兵力不足，无法出征',
  'insufficient gold': '金币不足',
  'insufficient city gold': '城金不足',
  'invalid gold amount': '数量无效',
  'exchange is on cooldown': '兑换冷却中，请稍后再试',
}

/** 从错误响应体中提取可读消息 */
function extractMessage(status: number, body: unknown): string {
  if (body && typeof body === 'object' && 'error' in body) {
    const raw = (body as { error: string }).error
    return ERROR_MESSAGES[raw] ?? raw
  }
  if (status === 401) return '未授权，请重新登录'
  if (status === 403) return '无权限执行此操作'
  if (status === 404) return '请求的资源不存在'
  if (status >= 500) return '服务器异常，请稍后重试'
  return `请求失败 (${status})`
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const url = `${BASE_URL}${path}`

  // 自动注入 JWT token（如果已登录）
  const token = localStorage.getItem('hero3_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> ?? {}),
  }
  if (token && !headers['Authorization']) {
    headers['Authorization'] = `Bearer ${token}`
  }

  let res: Response
  try {
    res = await fetch(url, {
      ...options,
      headers,
    })
  } catch {
    toast.error('网络连接失败，请检查网络')
    throw new ApiError(0, null)
  }

  if (!res.ok) {
    const body = await res.json().catch(() => null)
    const message = extractMessage(res.status, body)
    toast.error(message)

    // 账号不存在或未授权时清除本地登录状态
    if (body && typeof body === 'object' && 'error' in body && (body as { error: string }).error === 'account not found') {
      useAccountStore.getState().logout()
    }
    if (res.status === 401) {
      localStorage.removeItem('hero3_token')
      useAccountStore.getState().logout()
    }

    throw new ApiError(res.status, body)
  }

  return res.json() as Promise<T>
}

export const api = {
  get<T>(path: string) {
    return request<T>(path)
  },
  post<T>(path: string, body: unknown) {
    return request<T>(path, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },
  delete<T>(path: string) {
    return request<T>(path, {
      method: 'DELETE',
    })
  },
}
