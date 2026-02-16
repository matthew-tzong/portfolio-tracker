import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { apiRequest } from '../lib/api'
import { supabase } from '../lib/supabase'

// Category types.
interface Category {
  id: number
  name: string
  expense: boolean
}

// Budget API response type.
interface BudgetResponse {
  month: string
  allocations: Record<string, number>
  spent: Record<string, number>
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

export function BudgetTracker() {
  const [user, setUser] = useState<{ email?: string } | null>(null)
  const [categories, setCategories] = useState<Category[]>([])
  const [month, setMonth] = useState(currentMonth())
  const [allocations, setAllocations] = useState<Record<string, number>>({})
  const [spent, setSpent] = useState<Record<string, number>>({})
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)
  const [totalBudget, setTotalBudget] = useState<string>('')

  // Loads the user from Supabase.
  useEffect(() => {
    supabase.auth.getUser().then(({ data: { user } }) => setUser(user))
  }, [])

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

  // Loads the budget for the selected month.
  const loadBudget = useCallback(async () => {
    if (!month) {
      setAllocations({})
      setSpent({})
      return
    }
    setLoading(true)
    setError(null)
    try {
      const res = await apiRequest<BudgetResponse>(`/api/budget?month=${encodeURIComponent(month)}`)
      const nextAllocations = res.allocations ?? {}
      setAllocations(nextAllocations)
      setSpent(res.spent ?? {})

      // Computes the current budget allocations.
      const totalAllocatedCents = Object.values(nextAllocations).reduce(
        (sum, value) => sum + (typeof value === 'number' ? value : 0),
        0,
      )
      setTotalBudget(totalAllocatedCents > 0 ? (totalAllocatedCents / 100).toString() : '')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load budget')
      setAllocations({})
      setSpent({})
    } finally {
      setLoading(false)
    }
  }, [month])

  // Loads the budget when the month changes.
  useEffect(() => {
    void loadBudget()
  }, [loadBudget])

  // Handles sign out.
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

  // Builds month options for the last 24 months.
  const monthOptions: string[] = []
  const day = new Date()
  for (let i = 0; i < 24; i++) {
    const year = day.getFullYear()
    const m = String(day.getMonth() + 1).padStart(2, '0')
    monthOptions.push(`${year}-${m}`)
    day.setMonth(day.getMonth() - 1)
  }

  // Handles budget change for a category (value is in dollars).
  const handleAllocationChange = (categoryId: number, value: string) => {
    const amount = parseFloat(value)
    setAllocations((prev) => {
      const nextAllocations = { ...prev }
      const categoryName = String(categoryId)
      if (Number.isNaN(amount)) {
        nextAllocations[categoryName] = 0
      } else {
        nextAllocations[categoryName] = Math.round(amount * 100)
      }
      return nextAllocations
    })
  }

  // Saves the budget allocations.
  const handleSaveBudget = async () => {
    setSaving(true)
    setError(null)
    setSuccessMessage(null)

    // Normalize allocations so every expense category has an entry (at least 0).
    const normalizedAllocations: Record<string, number> = {}
    categories
      .filter((c) => c.expense)
      .forEach((category) => {
        const key = String(category.id)
        const value = allocations[key]
        normalizedAllocations[key] = typeof value === 'number' && !Number.isNaN(value) ? value : 0
      })

    // Validate that allocations sum to the designated total budget.
    const parsedTotal = parseFloat(totalBudget)
    const totalAllocatedCents = Object.values(normalizedAllocations).reduce(
      (sum, value) => sum + (typeof value === 'number' ? value : 0),
      0,
    )

    if (Number.isNaN(parsedTotal) || parsedTotal <= 0) {
      setError('Enter a total monthly budget greater than 0 before saving.')
      setSaving(false)
      return
    }

    // Computes target budget.
    const targetCents = Math.round(parsedTotal * 100)
    if (totalAllocatedCents !== targetCents) {
      setError(
        `Invalid allocations: per-category totals ${formatCurrency(
          totalAllocatedCents,
        )} must equal your total budget ${formatCurrency(targetCents)}.`,
      )
      setSaving(false)
      return
    }

    try {
      await apiRequest('/api/budget', {
        method: 'PUT',
        body: JSON.stringify({ allocations: normalizedAllocations }),
      })
      setSuccessMessage('Budget saved. It applies to all months until you change it.')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to save budget')
    } finally {
      setSaving(false)
    }
  }

  // Returns the budget tracker UI.
  return (
    <div className="max-w-4xl mx-auto py-12 px-5">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-semibold text-gray-900">Budget tracker</h1>
        <div className="flex items-center gap-3">
          <Link
            to="/dashboard"
            className="py-2 px-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Dashboard
          </Link>
          <Link
            to="/expenses"
            className="py-2 px-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Expense tracker
          </Link>
          <Link
            to="/links"
            className="py-2 px-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
          >
            Manage connections
          </Link>
          <button
            onClick={handleSignOut}
            className="py-2 px-4 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700"
          >
            Sign Out
          </button>
        </div>
      </div>

      <p className="text-gray-600 mb-6">Signed in as {user?.email ?? '...'}</p>

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
            <p className="mt-1 text-xs text-gray-500">
              Budget numbers are global; the month only changes the spent column.
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Total monthly budget
            </label>
            <input
              type="number"
              min={0}
              step="0.01"
              value={totalBudget}
              onChange={(e) => setTotalBudget(e.target.value)}
              placeholder="0.00"
              className="block w-40 rounded-md border border-gray-300 px-3 py-2 text-sm text-right focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
            />
            <p className="mt-1 text-xs text-gray-500">
              Per-category budgets must add up exactly to this total.
            </p>
          </div>
        </div>

        {!loading && (
          <div className="mb-3 text-xs text-gray-600">
            <span className="mr-3">
              Total allocated:{' '}
              {formatCurrency(
                Object.values(allocations).reduce(
                  (sum, value) => sum + (typeof value === 'number' ? value : 0),
                  0,
                ),
              )}
            </span>
            {totalBudget && (
              <span>
                Target: {formatCurrency(Math.round((parseFloat(totalBudget || '0') || 0) * 100))}
              </span>
            )}
          </div>
        )}

        {error && <p className="text-sm text-red-600 mb-3">{error}</p>}
        {successMessage && <p className="text-sm text-green-700 mb-3">{successMessage}</p>}

        {loading ? (
          <p className="text-sm text-gray-600">Loading budget…</p>
        ) : (
          <>
            <div className="overflow-hidden rounded-md border border-gray-200">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-2 text-left font-medium text-gray-700">Category</th>
                    <th className="px-4 py-2 text-right font-medium text-gray-700">
                      Budget (per month)
                    </th>
                    <th className="px-4 py-2 text-right font-medium text-gray-700">
                      Spent ({month})
                    </th>
                    <th className="px-4 py-2 text-right font-medium text-gray-700">Remaining</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 bg-white">
                  {categories.filter((c) => c.expense).length === 0 ? (
                    <tr>
                      <td colSpan={4} className="px-4 py-6 text-center text-gray-500">
                        No expense categories available.
                      </td>
                    </tr>
                  ) : (
                    categories
                      .filter((c) => c.expense)
                      .map((category) => {
                        const key = String(category.id)
                        const allocatedCents = allocations[key] ?? 0
                        const spentCents = spent[key] ?? 0
                        const remainingCents = allocatedCents - spentCents
                        const overBudget = allocatedCents > 0 && spentCents > allocatedCents

                        return (
                          <tr key={category.id}>
                            <td className="px-4 py-2 text-gray-800">{category.name}</td>
                            <td className="px-4 py-2 text-right">
                              <input
                                type="number"
                                min={0}
                                step="0.01"
                                value={allocatedCents > 0 ? (allocatedCents / 100).toString() : ''}
                                onChange={(e) => handleAllocationChange(category.id, e.target.value)}
                                className="w-28 rounded-md border border-gray-300 px-2 py-1 text-right text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
                                placeholder="0.00"
                              />
                            </td>
                            <td className="px-4 py-2 text-right">
                              {spentCents === 0 ? (
                                <span className="text-gray-400">—</span>
                              ) : (
                                <span className={overBudget ? 'text-red-700' : 'text-gray-900'}>
                                  {formatCurrency(spentCents)}
                                </span>
                              )}
                            </td>
                            <td className="px-4 py-2 text-right">
                              {allocatedCents === 0 ? (
                                <span className="text-gray-400">No budget</span>
                              ) : (
                                <span
                                  className={
                                    remainingCents < 0
                                      ? 'text-red-700 font-medium'
                                      : 'text-green-700'
                                  }
                                >
                                  {formatCurrency(remainingCents)}
                                </span>
                              )}
                            </td>
                          </tr>
                        )
                      })
                  )}
                </tbody>
              </table>
            </div>

            {/* Overall totals across all categories for this month */}
            <div className="mt-4 flex justify-between text-sm text-gray-800">
              <div>
                <p className="font-medium">
                  Total budgeted:{' '}
                  {formatCurrency(
                    Object.values(allocations).reduce(
                      (sum, value) => sum + (typeof value === 'number' ? value : 0),
                      0,
                    ),
                  )}
                </p>
                <p>
                  Total spent in {month}:{' '}
                  {formatCurrency(
                    Object.values(spent).reduce(
                      (sum, value) => sum + (typeof value === 'number' ? value : 0),
                      0,
                    ),
                  )}
                </p>
              </div>
              <div className="text-right">
                <p className="font-semibold">
                  Total remaining:{' '}
                  {formatCurrency(
                    Object.values(allocations).reduce(
                      (sum, value) => sum + (typeof value === 'number' ? value : 0),
                      0,
                    ) -
                      Object.values(spent).reduce(
                        (sum, value) => sum + (typeof value === 'number' ? value : 0),
                        0,
                      ),
                  )}
                </p>
              </div>
            </div>
          </>
        )}

        <div className="mt-4 flex justify-end">
          <button
            type="button"
            onClick={handleSaveBudget}
            disabled={saving}
            className="inline-flex items-center px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? 'Saving…' : 'Save budget'}
          </button>
        </div>
      </div>
    </div>
  )
}
