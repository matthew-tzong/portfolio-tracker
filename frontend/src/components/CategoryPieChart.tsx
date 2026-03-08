import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip, Legend } from 'recharts'

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
      {title && <h3 className="text-md font-bold text-white mb-4">{title}</h3>}
      <div style={{ width: '100%', minWidth: 0, minHeight: height }}>
        <ResponsiveContainer width="100%" height={height}>
          <PieChart>
            <Pie
              data={nonZeroData}
              dataKey="value"
              nameKey="name"
              cx="50%"
              cy="50%"
              outerRadius="80%"
              paddingAngle={0}
              label={({ name, percent }) => `${name} (${((percent || 0) * 100).toFixed(0)}%)`}
              labelLine={{ stroke: '#52525b', strokeWidth: 1 }}
            >
              {nonZeroData.map((entry, index) => (
                <Cell
                  key={entry.name}
                  fill={COLORS[index % COLORS.length]}
                  stroke="none"
                  strokeWidth={0}
                />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: '#18181b',
                border: '1px solid #27272a',
                borderRadius: '12px',
                fontSize: '12px',
                color: '#fff',
              }}
              itemStyle={{ color: '#fff' }}
              formatter={(value: unknown, name: unknown) =>
                typeof value === 'number'
                  ? [formatCurrency(value), String(name)]
                  : [String(value), String(name)]
              }
            />
            <Legend
              verticalAlign="bottom"
              height={36}
              formatter={(value) => (
                <span style={{ color: '#d4d4d8', fontSize: '12px', fontWeight: 'bold' }}>
                  {value}
                </span>
              )}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  )
}
