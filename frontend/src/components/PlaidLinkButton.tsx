import { useState } from 'react'
import { apiRequest } from '../lib/api'

// Adds a callback to handle the Plaid link.
interface PlaidLinkButtonProps {
  onLinked?: () => void
}

// Button that opens Plaid Link using the browser script.
export function PlaidLinkButton({ onLinked }: PlaidLinkButtonProps) {
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
      const { linkToken } = await apiRequest<{ linkToken: string }>('/api/plaid/link-token')

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
        className="py-2.5 px-4 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? 'Adding Plaid Item...' : 'Add Plaid Item'}
      </button>
      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
    </div>
  )
}
