import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { apiRequest } from '../lib/api'
import { supabase } from '../lib/supabase'

// Account type.
interface Account {
  provider: string
  plaidItemId?: string
  accountId: string
  name: string
  mask?: string
  type: string
  subtype?: string
  balanceCents: number
  isLiability: boolean
}

// Accounts response type.
interface AccountsResponse {
  accounts: Account[]
  netWorthCents: number
  cashCents: number
  investmentsCents: number
  liabilitiesCents: number
}

// Main signed-in dashboard for the single-user app, shows autheticated user
export function Dashboard() {
  const [user, setUser] = useState<{ email?: string } | null>(null)
  const [pingResult, setPingResult] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [accountsData, setAccountsData] = useState<AccountsResponse | null>(null)
  const [accountsLoading, setAccountsLoading] = useState(false)
  const [accountsError, setAccountsError] = useState<string | null>(null)

  // On mount, fetch the current authenticated user so we can show their email.
  useEffect(() => {
    supabase.auth.getUser().then(({ data: { user } }) => {
      setUser(user)
    })
  }, [])

  // Loads current accounts and net worth from the backend.
  const loadAccounts = async () => {
    setAccountsLoading(true)
    setAccountsError(null)
    // Fetches the accounts from the backend.
    try {
      const res = await apiRequest<AccountsResponse>('/api/accounts')
      setAccountsData(res)
    } catch (err: unknown) {
      setAccountsError(err instanceof Error ? err.message : 'Failed to load accounts')
    } finally {
      setAccountsLoading(false)
    }
  }

  // Loads the accounts on mount.
  useEffect(() => {
    void loadAccounts()
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

  // Formats the currency.
  const formatCurrency = (cents: number) =>
    new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      maximumFractionDigits: 2,
    }).format(cents / 100)

  // Returns the dashboard.
  return (
    <div className="max-w-3xl mx-auto py-12 px-5">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-semibold text-gray-900">My portfolio</h1>
        <div className="flex items-center gap-3">
          <Link
            to="/expenses"
            className="py-2 px-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-offset-2"
          >
            Expense tracker
          </Link>
          <Link
            to="/budget"
            className="py-2 px-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-offset-2"
          >
            Budget tracker
          </Link>
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

      <div className="mb-6 p-5 bg-white rounded-lg border border-gray-200 shadow-sm">
        <h2 className="text-lg font-medium text-gray-900 mb-2">Net worth</h2>
        {accountsLoading && (
          <p className="text-sm text-gray-600">Loading accounts and balances...</p>
        )}
        {accountsError && (
          <p className="text-sm text-red-600">Failed to load accounts: {accountsError}</p>
        )}
        {!accountsLoading && !accountsError && accountsData && (
          <>
            <p className="text-3xl font-semibold text-gray-900 mb-3">
              {formatCurrency(accountsData.netWorthCents)}
            </p>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm mb-4">
              <div className="bg-green-50 border border-green-100 rounded-md p-3">
                <p className="text-xs uppercase tracking-wide text-green-800">Cash</p>
                <p className="text-base font-medium text-green-900">
                  {formatCurrency(accountsData.cashCents)}
                </p>
              </div>
              <div className="bg-blue-50 border border-blue-100 rounded-md p-3">
                <p className="text-xs uppercase tracking-wide text-blue-800">Investments</p>
                <p className="text-base font-medium text-blue-900">
                  {formatCurrency(accountsData.investmentsCents)}
                </p>
              </div>
              <div className="bg-red-50 border border-red-100 rounded-md p-3">
                <p className="text-xs uppercase tracking-wide text-red-800">Liabilities</p>
                <p className="text-base font-medium text-red-900">
                  {formatCurrency(-accountsData.liabilitiesCents)}
                </p>
              </div>
            </div>

            {(accountsData.accounts?.length ?? 0) > 0 && (
              <div className="mt-4">
                <h3 className="text-sm font-medium text-gray-800 mb-2">Accounts</h3>
                <div className="overflow-hidden rounded-md border border-gray-200 bg-white">
                  <table className="min-w-full divide-y divide-gray-200 text-sm">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-4 py-2 text-left font-medium text-gray-700">Name</th>
                        <th className="px-4 py-2 text-left font-medium text-gray-700">Type</th>
                        <th className="px-4 py-2 text-right font-medium text-gray-700">Balance</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {(accountsData.accounts ?? []).map((acct) => (
                        <tr key={`${acct.provider}-${acct.accountId}`}>
                          <td className="px-4 py-2">
                            <div className="font-medium text-gray-900">{acct.name}</div>
                            <div className="text-xs text-gray-500">
                              {acct.mask ? `•••• ${acct.mask}` : acct.accountId}
                            </div>
                          </td>
                          <td className="px-4 py-2 text-gray-700">
                            {acct.subtype ? `${acct.type} · ${acct.subtype}` : acct.type}
                          </td>
                          <td className="px-4 py-2 text-right">
                            <span
                              className={
                                acct.isLiability
                                  ? 'text-red-700 font-medium'
                                  : 'text-gray-900 font-medium'
                              }
                            >
                              {formatCurrency(acct.balanceCents)}
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </>
        )}
      </div>

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
    </div>
  )
}
