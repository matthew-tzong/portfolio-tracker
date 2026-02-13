import { useState } from 'react'
import { openSnaptradeConnect } from '../lib/snaptrade'

// Adds callback to handle the Snaptrade connection.
interface SnaptradeConnectSectionProps {
  onConnected?: () => void
}

// Connects to Snaptrade and handles the connection.
export function SnaptradeConnectSection({ onConnected }: SnaptradeConnectSectionProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Opens the Snaptrade Connect URL in a new tab.
  const openConnect = async () => {
    setLoading(true)
    setError(null)

    try {
      await openSnaptradeConnect(onConnected)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to open Snaptrade Connect')
    } finally {
      setLoading(false)
    }
  }

  // Returns the Snaptrade connect section.
  return (
    <div className="p-5 bg-gray-50 rounded-lg border border-gray-100">
      <h2 className="text-lg font-medium text-gray-900 mb-2">Snaptrade connection</h2>
      <p className="text-sm text-gray-700 mb-3">
        Connect your brokerage (e.g., Fidelity) via Snaptrade to sync portfolio data.
      </p>
      <button
        type="button"
        onClick={openConnect}
        disabled={loading}
        className="py-2.5 px-4 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? 'Opening Snaptrade…' : 'Open Snaptrade Connect'}
      </button>
      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}
    </div>
  )
}
