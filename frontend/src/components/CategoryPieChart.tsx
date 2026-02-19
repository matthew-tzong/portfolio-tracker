import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'

// Category data point.
interface CategoryDatum {
  name: string
  value: number
}

// Category pie chart.
interface CategoryPieChartProps {
  title: string
  data: CategoryDatum[]
  height?: number
}

const COLORS = ['#2563EB', '#16A34A', '#F97316', '#EC4899', '#14B8A6', '#A855F7', '#FACC15']

// Formats a number as a currency string.
function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format(value)
}

// Returns the category pie chart.
export function CategoryPieChart({ title, data, height = 260 }: CategoryPieChartProps) {
  if (!data.length) {
    return <p className="text-sm text-gray-500">No category data available.</p>
  }

  // Filters out zero data points.
  const nonZeroData = data.filter((d) => d.value > 0)
  if (!nonZeroData.length) {
    return <p className="text-sm text-gray-500">No non-zero category values to show.</p>
  }

  return (
    <div className="w-full">
      <h3 className="text-md font-medium text-gray-800 mb-2">{title}</h3>
      <div style={{ width: '100%', height }}>
        <ResponsiveContainer>
          <PieChart>
            <Pie
              data={nonZeroData}
              dataKey="value"
              nameKey="name"
              cx="50%"
              cy="50%"
              outerRadius="80%"
              paddingAngle={2}
            >
              {nonZeroData.map((entry, index) => (
                <Cell
                  key={entry.name}
                  fill={COLORS[index % COLORS.length]}
                  stroke="#FFFFFF"
                  strokeWidth={1}
                />
              ))}
            </Pie>
            <Tooltip
              formatter={(value: unknown, name: unknown) =>
                typeof value === 'number'
                  ? [formatCurrency(value), String(name)]
                  : [String(value), String(name)]
              }
            />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
