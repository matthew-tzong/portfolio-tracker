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

  const [historyDaily, setHistoryDaily] = useState<HoldingDataPoint[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)

  const [accountMonthly, setAccountMonthly] = useState<SnapshotDataPoint[]>([])
  const [accountMonthlyLoading, setAccountMonthlyLoading] = useState(false)

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

  // Returns the portfolio page.
  return (
    <div className="max-w-7xl mx-auto py-8 px-5">
      <h1 className="text-2xl font-semibold text-gray-900 mb-6">Portfolio</h1>

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
    </div>
  )
}
