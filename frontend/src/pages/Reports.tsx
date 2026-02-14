import { useState, useEffect } from 'react'
import { getToken } from '../services/api'
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    BarElement,
    LineElement,
    ArcElement,
    PointElement,
    Title,
    Tooltip,
    Legend,
} from 'chart.js'
import { Bar, Pie, Line } from 'react-chartjs-2'
import * as XLSX from 'xlsx'

// Register Chart.js components
ChartJS.register(
    CategoryScale,
    LinearScale,
    BarElement,
    LineElement,
    ArcElement,
    PointElement,
    Title,
    Tooltip,
    Legend
)

const API_BASE = 'http://localhost:8080/api'

interface ReportSummary {
    total_requests: number
    total_amount: number
    total_approved: number
    total_pending: number
    total_rejected: number
    advance_total: number
    expense_total: number
    outstanding_advances: number
}

interface MonthlyData {
    month: string
    count: number
    amount: number
    advance: number
    expense: number
}

interface TypeBreakdown {
    type: string
    count: number
    amount: number
}

interface TrendData {
    date: string
    count: number
    amount: number
}

interface DetailRow {
    id: number
    user_id: number
    type: string
    amount: number
    description: string
    status: string
    approved_by_op_head_amount: number | null
    approved_by_finance_amount: number | null
    created_at: string
}

export default function Reports() {
    const [summary, setSummary] = useState<ReportSummary | null>(null)
    const [monthlyData, setMonthlyData] = useState<MonthlyData[]>([])
    const [typeBreakdown, setTypeBreakdown] = useState<TypeBreakdown[]>([])
    const [trends, setTrends] = useState<TrendData[]>([])
    const [details, setDetails] = useState<DetailRow[]>([])

    const [startDate, setStartDate] = useState('')
    const [endDate, setEndDate] = useState('')
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState('')

    useEffect(() => {
        // Set default date range (last 30 days)
        const end = new Date()
        const start = new Date()
        start.setDate(start.getDate() - 30)

        setStartDate(start.toISOString().split('T')[0])
        setEndDate(end.toISOString().split('T')[0])
    }, [])

    useEffect(() => {
        if (startDate && endDate) {
            fetchReports()
        }
    }, [startDate, endDate])

    const fetchReports = async () => {
        setLoading(true)
        setError('')

        const token = getToken()
        const headers = {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json',
        }

        const params = new URLSearchParams({
            start_date: startDate,
            end_date: endDate,
        })

        try {
            // Fetch all report data in parallel
            const [summaryRes, monthlyRes, breakdownRes, trendsRes, detailsRes] = await Promise.all([
                fetch(`${API_BASE}/reports/summary?${params}`, { headers }),
                fetch(`${API_BASE}/reports/monthly?${params}`, { headers }),
                fetch(`${API_BASE}/reports/breakdown?${params}`, { headers }),
                fetch(`${API_BASE}/reports/trends?${params}`, { headers }),
                fetch(`${API_BASE}/reports/details?${params}`, { headers }),
            ])

            if (!summaryRes.ok) throw new Error('Failed to fetch summary')
            if (!monthlyRes.ok) throw new Error('Failed to fetch monthly data')
            if (!breakdownRes.ok) throw new Error('Failed to fetch breakdown')
            if (!trendsRes.ok) throw new Error('Failed to fetch trends')
            if (!detailsRes.ok) throw new Error('Failed to fetch details')

            const [summaryData, monthlyDataRes, breakdownData, trendsData, detailsData] = await Promise.all([
                summaryRes.json(),
                monthlyRes.json(),
                breakdownRes.json(),
                trendsRes.json(),
                detailsRes.json(),
            ])

            setSummary(summaryData)
            setMonthlyData(monthlyDataRes)
            setTypeBreakdown(breakdownData)
            setTrends(trendsData)
            setDetails(detailsData)
        } catch (err: any) {
            setError(err.message || 'Failed to fetch reports')
        } finally {
            setLoading(false)
        }
    }

    const exportToExcel = () => {
        const ws = XLSX.utils.json_to_sheet(details.map(d => ({
            ID: d.id,
            'User ID': d.user_id,
            Type: d.type,
            Amount: d.amount,
            Description: d.description,
            Status: d.status,
            'Op Head Approved': d.approved_by_op_head_amount || 'N/A',
            'Finance Approved': d.approved_by_finance_amount || 'N/A',
            'Created At': new Date(d.created_at).toLocaleString(),
        })))

        const wb = XLSX.utils.book_new()
        XLSX.utils.book_append_sheet(wb, ws, 'Reimbursements')
        XLSX.writeFile(wb, `reimbursements_${startDate}_to_${endDate}.xlsx`)
    }

    const exportToCSV = () => {
        const ws = XLSX.utils.json_to_sheet(details.map(d => ({
            ID: d.id,
            'User ID': d.user_id,
            Type: d.type,
            Amount: d.amount,
            Description: d.description,
            Status: d.status,
            'Op Head Approved': d.approved_by_op_head_amount || 'N/A',
            'Finance Approved': d.approved_by_finance_amount || 'N/A',
            'Created At': new Date(d.created_at).toLocaleString(),
        })))

        const csv = XLSX.utils.sheet_to_csv(ws)
        const blob = new Blob([csv], { type: 'text/csv' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `reimbursements_${startDate}_to_${endDate}.csv`
        a.click()
        URL.revokeObjectURL(url)
    }

    // Chart configurations
    const monthlyChartData = {
        labels: monthlyData.map(m => m.month).reverse(),
        datasets: [
            {
                label: 'Advance',
                data: monthlyData.map(m => m.advance).reverse(),
                backgroundColor: 'rgba(99, 102, 241, 0.8)',
            },
            {
                label: 'Expense',
                data: monthlyData.map(m => m.expense).reverse(),
                backgroundColor: 'rgba(16, 185, 129, 0.8)',
            },
        ],
    }

    const pieChartData = {
        labels: typeBreakdown.map(t => t.type),
        datasets: [
            {
                data: typeBreakdown.map(t => t.amount),
                backgroundColor: [
                    'rgba(99, 102, 241, 0.8)',
                    'rgba(16, 185, 129, 0.8)',
                ],
            },
        ],
    }

    const lineChartData = {
        labels: trends.map(t => t.date),
        datasets: [
            {
                label: 'Amount',
                data: trends.map(t => t.amount),
                borderColor: 'rgba(99, 102, 241, 1)',
                backgroundColor: 'rgba(99, 102, 241, 0.1)',
                tension: 0.4,
            },
        ],
    }

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                position: 'top' as const,
            },
        },
    }

    return (
        <div className="reports-container">
            <h1>Reports & Analytics</h1>

            {/* Date Range Picker */}
            <div className="date-range-picker">
                <div className="form-group">
                    <label>Start Date</label>
                    <input
                        type="date"
                        value={startDate}
                        onChange={(e) => setStartDate(e.target.value)}
                    />
                </div>
                <div className="form-group">
                    <label>End Date</label>
                    <input
                        type="date"
                        value={endDate}
                        onChange={(e) => setEndDate(e.target.value)}
                    />
                </div>
                <button onClick={fetchReports} disabled={loading}>
                    {loading ? 'Loading...' : 'Refresh'}
                </button>
            </div>

            {error && <div className="error">{error}</div>}

            {/* Summary Cards */}
            {summary && (
                <div className="summary-cards">
                    <div className="card">
                        <h3>Total Requests</h3>
                        <p className="stat">{summary.total_requests}</p>
                    </div>
                    <div className="card">
                        <h3>Total Amount</h3>
                        <p className="stat">₹{summary.total_amount.toFixed(2)}</p>
                    </div>
                    <div className="card success">
                        <h3>Approved</h3>
                        <p className="stat">₹{summary.total_approved.toFixed(2)}</p>
                    </div>
                    <div className="card warning">
                        <h3>Pending</h3>
                        <p className="stat">₹{summary.total_pending.toFixed(2)}</p>
                    </div>
                    <div className="card danger">
                        <h3>Rejected</h3>
                        <p className="stat">₹{summary.total_rejected.toFixed(2)}</p>
                    </div>
                    <div className="card info">
                        <h3>Outstanding Advances</h3>
                        <p className="stat">₹{summary.outstanding_advances.toFixed(2)}</p>
                    </div>
                </div>
            )}

            {/* Charts */}
            <div className="charts-grid">
                <div className="chart-container">
                    <h2>Monthly Breakdown</h2>
                    <div className="chart-wrapper">
                        <Bar data={monthlyChartData} options={chartOptions} />
                    </div>
                </div>

                <div className="chart-container">
                    <h2>Type Distribution</h2>
                    <div className="chart-wrapper">
                        <Pie data={pieChartData} options={chartOptions} />
                    </div>
                </div>

                <div className="chart-container full-width">
                    <h2>Trends Over Time</h2>
                    <div className="chart-wrapper">
                        <Line data={lineChartData} options={chartOptions} />
                    </div>
                </div>
            </div>

            {/* Export Buttons */}
            <div className="export-section">
                <h2>Export Data</h2>
                <div className="export-buttons">
                    <button onClick={exportToExcel} className="btn-primary">
                        📊 Export to Excel
                    </button>
                    <button onClick={exportToCSV} className="btn-secondary">
                        📄 Export to CSV
                    </button>
                </div>
            </div>

            {/* Data Table */}
            <div className="data-table-section">
                <h2>Detailed Transactions ({details.length})</h2>
                <div className="table-wrapper">
                    <table>
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Type</th>
                                <th>Amount</th>
                                <th>Description</th>
                                <th>Status</th>
                                <th>Op Head Approved</th>
                                <th>Finance Approved</th>
                                <th>Created At</th>
                            </tr>
                        </thead>
                        <tbody>
                            {details.map(d => (
                                <tr key={d.id}>
                                    <td>{d.id}</td>
                                    <td>
                                        <span className={`badge ${d.type.toLowerCase()}`}>
                                            {d.type}
                                        </span>
                                    </td>
                                    <td>₹{d.amount.toFixed(2)}</td>
                                    <td>{d.description}</td>
                                    <td>
                                        <span className={`status ${d.status.toLowerCase().replace(/_/g, '-')}`}>
                                            {d.status.replace(/_/g, ' ')}
                                        </span>
                                    </td>
                                    <td>
                                        {d.approved_by_op_head_amount
                                            ? `₹${d.approved_by_op_head_amount.toFixed(2)}`
                                            : '-'}
                                    </td>
                                    <td>
                                        {d.approved_by_finance_amount
                                            ? `₹${d.approved_by_finance_amount.toFixed(2)}`
                                            : '-'}
                                    </td>
                                    <td>{new Date(d.created_at).toLocaleString()}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    )
}
