import { useCallback, useEffect, useMemo, useState } from 'react'
import { apiRequest } from '../lib/api'
import { CategoryPieChart } from './CategoryPieChart'

// Category types.
interface Category {
  id: number
  name: string
  expense: boolean
}

// Transaction types.
interface Transaction {
  id: number
  date: string
  amountCents: number
  name: string
  merchantName?: string
  categoryId?: number
  categoryName?: string
  pending: boolean
}

// Transactions response type.
interface TransactionsResponse {
  transactions: Transaction[]
}

// Monthly summary of transactions
interface TransactionsSummaryResponse {
  incomeCents: number
  expensesCents: number
  investedCents: number
}

// Categories response type.
interface CategoriesResponse {
  categories: Category[]
}

// Returns the current month in YYYY-MM.
function currentMonth(): string {
  const day = new Date()
  const year = day.getFullYear()
  const month = String(day.getMonth() + 1).padStart(2, '0')
  return `${year}-${month}`
}

export function ExpenseTracker() {
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [month, setMonth] = useState(currentMonth())
  const [categoryId, setCategoryId] = useState<string>('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<TransactionsSummaryResponse | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Loads the categories from the backend.
  const loadCategories = useCallback(async () => {
    try {
      const res = await apiRequest<CategoriesResponse>('/api/categories')
      setCategories(res.categories ?? [])
    } catch {
      setCategories([])
    }
  }, [])

  // Loads the categories on mount.
  useEffect(() => {
    void loadCategories()
  }, [loadCategories])

  // Loads the transactions from the backend.
  const loadTransactions = useCallback(async () => {
    setLoading(true)
    setError(null)
    // Creates the transaction query string.
    try {
      const params = new URLSearchParams()
      if (month) {
        params.set('month', month)
      }
      if (categoryId) {
        params.set('category', categoryId)
      }
      if (search.trim()) {
        params.set('search', search.trim())
      }
      const query = params.toString()
      const res = await apiRequest<TransactionsResponse>(
        `/api/transactions${query ? `?${query}` : ''}`,
      )
      setTransactions(res.transactions ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load transactions')
      setTransactions([])
    } finally {
      setLoading(false)
    }
  }, [month, categoryId, search])

  // Loads the transactions on mount and when filters change.
  useEffect(() => {
    void loadTransactions()
  }, [loadTransactions])

  // Loads the monthly summary (income, expenses, invested) when month changes.
  const loadSummary = useCallback(async () => {
    if (!month) {
      setSummary(null)
      setSummaryLoading(false)
      return
    }
    setSummaryLoading(true)
    setError(null)
    try {
      const res = await apiRequest<TransactionsSummaryResponse>(
        `/api/transactions/summary?month=${encodeURIComponent(month)}`,
      )
      setSummary(res)
    } catch {
      setSummary(null)
    } finally {
      setSummaryLoading(false)
    }
  }, [month])

  // Loads the monthly summary on mount and when month changes.
  useEffect(() => {
    void loadSummary()
  }, [loadSummary])

  // Formats the currency.
  const formatCurrency = (cents: number) =>
    new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      maximumFractionDigits: 2,
    }).format(cents / 100)

  // Builds month options for the last 24 months.
  const monthOptions: string[] = []
  const day = new Date()
  for (let i = 0; i < 24; i++) {
    const year = day.getFullYear()
    const month = String(day.getMonth() + 1).padStart(2, '0')
    monthOptions.push(`${year}-${month}`)
    day.setMonth(day.getMonth() - 1)
  }

  // Aggregates expenses by category
  const categoryBreakdown = useMemo(() => {
    if (!transactions.length) {
      return []
    }
    const totalsByCategory: Record<string, number> = {}
    transactions.forEach((transaction) => {
      if (transaction.amountCents >= 0) {
        return
      }
      const categoryName = transaction.categoryName || 'Uncategorized'
      totalsByCategory[categoryName] =
        (totalsByCategory[categoryName] ?? 0) + Math.abs(transaction.amountCents)
    })
    return Object.entries(totalsByCategory).map(([name, valueCents]) => ({
      name,
      value: valueCents / 100,
    }))
  }, [transactions])

  // Returns the expense tracker page.
  return (
    <div className="max-w-4xl mx-auto py-8 px-5">
      <h1 className="text-2xl font-semibold text-gray-900 mb-6">Expense tracker</h1>

      <div className="mb-6 p-5 bg-white rounded-lg border border-gray-200 shadow-sm">
        <div className="flex flex-wrap items-end gap-4 mb-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Month</label>
            <select
              value={month}
              onChange={(e) => setMonth(e.target.value)}
              className="block w-40 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
            >
              {monthOptions.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Category</label>
            <select
              value={categoryId}
              onChange={(e) => setCategoryId(e.target.value)}
              className="block w-48 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
            >
              <option value="">All</option>
              {categories.map((c) => (
                <option key={c.id} value={String(c.id)}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Search</label>
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Name or merchant"
              className="block w-56 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
            />
          </div>
        </div>

        {error && <p className="text-sm text-red-600 mb-4">{error}</p>}

        {/* Monthly summary: income, after expenses, invested, after investments */}
        {month && (
          <div className="mb-6 p-4 bg-gray-50 rounded-lg border border-gray-200">
            <h3 className="text-sm font-semibold text-gray-800 mb-3">Month summary</h3>
            {summaryLoading ? (
              <p className="text-sm text-gray-500">Loading summary…</p>
            ) : summary ? (
              <dl className="space-y-1.5 text-sm">
                <div className="flex justify-between">
                  <dt className="text-gray-600">Income (before expenses)</dt>
                  <dd className="font-medium text-green-700">
                    {formatCurrency(summary.incomeCents)}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-600">Expenses</dt>
                  <dd className="font-medium text-red-700">
                    −{formatCurrency(summary.expensesCents)}
                  </dd>
                </div>
                <div className="flex justify-between border-t border-gray-200 pt-2 mt-2">
                  <dt className="text-gray-800 font-medium">After expenses (saved)</dt>
                  <dd className="font-medium text-gray-900">
                    {formatCurrency(summary.incomeCents - summary.expensesCents)}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-gray-600">Invested this month</dt>
                  <dd className="font-medium text-blue-700">
                    {formatCurrency(summary.investedCents)}
                  </dd>
                </div>
              </dl>
            ) : (
              <p className="text-sm text-gray-500">No summary for this month.</p>
            )}
          </div>
        )}

        {month && (
          <div className="mb-6">
            <CategoryPieChart title={`Expenses by category (${month})`} data={categoryBreakdown} />
          </div>
        )}

        {loading && <p className="text-sm text-gray-600">Loading transactions…</p>}
        {!loading && (
          <div className="overflow-hidden rounded-md border border-gray-200">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-2 text-left font-medium text-gray-700">Date</th>
                  <th className="px-4 py-2 text-left font-medium text-gray-700">Name</th>
                  <th className="px-4 py-2 text-left font-medium text-gray-700">Category</th>
                  <th className="px-4 py-2 text-right font-medium text-gray-700">Amount</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 bg-white">
                {transactions.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-8 text-center text-gray-500">
                      No transactions for this month. Link a bank or credit card in Manage
                      connections; transactions sync automatically each night.
                    </td>
                  </tr>
                ) : (
                  transactions.map((tx) => (
                    <tr key={tx.id}>
                      <td className="px-4 py-2 text-gray-700">{tx.date}</td>
                      <td className="px-4 py-2">
                        <div className="font-medium text-gray-900">{tx.name}</div>
                        {tx.merchantName && (
                          <div className="text-xs text-gray-500">{tx.merchantName}</div>
                        )}
                      </td>
                      <td className="px-4 py-2 text-gray-700">
                        {tx.categoryName ?? '—'}
                        {tx.pending && (
                          <span className="ml-1 text-xs text-amber-600">(pending)</span>
                        )}
                      </td>
                      <td className="px-4 py-2 text-right font-medium">
                        <span className={tx.amountCents >= 0 ? 'text-red-700' : 'text-green-700'}>
                          {formatCurrency(tx.amountCents)}
                        </span>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
