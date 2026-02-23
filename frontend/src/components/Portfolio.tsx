import { useCallback, useEffect, useState } from 'react'
import { apiRequest } from '../lib/api'
import { TimeSeriesChart } from './TimeSeriesChart'

// Holding in JSON format.
interface Holding {
  accountId: string
  accountName: string
  symbol: string
  quantity: number
  valueCents: number
}

// Holdings response in JSON format.
interface HoldingsResponse {
  holdings: Holding[]
}

// Snapshot data point for charts in JSON format.
interface SnapshotDataPoint {
  date: string
  portfolioValueCents: number
}

// Snapshots response in JSON format.
interface SnapshotsResponse {
  daily: SnapshotDataPoint[]
  monthly: SnapshotDataPoint[]
}

// Holding data point for charts in JSON format.
interface HoldingDataPoint {
  date: string
  accountId: string
  accountName?: string
  symbol: string
  quantity?: number
  valueCents: number
}

// Holdings history response in JSON format.
interface HoldingsHistoryResponse {
  daily: HoldingDataPoint[]
}

// Yearly portfolio summary by account.
interface YearlyPortfolioAccount {
  accountId: string
  portfolioValueCents: number
}

interface YearlyPortfolioSummaryResponse {
  year: number
  byAccount: YearlyPortfolioAccount[]
}

const START_MONTH = '2026-02'
const START_YEAR = 2026

// Selected item for the portfolio.
type SelectedItem =
  | null
  | { type: 'total' }
  | { type: 'account'; accountId: string; accountName: string }
  | { type: 'holding'; accountId: string; accountName: string; symbol: string }

// Formats the currency.
function formatCurrency(cents: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(cents / 100)
}

export function Portfolio() {
  const [holdings, setHoldings] = useState<Holding[]>([])
  const [holdingsLoading, setHoldingsLoading] = useState(false)
  const [holdingsError, setHoldingsError] = useState<string | null>(null)

  const [snapshots, setSnapshots] = useState<SnapshotsResponse | null>(null)
  const [snapshotsLoading, setSnapshotsLoading] = useState(false)
  const [snapshotsError, setSnapshotsError] = useState<string | null>(null)

  const [selected, setSelected] = useState<SelectedItem>(null)

  const currentDate = new Date()
  const lastMonth = new Date(currentDate.getFullYear(), currentDate.getMonth() - 1, 1)
  const [exportMonth, setExportMonth] = useState(() => {
    const lm = `${lastMonth.getFullYear()}-${String(lastMonth.getMonth() + 1).padStart(2, '0')}`
    return lm < START_MONTH ? START_MONTH : lm
  })

  const [historyDaily, setHistoryDaily] = useState<HoldingDataPoint[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)

  const [accountMonthly, setAccountMonthly] = useState<SnapshotDataPoint[]>([])
  const [accountMonthlyLoading, setAccountMonthlyLoading] = useState(false)

  const [yearlySummaryYear, setYearlySummaryYear] = useState<string>('')
  const [yearlySummary, setYearlySummary] = useState<YearlyPortfolioSummaryResponse | null>(null)
  const [yearlySummaryLoading, setYearlySummaryLoading] = useState(false)
  const [yearlySummaryError, setYearlySummaryError] = useState<string | null>(null)

  // Loads the holdings.
  const loadHoldings = useCallback(async () => {
    setHoldingsLoading(true)
    setHoldingsError(null)
    try {
      const data = await apiRequest<HoldingsResponse>('/api/portfolio/holdings')
      setHoldings(data.holdings)
    } catch (err) {
      setHoldingsError(err instanceof Error ? err.message : 'Failed to load holdings')
    } finally {
      setHoldingsLoading(false)
    }
  }, [])

  // Loads the snapshots.
  const loadSnapshots = useCallback(async () => {
    setSnapshotsLoading(true)
    setSnapshotsError(null)
    try {
      const data = await apiRequest<SnapshotsResponse>('/api/portfolio/snapshots')
      setSnapshots(data)
    } catch (err) {
      setSnapshotsError(err instanceof Error ? err.message : 'Failed to load snapshots')
    } finally {
      setSnapshotsLoading(false)
    }
  }, [])

  // Loads the holdings and snapshots on mount.
  useEffect(() => {
    loadHoldings()
    loadSnapshots()
  }, [loadHoldings, loadSnapshots])

  // Loads yearly portfolio summary by account.
  const loadYearlySummary = useCallback(async () => {
    if (!yearlySummaryYear) {
      setYearlySummary(null)
      setYearlySummaryError(null)
      setYearlySummaryLoading(false)
      return
    }
    setYearlySummaryLoading(true)
    setYearlySummaryError(null)
    try {
      const res = await apiRequest<YearlyPortfolioSummaryResponse>(
        `/api/portfolio/summary/yearly?year=${encodeURIComponent(yearlySummaryYear)}`,
      )
      setYearlySummary(res)
    } catch (err) {
      setYearlySummaryError(err instanceof Error ? err.message : 'Failed to load yearly summary')
      setYearlySummary(null)
    } finally {
      setYearlySummaryLoading(false)
    }
  }, [yearlySummaryYear])

  useEffect(() => {
    void loadYearlySummary()
  }, [loadYearlySummary])

  // Loads the holdings and snapshots when the selected item changes.
  useEffect(() => {
    if (!selected || selected.type === 'total') {
      setHistoryDaily([])
      setAccountMonthly([])
      return
    }
    // Loads the holdings history for an account.
    if (selected.type === 'account') {
      setHistoryLoading(true)
      apiRequest<HoldingsHistoryResponse>(
        `/api/portfolio/holdings/history?accountId=${encodeURIComponent(selected.accountId)}`,
      )
        .then((data) => setHistoryDaily(data.daily))
        .catch(() => setHistoryDaily([]))
        .finally(() => setHistoryLoading(false))
      setAccountMonthlyLoading(true)
      apiRequest<SnapshotsResponse>(
        `/api/portfolio/snapshots?accountId=${encodeURIComponent(selected.accountId)}`,
      )
        .then((data) => setAccountMonthly(data.monthly ?? []))
        .catch(() => setAccountMonthly([]))
        .finally(() => setAccountMonthlyLoading(false))
      return
    }
    // Loads the holdings history for a holding.
    if (selected.type === 'holding') {
      setAccountMonthly([])
      setHistoryLoading(true)
      apiRequest<HoldingsHistoryResponse>(
        `/api/portfolio/holdings/history?symbol=${encodeURIComponent(selected.symbol)}`,
      )
        .then((data) => setHistoryDaily(data.daily))
        .catch(() => setHistoryDaily([]))
        .finally(() => setHistoryLoading(false))
      return
    }
  }, [selected])

  // Calculates the total portfolio value
  const holdingsList = holdings ?? []
  const totalPortfolioValue = holdingsList.reduce((sum, holding) => sum + holding.valueCents, 0)

  // Calculates the holdings by account.
  const holdingsByAccount = holdingsList.reduce(
    (accountMap, holding) => {
      if (!accountMap[holding.accountId]) {
        accountMap[holding.accountId] = {
          accountName: holding.accountName,
          holdings: [],
          totalValue: 0,
        }
      }
      accountMap[holding.accountId].holdings.push(holding)
      accountMap[holding.accountId].totalValue += holding.valueCents
      return accountMap
    },
    {} as Record<string, { accountName: string; holdings: Holding[]; totalValue: number }>,
  )

  // Gets the snapshots for the total portfolio.
  const totalDailySeries = snapshots?.daily ?? []
  const totalMonthlySeries = (snapshots?.monthly ?? []).slice(-12)
  const totalDailyChartData = totalDailySeries.map((snapshot) => ({
    date: snapshot.date,
    value: snapshot.portfolioValueCents / 100,
  }))

  const totalMonthlyChartData = totalMonthlySeries.map((snapshot) => ({
    date: snapshot.date,
    value: snapshot.portfolioValueCents / 100,
  }))

  // Gets the daily snapshots for the selected account.
  const accountDailySeries = (() => {
    if (!selected || selected.type !== 'account' || !historyDaily.length) return []
    const byDate: Record<string, number> = {}
    historyDaily.forEach((holding) => {
      byDate[holding.date] = (byDate[holding.date] ?? 0) + holding.valueCents
    })
    return Object.entries(byDate)
      .map(([date, valueCents]) => ({ date, valueCents }))
      .sort((a, b) => a.date.localeCompare(b.date))
  })()

  // Gets the daily snapshots for the selected holding.
  const holdingDailySeries = (() => {
    if (!selected || selected.type !== 'holding' || !historyDaily.length) {
      return []
    }
    const byDate: Record<string, number> = {}
    historyDaily.forEach((holding) => {
      byDate[holding.date] = (byDate[holding.date] ?? 0) + holding.valueCents
    })
    return Object.entries(byDate)
      .map(([date, valueCents]) => ({ date, valueCents }))
      .sort((a, b) => a.date.localeCompare(b.date))
  })()

  // Returns the label for the selected item.
  const selectedLabel = (() => {
    if (selected === null) {
      return null
    }
    if (selected.type === 'total') {
      return 'Total portfolio'
    }
    if (selected.type === 'account') {
      return selected.accountName
    }
    return `${selected.symbol} (${selected.accountName})`
  })()

  // Exports portfolio snapshots for a month as CSV.
  const handleExportSnapshots = async () => {
    try {
      const { supabase } = await import('../lib/supabase')
      const {
        data: { session },
      } = await supabase.auth.getSession()
      const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
      const response = await fetch(
        `${API_URL}/api/export/portfolio/snapshots?month=${exportMonth}`,
        {
          headers: {
            Authorization: `Bearer ${session?.access_token}`,
          },
        },
      )

      if (!response.ok) {
        throw new Error('Failed to export snapshots')
      }

      // Downloads the CSV file.
      const blob = await response.blob()
      const objectURL = window.URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = objectURL
      anchor.download = `portfolio-snapshots-${exportMonth}.csv`
      document.body.appendChild(anchor)
      anchor.click()
      window.URL.revokeObjectURL(objectURL)
      document.body.removeChild(anchor)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to export snapshots')
    }
  }

  // Exports portfolio holdings for a month as CSV.
  const handleExportHoldings = async () => {
    try {
      const { supabase } = await import('../lib/supabase')
      const {
        data: { session },
      } = await supabase.auth.getSession()
      const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
      const response = await fetch(
        `${API_URL}/api/export/portfolio/holdings?month=${exportMonth}`,
        {
          headers: {
            Authorization: `Bearer ${session?.access_token}`,
          },
        },
      )

      if (!response.ok) {
        throw new Error('Failed to export holdings')
      }

      // Downloads the CSV file.
      const blob = await response.blob()
      const objectURL = window.URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = objectURL
      anchor.download = `portfolio-holdings-${exportMonth}.csv`
      document.body.appendChild(anchor)
      anchor.click()
      window.URL.revokeObjectURL(objectURL)
      document.body.removeChild(anchor)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to export holdings')
    }
  }

  // Generates month options for export dropdown (last 13 months including current).
  const generateMonthOptions = () => {
    const monthOptions: string[] = []
    const now = new Date()
    const startMonthDate = new Date(Date.UTC(START_YEAR, 1, 1))
    const cursor = new Date(Date.UTC(now.getFullYear(), now.getMonth(), 1))
    while (cursor >= startMonthDate) {
      const y = cursor.getUTCFullYear()
      const m = String(cursor.getUTCMonth() + 1).padStart(2, '0')
      monthOptions.push(`${y}-${m}`)
      cursor.setUTCMonth(cursor.getUTCMonth() - 1)
    }
    return monthOptions
  }

  // Returns the portfolio page.
  return (
    <div className="max-w-7xl mx-auto py-8 px-5">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Portfolio</h1>
        <div className="flex items-center gap-3">
          <select
            value={exportMonth}
            onChange={(e) => setExportMonth(e.target.value)}
            className="block rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
          >
            {generateMonthOptions().map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={handleExportSnapshots}
            className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            Export Snapshots
          </button>
          <button
            type="button"
            onClick={handleExportHoldings}
            className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            Export Holdings
          </button>
        </div>
      </div>

      {/* Total portfolio value for the day (data updates via nightly cron) */}
      <div className="mb-6 p-5 bg-white rounded-lg border border-gray-200 shadow-sm">
        <h2 className="text-lg font-medium text-gray-900 mb-4">Portfolio value today</h2>
        {holdingsError && <p className="text-sm text-red-600 mb-4">Error: {holdingsError}</p>}
        {holdingsLoading && !holdingsList.length && (
          <p className="text-sm text-gray-600">Loading...</p>
        )}
        {!holdingsLoading && holdingsList.length === 0 && !holdingsError && (
          <p className="text-sm text-gray-600">
            No holdings. Connect a Snaptrade account to see positions.
          </p>
        )}
        {holdingsList.length > 0 && (
          <p className="text-3xl font-semibold text-gray-900">
            {formatCurrency(totalPortfolioValue)}
          </p>
        )}
      </div>

      {/* By account and by holding — selectable */}
      <div className="mb-6 p-5 bg-white rounded-lg border border-gray-200 shadow-sm">
        <h2 className="text-lg font-medium text-gray-900 mb-4">By account and position</h2>
        {holdingsList.length === 0 && (
          <p className="text-sm text-gray-500">Connect an account to see breakdown.</p>
        )}
        {holdingsList.length > 0 && (
          <div className="space-y-4">
            {/* Clickable total row */}
            <button
              type="button"
              onClick={() => setSelected(selected?.type === 'total' ? null : { type: 'total' })}
              className={`w-full text-left p-4 rounded-lg border-2 transition-colors ${
                selected?.type === 'total'
                  ? 'border-blue-600 bg-blue-50'
                  : 'border-gray-200 hover:border-gray-300 bg-gray-50/50'
              }`}
            >
              <span className="font-medium text-gray-900">Total portfolio</span>
              <span className="ml-2 text-gray-600">{formatCurrency(totalPortfolioValue)}</span>
              <span className="ml-2 text-xs text-gray-500">
                Click to see 30-day and 12-month performance
              </span>
            </button>

            {Object.entries(holdingsByAccount).map(([accountId, accountData]) => (
              <div key={accountId} className="border border-gray-200 rounded-lg overflow-hidden">
                {/* Clickable account row */}
                <button
                  type="button"
                  onClick={() =>
                    setSelected(
                      selected?.type === 'account' && selected.accountId === accountId
                        ? null
                        : { type: 'account', accountId, accountName: accountData.accountName },
                    )
                  }
                  className={`w-full text-left px-4 py-3 flex items-center justify-between ${
                    selected?.type === 'account' && selected.accountId === accountId
                      ? 'bg-blue-50 border-l-4 border-blue-600'
                      : 'hover:bg-gray-50'
                  }`}
                >
                  <span className="font-medium text-gray-900">{accountData.accountName}</span>
                  <span className="text-gray-700">{formatCurrency(accountData.totalValue)}</span>
                </button>
                {/* Holdings within account */}
                <div className="border-t border-gray-100">
                  {accountData.holdings.map((h, idx) => (
                    <button
                      type="button"
                      key={`${h.accountId}-${h.symbol}-${idx}`}
                      onClick={() =>
                        setSelected(
                          selected?.type === 'holding' &&
                            selected.accountId === h.accountId &&
                            selected.symbol === h.symbol
                            ? null
                            : {
                                type: 'holding',
                                accountId: h.accountId,
                                accountName: h.accountName,
                                symbol: h.symbol,
                              },
                        )
                      }
                      className={`w-full text-left px-6 py-2 flex items-center justify-between text-sm ${
                        selected?.type === 'holding' &&
                        selected.accountId === h.accountId &&
                        selected.symbol === h.symbol
                          ? 'bg-blue-50/80'
                          : 'hover:bg-gray-50'
                      }`}
                    >
                      <span className="text-gray-700">{h.symbol}</span>
                      <span className="text-gray-600">
                        {h.quantity.toFixed(4)} · {formatCurrency(h.valueCents)}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Performance for selected item: Last 30 days + Past 12 months */}
      {selected !== null && selectedLabel && (
        <div className="mb-6 p-5 bg-white rounded-lg border border-gray-200 shadow-sm">
          <h2 className="text-lg font-medium text-gray-900 mb-1">Performance: {selectedLabel}</h2>
          {selected.type === 'total' && snapshotsError && (
            <p className="text-sm text-red-600 mb-2">Snapshot error: {snapshotsError}</p>
          )}
          <p className="text-sm text-gray-500 mb-4">
            Last 30 days (daily) and past 12 months (monthly). Data appears after the nightly cron
            runs.
          </p>

          {/* Last 30 days (daily) */}
          <div className="mt-4 space-y-4">
            {selected.type === 'total' && (
              <>
                <TimeSeriesChart
                  title="Last 30 days (daily portfolio value)"
                  data={totalDailyChartData}
                />
                {totalDailySeries.length === 0 && !snapshotsLoading && (
                  <p className="text-sm text-gray-500">No daily snapshot data yet.</p>
                )}
              </>
            )}
            {selected.type === 'account' && (
              <>
                {historyLoading && <p className="text-sm text-gray-500">Loading...</p>}
                {!historyLoading && accountDailySeries.length === 0 && (
                  <p className="text-sm text-gray-500">No daily history for this account yet.</p>
                )}
                {!historyLoading && accountDailySeries.length > 0 && (
                  <TimeSeriesChart
                    title="Last 30 days (daily value for this account)"
                    data={accountDailySeries.map((s) => ({
                      date: s.date,
                      value: s.valueCents / 100,
                    }))}
                  />
                )}
              </>
            )}
            {selected.type === 'holding' && (
              <>
                {historyLoading && <p className="text-sm text-gray-500">Loading...</p>}
                {!historyLoading && holdingDailySeries.length === 0 && (
                  <p className="text-sm text-gray-500">No daily history for this holding yet.</p>
                )}
                {!historyLoading && holdingDailySeries.length > 0 && (
                  <TimeSeriesChart
                    title="Last 30 days (daily value for this holding)"
                    data={holdingDailySeries.map((s) => ({
                      date: s.date,
                      value: s.valueCents / 100,
                    }))}
                  />
                )}
              </>
            )}
          </div>

          {/* Past 12 months (monthly) */}
          <div className="mt-6">
            {selected.type === 'total' && (
              <>
                <TimeSeriesChart
                  title="Past 12 months (monthly total portfolio value)"
                  data={totalMonthlyChartData}
                />
                {totalMonthlySeries.length === 0 && !snapshotsLoading && (
                  <p className="text-sm text-gray-500 mt-2">No monthly snapshot data yet.</p>
                )}
              </>
            )}
            {selected.type === 'account' && (
              <>
                {accountMonthlyLoading && (
                  <p className="text-sm text-gray-500">Loading monthly...</p>
                )}
                {!accountMonthlyLoading && accountMonthly.length === 0 && (
                  <p className="text-sm text-gray-500">No monthly history for this account yet.</p>
                )}
                {!accountMonthlyLoading && accountMonthly.length > 0 && (
                  <TimeSeriesChart
                    title="Past 12 months (monthly value for this account)"
                    data={accountMonthly.slice(-12).map((s) => ({
                      date: s.date,
                      value: s.portfolioValueCents / 100,
                    }))}
                  />
                )}
              </>
            )}
          </div>

          {selected.type === 'holding' && (
            <p className="text-sm text-gray-500 mt-4">
              Monthly performance per holding is not stored; only total portfolio and per-account
              monthly history are available.
            </p>
          )}
        </div>
      )}

      {/* Yearly portfolio summary by account (end-of-year value) */}
      <div className="mt-8 p-5 bg-white rounded-lg border border-gray-200 shadow-sm">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Yearly portfolio summary</h2>
        <p className="text-sm text-gray-600 mb-4">
          End-of-year portfolio value per account (from retained yearly summaries). Data appears
          after retention has run for that year.
        </p>
        <div className="flex items-end gap-4 mb-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Year</label>
            <select
              value={yearlySummaryYear}
              onChange={(e) => setYearlySummaryYear(e.target.value)}
              className="block w-32 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
            >
              <option value="">Select…</option>
              {Array.from(
                { length: Math.max(0, new Date().getFullYear() - START_YEAR + 1) },
                (_, i) => new Date().getFullYear() - i,
              ).map((y) => (
                <option key={y} value={y}>
                  {y}
                </option>
              ))}
            </select>
          </div>
        </div>
        {yearlySummaryError && <p className="text-sm text-red-600 mb-4">{yearlySummaryError}</p>}
        {!yearlySummaryYear ? (
          <p className="text-sm text-gray-500">Select a year to view yearly totals.</p>
        ) : yearlySummaryLoading ? (
          <p className="text-sm text-gray-500">Loading…</p>
        ) : yearlySummary && yearlySummary.byAccount.length > 0 ? (
          <div className="overflow-hidden rounded-md border border-gray-200">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-2 text-left font-medium text-gray-700">Account</th>
                  <th className="px-4 py-2 text-right font-medium text-gray-700">
                    End-of-year value
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 bg-white">
                {yearlySummary.byAccount.map((row) => {
                  const displayName = holdingsByAccount[row.accountId]?.accountName ?? row.accountId
                  return (
                    <tr key={row.accountId}>
                      <td className="px-4 py-2 font-medium text-gray-900">{displayName}</td>
                      <td className="px-4 py-2 text-right text-gray-700">
                        {formatCurrency(row.portfolioValueCents)}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
            <div className="px-4 py-2 bg-gray-50 border-t border-gray-200 text-sm font-medium text-gray-900">
              Total:{' '}
              {formatCurrency(
                yearlySummary.byAccount.reduce((s, r) => s + r.portfolioValueCents, 0),
              )}
            </div>
          </div>
        ) : yearlySummary && yearlySummary.byAccount.length === 0 ? (
          <p className="text-sm text-gray-500">
            No yearly data for {yearlySummaryYear}. Summaries are created when retention runs (e.g.
            after year-end).
          </p>
        ) : null}
      </div>
    </div>
  )
}
