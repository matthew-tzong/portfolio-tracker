import { useEffect, useState } from 'react'
import { apiRequest } from '../lib/api'
import { Link } from 'react-router-dom'

// Types for the Plaid item.
interface PlaidItem {
  itemId: string
  institutionName?: string
  status: string
  lastUpdated: string
}

// Type for the Snaptrade connection.
interface SnaptradeConnection {
  id: string
  brokerage: string
  status: string
  lastSynced?: string
}

// Type for the links response.
interface LinksResponse {
  plaidItems: PlaidItem[]
  snaptradeConnections: SnaptradeConnection[]
}

// Returns the link management page.
export function LinkManagement() {
  const [data, setData] = useState<LinksResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionMessage, setActionMessage] = useState<string | null>(null)

  // Loads the links.
  const load = async () => {
    setLoading(true)
    setError(null)
    setActionMessage(null)
    try {
      const res = await apiRequest<LinksResponse>('/api/links')
      setData(res)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load links')
    } finally {
      setLoading(false)
    }
  }

  // Syncs Snaptrade connections from the API into the database.
  const syncSnaptradeConnections = async () => {
    try {
      await apiRequest('/api/snaptrade/sync-connections', { method: 'POST' })
    } catch (err: unknown) {
      throw new Error(err instanceof Error ? err.message : 'Failed to sync Snaptrade connections')
    }
  }

  useEffect(() => {
    void load()
  }, [])

  // Removes a Plaid item.
  const removePlaidItem = async (itemId: string) => {
    setActionMessage(null)
    setError(null)
    try {
      await apiRequest('/api/plaid/remove-item', {
        method: 'POST',
        body: JSON.stringify({ itemId }),
      })
      setActionMessage('Plaid item removed.')
      await load()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to remove Plaid item')
    }
  }

  // Removes a Snaptrade connection.
  const removeSnaptradeConnection = async (connectionId: string) => {
    setActionMessage(null)
    setError(null)
    try {
      await apiRequest('/api/snaptrade/remove-connection', {
        method: 'POST',
        body: JSON.stringify({ connectionId }),
      })
      setActionMessage('Snaptrade connection removed.')
      await syncSnaptradeConnections()
      await load()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to remove Snaptrade connection')
    }
  }

  // Returns the link management page.
  return (
    <div className="max-w-4xl mx-auto py-12 px-5">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Connections</h1>
        <div className="flex items-center gap-2">
          <Link
            to="/dashboard"
            className="py-2 px-3 text-xs font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-offset-2"
          >
            Dashboard
          </Link>
          <button
            type="button"
            onClick={() => load()}
            className="py-2 px-3 text-xs font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-offset-2"
          >
            Refresh
          </button>
        </div>
      </div>
      <p className="text-sm text-gray-600 mb-4">
        Manage your Plaid bank connections and Snaptrade brokerage connections. This app is
        single-user: these are your connections only.
      </p>

      {loading && (
        <div className="mt-4 text-gray-600 text-sm">
          <p>Loading connections...</p>
        </div>
      )}

      {error && (
        <div className="mt-4 text-sm text-red-600">
          <p>{error}</p>
        </div>
      )}

      {actionMessage && (
        <div className="mt-4 text-sm text-green-700">
          <p>{actionMessage}</p>
        </div>
      )}

      {!loading && data && (
        <div className="mt-6 space-y-6">
          <section>
            <h2 className="text-lg font-medium text-gray-900 mb-2">Plaid items</h2>
            {data.plaidItems.length === 0 ? (
              <p className="text-sm text-gray-600">
                No Plaid connections yet. Use the dashboard to link an account.
              </p>
            ) : (
              <div className="overflow-hidden rounded-md border border-gray-200 bg-white">
                <table className="min-w-full divide-y divide-gray-200 text-sm">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-2 text-left font-medium text-gray-700">Institution</th>
                      <th className="px-4 py-2 text-left font-medium text-gray-700">Status</th>
                      <th className="px-4 py-2 text-left font-medium text-gray-700">
                        Last updated
                      </th>
                      <th className="px-4 py-2 text-right font-medium text-gray-700">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {data.plaidItems.map((item) => (
                      <tr key={item.itemId}>
                        <td className="px-4 py-2">
                          <div className="font-medium text-gray-900">
                            {item.institutionName || 'Plaid item'}
                          </div>
                          <div className="text-xs text-gray-500 truncate">
                            Item ID: {item.itemId}
                          </div>
                        </td>
                        <td className="px-4 py-2">
                          <span
                            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                              item.status === 'OK'
                                ? 'bg-green-50 text-green-700'
                                : 'bg-red-50 text-red-700'
                            }`}
                          >
                            {item.status}
                          </span>
                        </td>
                        <td className="px-4 py-2 text-sm text-gray-600">
                          {new Date(item.lastUpdated).toLocaleString()}
                        </td>
                        <td className="px-4 py-2 text-right">
                          <button
                            type="button"
                            onClick={() => removePlaidItem(item.itemId)}
                            className="inline-flex items-center rounded-md border border-gray-300 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2"
                          >
                            Remove
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section>
            <h2 className="text-lg font-medium text-gray-900 mb-2">Snaptrade connections</h2>
            {data.snaptradeConnections.length === 0 ? (
              <p className="text-sm text-gray-600">
                No Snaptrade connections yet. Use the dashboard to open Snaptrade Connect; this page
                will refresh connections automatically.
              </p>
            ) : (
              <div className="overflow-hidden rounded-md border border-gray-200 bg-white">
                <table className="min-w-full divide-y divide-gray-200 text-sm">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="px-4 py-2 text-left font-medium text-gray-700">Brokerage</th>
                      <th className="px-4 py-2 text-left font-medium text-gray-700">Status</th>
                      <th className="px-4 py-2 text-left font-medium text-gray-700">Last synced</th>
                      <th className="px-4 py-2 text-right font-medium text-gray-700">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {data.snaptradeConnections.map((conn) => (
                      <tr key={conn.id}>
                        <td className="px-4 py-2">
                          <div className="font-medium text-gray-900">{conn.brokerage}</div>
                          <div className="text-xs text-gray-500 truncate">
                            Connection ID: {conn.id}
                          </div>
                        </td>
                        <td className="px-4 py-2">
                          <span
                            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                              conn.status === 'OK'
                                ? 'bg-green-50 text-green-700'
                                : 'bg-red-50 text-red-700'
                            }`}
                          >
                            {conn.status}
                          </span>
                        </td>
                        <td className="px-4 py-2 text-sm text-gray-600">
                          {conn.lastSynced ? new Date(conn.lastSynced).toLocaleString() : '—'}
                        </td>
                        <td className="px-4 py-2 text-right">
                          <button
                            type="button"
                            onClick={() => removeSnaptradeConnection(conn.id)}
                            className="inline-flex items-center rounded-md border border-gray-300 bg-white px-3 py-1.5 text-xs font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2"
                          >
                            Remove
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </div>
      )}
    </div>
  )
}
