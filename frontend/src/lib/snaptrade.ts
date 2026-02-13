import { apiRequest } from './api'

// Timeout for the visibility listener.
const RETURN_TO_TAB_TIMEOUT_MS = 5 * 60 * 1000 // 5 minutes

// Opens the Snaptrade Connect portal in a new tab and syncs connections.
export async function openSnaptradeConnect(onConnected?: () => void): Promise<void> {
  const { redirectUri } = await apiRequest<{ redirectUri: string }>('/api/snaptrade/connect-url', {
    method: 'POST',
  })
  window.open(redirectUri, '_blank', 'noopener,noreferrer')

  let cleanedUp = false
  let timeoutId: number | null = null

  // Cleans up the focus/visibility listener and timeout.
  const cleanup = () => {
    if (cleanedUp) return
    cleanedUp = true
    document.removeEventListener('visibilitychange', onVisibilityChange)
    if (timeoutId !== null) clearTimeout(timeoutId)
  }

  // Syncs connections and calls the onConnected callback.
  const onReturn = () => {
    cleanup()
    syncSnaptradeConnections()
      .then(() => onConnected?.())
      .catch(() => onConnected?.()) // Still run callback so UI can update
  }

  // Handles visibility change events.
  const onVisibilityChange = () => {
    if (document.visibilityState === 'visible') onReturn()
  }

  // Adds the visibility listener.
  document.addEventListener('visibilitychange', onVisibilityChange)
  timeoutId = window.setTimeout(cleanup, RETURN_TO_TAB_TIMEOUT_MS)
}

// Syncs Snaptrade connections from the API into the database.
export async function syncSnaptradeConnections(): Promise<void> {
  await apiRequest('/api/snaptrade/sync-connections', { method: 'POST' })
}
