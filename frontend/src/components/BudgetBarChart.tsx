import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

// Budget data point.
interface BudgetDataPoint {
  name: string
  budget: number
  spent: number
}

// Budget bar chart.
interface BudgetBarChartProps {
  title: string
  data: BudgetDataPoint[]
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

// Returns the budget bar chart.
export function BudgetBarChart({ title, data, height = 280 }: BudgetBarChartProps) {
  if (!data.length) {
    return <p className="text-sm text-gray-500">No budget data available.</p>
  }

  // Filters out zero data points.
  const nonZeroData = data.filter((dataPoint) => dataPoint.budget > 0 || dataPoint.spent > 0)
  if (!nonZeroData.length) {
    return <p className="text-sm text-gray-500">All categories have zero budget and spend.</p>
  }

  return (
    <div className="w-full">
      <h3 className="text-md font-medium text-gray-800 mb-2">{title}</h3>
      <div style={{ width: '100%', height }}>
        <ResponsiveContainer>
          <BarChart
            data={nonZeroData}
            margin={{ top: 10, right: 20, bottom: 40, left: 0 }}
            barCategoryGap={16}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
            <XAxis
              dataKey="name"
              angle={-35}
              textAnchor="end"
              height={50}
              interval={0}
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
              formatter={(value: unknown, name: unknown) =>
                typeof value === 'number'
                  ? [formatCurrency(value), name === 'budget' ? 'Budget' : 'Spent']
                  : [String(value), String(name)]
              }
            />
            <Legend />
            <Bar dataKey="budget" name="Budget" fill="#93C5FD" />
            <Bar dataKey="spent" name="Spent" fill="#FCA5A5" />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
