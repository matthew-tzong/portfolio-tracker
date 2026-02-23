import { beforeEach, describe, expect, it, vi } from 'vitest'

// Mock for the fetch function.
const fetchMock = vi.fn()

// Mock for the supabase client.
vi.mock('./supabase', () => {
  const getSession = vi.fn()
  return {
    supabase: {
      auth: {
        getSession,
      },
    },
  }
})

// Stub the fetch function.
vi.stubGlobal('fetch', fetchMock)

// Import the apiRequest function.
import { apiRequest } from './api'

// Test suite for the apiRequest function.
describe('apiRequest', () => {
  beforeEach(() => {
    fetchMock.mockReset()
  })

  // Test that the function throws when there is no active session.
  it('throws when there is no active session', async () => {
    const { supabase } = await import('./supabase')
    ;(supabase.auth.getSession as any).mockResolvedValue({ data: { session: null } })
    await expect(apiRequest('/api/test')).rejects.toThrow('Not authenticated')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  // Test that the function attaches the bearer token and returns parsed JSON.
  it('attaches bearer token and returns parsed JSON', async () => {
    const { supabase } = await import('./supabase')
    ;(supabase.auth.getSession as any).mockResolvedValue({
      data: { session: { access_token: 'test-token' } },
    })
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ok: true }),
    })
    const result = await apiRequest<{ ok: boolean }>('/api/protected')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toContain('/api/protected')
    expect(options?.headers).toMatchObject({
      Authorization: 'Bearer test-token',
      'Content-Type': 'application/json',
    })
    expect(result).toEqual({ ok: true })
  })

  // Test that the function throws a helpful error when the response is not ok.
  it('throws a helpful error when response is not ok', async () => {
    const { supabase } = await import('./supabase')
    ;(supabase.auth.getSession as any).mockResolvedValue({
      data: { session: { access_token: 'test-token' } },
    })
    fetchMock.mockResolvedValue({
      ok: false,
      text: () => Promise.resolve('Something went wrong'),
    })
    await expect(apiRequest('/api/error')).rejects.toThrow('API error: Something went wrong')
  })
})
