import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { apiRequest } from '../lib/api'
import { supabase } from '../lib/supabase'
import { PlaidLinkButton } from './PlaidLinkButton'
import { SnaptradeConnectSection } from './SnaptradeConnectSection.tsx'

// Main signed-in dashboard for the single-user app, shows autheticated user
export function Dashboard() {
  const [user, setUser] = useState<{ email?: string } | null>(null)
  const [pingResult, setPingResult] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)

  // Handles the Plaid link success message.
  const handlePlaidLinked = () => {
    setSuccessMessage('Bank account linked successfully!')
    setTimeout(() => setSuccessMessage(null), 5000)
  }

  // Handles the Snaptrade connection success message + sync.
  const handleSnaptradeConnected = () => {
    ;(async () => {
      // Sync the Snaptrade connections.
      try {
        await apiRequest('/api/snaptrade/sync-connections', { method: 'POST' })
        setSuccessMessage('Brokerage connection synced successfully!')
      } catch (err: unknown) {
        // If sync fails, then say that the sync may not be up to date.
        setSuccessMessage(
          `Snaptrade Connect opened, but sync may not be up to date: ${err instanceof Error ? err.message : 'Unknown error'}`,
        )
      } finally {
        setTimeout(() => setSuccessMessage(null), 5000)
      }
    })()
  }

  // On mount, fetch the current authenticated user so we can show their email.
  useEffect(() => {
    supabase.auth.getUser().then(({ data: { user } }) => {
      setUser(user)
    })
  }, [])

  // Tests the validation of the Supabase JWT.
  const testProtectedEndpoint = async () => {
    setLoading(true)
    try {
      const result = await apiRequest<{ message: string }>('/api/protected/ping')
      setPingResult(result.message)
    } catch (err: unknown) {
      setPingResult(`Error: ${err instanceof Error ? err.message : 'Request failed'}`)
    } finally {
      setLoading(false)
    }
  }

  // Signs the user out of Supabase and redirects them to `/auth`.
  const handleSignOut = async () => {
    await supabase.auth.signOut()
  }

  // Returns the dashboard.
  return (
    <div className="max-w-3xl mx-auto py-12 px-5">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-semibold text-gray-900">My portfolio</h1>
        <div className="flex items-center gap-3">
          <Link
            to="/links"
            className="py-2 px-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-offset-2"
          >
            Manage connections
          </Link>
          <button
            onClick={handleSignOut}
            className="py-2 px-4 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2"
          >
            Sign Out
          </button>
        </div>
      </div>

      <p className="text-gray-600 mb-6">Signed in as {user?.email ?? '...'}</p>

      {successMessage && (
        <div className="mb-6 p-4 bg-green-50 border border-green-200 rounded-lg">
          <p className="text-sm font-medium text-green-800">{successMessage}</p>
        </div>
      )}

      <div className="mb-6 p-5 bg-gray-100 rounded-lg">
        <h2 className="text-lg font-medium text-gray-900 mb-1">Protected API Test</h2>
        <p className="text-sm text-gray-600 mb-3">Test the Go backend JWT validation:</p>
        <button
          onClick={testProtectedEndpoint}
          disabled={loading}
          className="mt-2 py-2.5 px-4 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? 'Testing...' : 'Test Protected Endpoint'}
        </button>
        {pingResult && (
          <div className="mt-3 p-3 bg-white rounded border border-gray-200 text-sm">
            <strong>Result:</strong> {pingResult}
          </div>
        )}
      </div>

      <div className="grid gap-6 md:grid-cols-2 mt-6">
        <div className="p-5 bg-blue-50 rounded-lg border border-blue-100">
          <h2 className="text-lg font-medium text-gray-900 mb-2">Connections</h2>
          <p className="text-sm text-gray-700 mb-3">
            Start by linking your bank or credit card accounts.
          </p>
          <PlaidLinkButton onLinked={handlePlaidLinked} />
        </div>
        <SnaptradeConnectSection onConnected={handleSnaptradeConnected} />
      </div>
    </div>
  )
}
