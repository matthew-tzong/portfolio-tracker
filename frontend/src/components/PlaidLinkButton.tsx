import { useState } from 'react'
import { apiRequest } from '../lib/api'

// Adds a callback to handle the Plaid link.
interface PlaidLinkButtonProps {
  onLinked?: () => void
  products?: string[]
}

// Button that opens Plaid Link using the browser script.
export function PlaidLinkButton({
  onLinked,
  products = ['transactions', 'investments'],
}: PlaidLinkButtonProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Handles the click event to open the Plaid Link modal.
  const handleClick = async () => {
    setLoading(true)
    setError(null)

    try {
      if (!window.Plaid) {
        throw new Error('Plaid Link script not loaded')
      }

      // Fetches the link token from the Go backend to add Plaid Link to the page.
      const productsQuery = products.length > 0 ? `?products=${products.join(',')}` : ''
      const { linkToken } = await apiRequest<{ linkToken: string }>(
        `/api/plaid/link-token${productsQuery}`,
      )

      const handler = window.Plaid.create({
        token: linkToken,
        onSuccess: async (publicToken, metadata) => {
          try {
            await apiRequest('/api/plaid/exchange-token', {
              method: 'POST',
              body: JSON.stringify({
                publicToken,
                institutionName: metadata.institution?.name,
                institutionId: metadata.institution?.institution_id,
              }),
            })
            if (onLinked) {
              onLinked()
            }
          } catch (err) {
            console.error('Failed to exchange Plaid public token', err)
            setError('Failed to save Plaid connection')
          }
        },
        onExit: (err) => {
          if (err) {
            console.error('Plaid Link exited with error', err)
          }
        },
      })

      handler.open()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to start Plaid Link')
    } finally {
      setLoading(false)
    }
  }

  // Returns the Plaid link button.
  return (
    <div>
      <button
        type="button"
        onClick={handleClick}
        disabled={loading}
        className="bg-blue-600 text-white px-8 py-3 rounded-full text-sm font-bold hover:bg-blue-500 transition-all shadow-lg active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? (
          <div className="flex items-center gap-2">
            <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            <span>Connecting...</span>
          </div>
        ) : (
          'Connect Bank via Plaid'
        )}
      </button>
      {error && (
        <p className="mt-4 text-xs text-red-400 font-medium bg-red-500/10 border border-red-500/20 p-3 rounded-xl">
          {error}
        </p>
      )}
    </div>
  )
}
