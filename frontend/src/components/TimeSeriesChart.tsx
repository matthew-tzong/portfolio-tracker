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
function formatDateLabel(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

// Returns the time series chart.
export function TimeSeriesChart({ title, data, height = 260 }: TimeSeriesChartProps) {
  if (!data.length) {
    return <p className="text-sm text-gray-500">No data available yet.</p>
  }

  const sortedData = [...data].sort((a, b) => a.date.localeCompare(b.date))

  return (
    <div className="w-full">
      <h3 className="text-md font-medium text-gray-800 mb-2">{title}</h3>
      <div style={{ width: '100%', height }}>
        <ResponsiveContainer>
          <LineChart data={sortedData} margin={{ top: 10, right: 20, bottom: 10, left: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
            <XAxis
              dataKey="date"
              tickFormatter={formatDateLabel}
              tickLine={false}
              axisLine={{ stroke: '#E5E7EB' }}
              tick={{ fontSize: 11, fill: '#6B7280' }}
            />
            <YAxis
              tickFormatter={formatCurrency}
              tickLine={false}
              axisLine={{ stroke: '#E5E7EB' }}
              tick={{ fontSize: 11, fill: '#6B7280' }}
            />
            <Tooltip
              formatter={(value: unknown) =>
                typeof value === 'number' ? formatCurrency(value) : String(value)
              }
              labelFormatter={(label) => formatDateLabel(String(label))}
            />
            <Line
              type="monotone"
              dataKey="value"
              stroke="#2563EB"
              strokeWidth={2}
              dot={{ r: 2 }}
              activeDot={{ r: 4 }}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
