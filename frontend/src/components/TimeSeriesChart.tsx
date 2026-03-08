import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

// Time series data point.
interface TimeSeriesPoint {
  date: string
  value: number
}

// Time series chart
interface TimeSeriesChartProps {
  title: string
  data: TimeSeriesPoint[]
  height?: number
  isMonthly?: boolean
}

// Formats a number as a currency string.
function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format(value)
}

// Formats a date string.
function formatDateLabel(dateStr: string, isMonthly?: boolean): string {
  // Get date from YYYY-MM-DD string.
  const [year, month, day] = dateStr.split('-').map(Number)
  const date = new Date(year, month - 1, day)

  // If monthly, format as "Month Year".
  if (isMonthly) {
    return date.toLocaleDateString('en-US', {
      month: 'long',
      year: 'numeric',
    })
  }

  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

// Returns the time series chart.
export function TimeSeriesChart({ title, data, height = 300, isMonthly }: TimeSeriesChartProps) {
  if (!data.length) {
    return (
      <div
        className="flex items-center justify-center p-8 bg-zinc-900/50 rounded-2xl border border-border"
        style={{ height }}
      >
        <p className="text-sm text-zinc-500 font-medium italic">
          No performance data available yet.
        </p>
      </div>
    )
  }

  const sortedData = [...data].sort((a, b) => a.date.localeCompare(b.date))

  return (
    <div className="w-full">
      {title && <h3 className="text-md font-bold text-white mb-4">{title}</h3>}
      <div style={{ width: '100%', minWidth: 0, minHeight: height }}>
        <ResponsiveContainer width="100%" height={height}>
          <LineChart data={sortedData} margin={{ top: 10, right: 10, bottom: 0, left: -20 }}>
            <defs>
              <linearGradient id="lineGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#22c55e" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#27272a" vertical={false} />
            <XAxis
              dataKey="date"
              tickFormatter={(label) => formatDateLabel(String(label), isMonthly)}
              tickLine={false}
              axisLine={false}
              tick={{ fontSize: 10, fill: '#71717a', fontWeight: 'bold' }}
              dy={10}
            />
            <YAxis
              tickFormatter={formatCurrency}
              tickLine={false}
              axisLine={false}
              tick={{ fontSize: 10, fill: '#71717a', fontWeight: 'bold' }}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: '#18181b',
                border: '1px solid #27272a',
                borderRadius: '12px',
                fontSize: '12px',
                color: '#fff',
              }}
              itemStyle={{ color: '#22c55e', fontWeight: 'bold' }}
              formatter={(value: unknown) =>
                typeof value === 'number' ? formatCurrency(value) : String(value)
              }
              labelFormatter={(label) => formatDateLabel(String(label), isMonthly)}
            />
            <Line
              type="monotone"
              dataKey="value"
              stroke="#22c55e"
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6, fill: '#22c55e', stroke: '#09090b', strokeWidth: 2 }}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
