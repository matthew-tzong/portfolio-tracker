import { useEffect, useState } from 'react'
import { apiRequest } from '../lib/api'
import { TimeSeriesChart } from './TimeSeriesChart'

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

// Net worth snapshot response type.
interface NetWorthSnapshot {
  month: string
  netWorthCents: number
  cashCents: number
  investmentsCents: number
  liabilitiesCents: number
}

// Main signed-in dashboard for the single-user app, shows authenticated user
export function Dashboard() {
  const [accountsData, setAccountsData] = useState<AccountsResponse | null>(null)
  const [accountsLoading, setAccountsLoading] = useState(false)
  const [accountsError, setAccountsError] = useState<string | null>(null)

  const [netWorthSnapshots, setNetWorthSnapshots] = useState<NetWorthSnapshot[]>([])
  const [netWorthLoading, setNetWorthLoading] = useState(false)
  const [netWorthError, setNetWorthError] = useState<string | null>(null)

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

  // Loads monthly net worth snapshots from the backend.
  const loadNetWorthSnapshots = async () => {
    setNetWorthLoading(true)
    setNetWorthError(null)
    try {
      const res = await apiRequest<{ monthly: NetWorthSnapshot[] }>('/api/net-worth/snapshots')
      setNetWorthSnapshots(res.monthly ?? [])
    } catch (err: unknown) {
      setNetWorthError(err instanceof Error ? err.message : 'Failed to load net worth history')
      setNetWorthSnapshots([])
    } finally {
      setNetWorthLoading(false)
    }
  }

  // Loads the accounts on mount.
  useEffect(() => {
    void loadAccounts()
    void loadNetWorthSnapshots()
  }, [])

  // Formats the currency.
  const formatCurrency = (cents: number) =>
    new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      maximumFractionDigits: 2,
    }).format(cents / 100)

  // Returns the dashboard page.
  return (
    <div className="max-w-4xl mx-auto py-12 px-6">
      <h1 className="text-3xl font-bold text-white tracking-tight mb-10">Dashboard</h1>

      {/* Net Worth Summary */}
      <div className="mb-8 bg-card border border-border rounded-4xl p-10 shadow-2xl relative overflow-hidden group">
        <div className="absolute top-0 right-0 p-8">
          <div className="bg-primary/10 border border-primary/20 rounded-full px-3 py-1 flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
            <span className="text-[10px] font-bold text-primary uppercase tracking-tighter">
              Today's Total
            </span>
          </div>
        </div>

        <h2 className="text-zinc-500 text-xs font-bold uppercase tracking-widest mb-2 ml-1">
          Total Net Worth
        </h2>
        {accountsLoading ? (
          <div className="h-12 w-48 bg-zinc-800 animate-pulse rounded-lg mb-8" />
        ) : (
          <p className="text-5xl font-bold text-white mb-10 tracking-tighter">
            {accountsData ? formatCurrency(accountsData.netWorthCents) : '$0.00'}
          </p>
        )}

        {accountsError && (
          <p className="text-sm text-red-400 mb-6 bg-red-500/10 border border-red-500/20 p-4 rounded-2xl">
            Error loading accounts: {accountsError}
          </p>
        )}

        {!accountsLoading && !accountsError && accountsData && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-zinc-900 border border-border p-6 rounded-3xl group hover:border-green-500/30 transition-all">
              <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest mb-1">
                Cash Assets
              </p>
              <p className="text-xl font-bold text-white">
                {formatCurrency(accountsData.cashCents)}
              </p>
            </div>
            <div className="bg-zinc-900 border border-border p-6 rounded-3xl group hover:border-blue-500/30 transition-all">
              <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest mb-1">
                Investments
              </p>
              <p className="text-xl font-bold text-white">
                {formatCurrency(accountsData.investmentsCents)}
              </p>
            </div>
            <div className="bg-zinc-900 border border-border p-6 rounded-3xl group hover:border-red-500/30 transition-all">
              <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest mb-1">
                Liabilities
              </p>
              <p className="text-xl font-bold text-red-400">
                {formatCurrency(-accountsData.liabilitiesCents)}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Net Worth Over Time Chart */}
      <div className="mb-8 bg-card border border-border rounded-4xl p-10 shadow-2xl">
        <h2 className="text-xl font-bold text-white mb-8">Net Worth Over Time</h2>
        {netWorthLoading ? (
          <div className="h-[300px] bg-zinc-800 animate-pulse rounded-3xl" />
        ) : netWorthError ? (
          <p className="text-sm text-red-400 bg-red-500/10 border border-red-500/20 p-4 rounded-2xl">
            {netWorthError}
          </p>
        ) : (
          <div className="h-[300px]">
            <TimeSeriesChart
              title=""
              data={netWorthSnapshots.map((s) => ({
                date: s.month,
                value: s.netWorthCents / 100,
              }))}
              height={300}
              isMonthly={true}
            />
          </div>
        )}
      </div>
    </div>
  )
}
