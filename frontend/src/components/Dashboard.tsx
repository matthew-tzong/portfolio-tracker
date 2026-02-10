import { useEffect, useState } from 'react'
import { apiRequest } from '../lib/api'
import { supabase } from '../lib/supabase'

// Main signed-in dashboard for the single-user app, shows autheticated user
export function Dashboard() {
  const [user, setUser] = useState<{ email?: string } | null>(null)
  const [pingResult, setPingResult] = useState<string>('')
  const [loading, setLoading] = useState(false)

  // On mount, fetch the current authenticated user so we can show their email.
  useEffect(() => {
    supabase.auth.getUser().then(({ data: { user } }) => {
      setUser(user)
    })
  }, [])

  // Calls protected endpoint to test validation of the Supabase JWT
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

  return (
    <div className="max-w-3xl mx-auto py-12 px-5">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-semibold text-gray-900">My portfolio</h1>
        <button
          onClick={handleSignOut}
          className="py-2 px-4 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2"
        >
          Sign Out
        </button>
      </div>

      <p className="text-gray-600 mb-6">Signed in as {user?.email ?? '...'}</p>

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

      <div className="p-5 bg-blue-50 rounded-lg border border-blue-100">
        <h2 className="text-lg font-medium text-gray-900 mb-2">Next steps</h2>
        <ul className="list-disc list-inside text-gray-700 space-y-1">
          <li>Link your Plaid accounts (Slice 2)</li>
          <li>Link your Snaptrade connection (Slice 2)</li>
          <li>View your accounts and net worth (Slice 3)</li>
        </ul>
      </div>
    </div>
  )
}
