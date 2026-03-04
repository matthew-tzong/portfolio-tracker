import { supabase } from './supabase'

const API_URL = import.meta.env.VITE_API_URL || ''

/*
 - Authenticated request to the Go backend (single-user app).
 - Attaches the current Supabase session access token as a Bearer JWT.
 - If no active session, throws an error and redirects to the auth page.
 */
export async function apiRequest<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const {
    data: { session },
  } = await supabase.auth.getSession()

  if (!session) {
    throw new Error('Not authenticated')
  }

  const response = await fetch(`${API_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${session.access_token}`,
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const error = await response.text().catch(() => 'Request failed')
    throw new Error(`API error: ${error}`)
  }

  if (response.status === 204) {
    return {} as T
  }

  const text = await response.text()
  return text ? JSON.parse(text) : ({} as T)
}
