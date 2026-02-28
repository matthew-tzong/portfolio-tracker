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
    <div className="p-8 bg-zinc-900 border border-border rounded-4xl group hover:border-violet-500/30 transition-all shadow-xl relative overflow-hidden">
      <div className="absolute -right-4 -top-4 w-24 h-24 bg-violet-500/5 rounded-full blur-2xl group-hover:bg-violet-500/10 transition-all" />
      <div className="relative">
        <h2 className="text-xl font-bold text-white mb-2">Brokerage Sync</h2>
        <p className="text-zinc-500 text-sm font-medium mb-8 leading-relaxed">
          Connect via Snaptrade to sync your portfolios and holdings from any brokerage.
        </p>
        <button
          type="button"
          onClick={openConnect}
          disabled={loading}
          className="bg-violet-600 text-white px-8 py-3 rounded-full text-sm font-bold hover:bg-violet-500 transition-all shadow-lg active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? (
            <div className="flex items-center gap-2">
              <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              <span>Opening...</span>
            </div>
          ) : (
            'Open Snaptrade Connect'
          )}
        </button>
        {error && (
          <p className="mt-4 text-xs text-red-400 font-medium bg-red-500/10 border border-red-500/20 p-3 rounded-xl">
            {error}
          </p>
        )}
      </div>
    </div>
  )
}
