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

/** 从错误响应体中提取可读消息 */
function extractMessage(status: number, body: unknown): string {
  if (body && typeof body === 'object' && 'error' in body) {
    return (body as { error: string }).error
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

  let res: Response
  try {
    res = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    })
  } catch {
    toast.error('网络连接失败，请检查网络')
    throw new ApiError(0, null)
  }

  if (!res.ok) {
    const body = await res.json().catch(() => null)
    const message = extractMessage(res.status, body)
    toast.error(message)

    // 账号不存在时清除本地登录状态
    if (body && typeof body === 'object' && 'error' in body && (body as { error: string }).error === 'account not found') {
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
