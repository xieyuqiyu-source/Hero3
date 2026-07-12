/** 验证统一客户端的 JWT 注入、错误解析和 401 处理。 */
import { describe, expect, it, vi } from 'vitest'
import { ApiError, createApiClient } from './client'

describe('API 客户端', () => {
  it('自动为受保护请求附加 Bearer JWT', async () => {
    const fetcher = vi.fn(async (_url: string | URL | Request, init?: RequestInit) => {
      expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer secret-token')
      return new Response(JSON.stringify({ ok: true }), { status: 200, headers: { 'Content-Type': 'application/json' } })
    }) as typeof fetch
    const client = createApiClient({ baseUrl: '/api/v1', getToken: () => 'secret-token', onUnauthorized: vi.fn(), fetcher })
    await expect(client.get('/accounts/a1')).resolves.toEqual({ ok: true })
  })

  it('401 时调用统一失效处理且不泄露 JWT', async () => {
    const onUnauthorized = vi.fn()
    const fetcher = vi.fn(async () => new Response(JSON.stringify({ error: 'expired' }), { status: 401 })) as typeof fetch
    const client = createApiClient({ baseUrl: '/api/v1', getToken: () => 'private-jwt', onUnauthorized, fetcher })
    const error = await client.get('/accounts/a1').catch((caught: unknown) => caught)
    expect(error).toBeInstanceOf(ApiError)
    expect((error as Error).message).not.toContain('private-jwt')
    expect(onUnauthorized).toHaveBeenCalledOnce()
  })

  it('网络失败转为可读错误', async () => {
    const fetcher = vi.fn(async () => { throw new Error('socket') }) as typeof fetch
    const client = createApiClient({ baseUrl: '/api/v1', getToken: () => null, onUnauthorized: vi.fn(), fetcher })
    await expect(client.get('/game/bootstrap')).rejects.toMatchObject({ status: 0, message: '网络连接失败，请检查后端服务或 API 地址' })
  })
})
