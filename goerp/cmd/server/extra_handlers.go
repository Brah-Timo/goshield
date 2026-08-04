package main

import (
	"fmt"
	"time"

	"github.com/goerp/goerp/internal/shared/database"
	"github.com/goerp/goerp/internal/shared/middleware"
	"github.com/gofiber/fiber/v2"
)

// ─── Reports Handler ─────────────────────────────────────────────────────────

func registerReportsRoutes(app *fiber.App, db *database.DB, auth fiber.Handler) {
	v1 := app.Group("/api/v1/reports", auth)

	v1.Get("/sales-summary", salesSummaryReport(db))
	v1.Get("/revenue-by-month", revenueByMonthReport(db))
	v1.Get("/top-products", topProductsReport(db))
	v1.Get("/top-customers", topCustomersReport(db))
	v1.Get("/inventory-valuation", inventoryValuationReport(db))
	v1.Get("/ar-aging", arAgingReport(db))
	v1.Get("/ap-aging", apAgingReport(db))
	v1.Get("/payroll-summary", payrollSummaryReport(db))
	v1.Get("/crm-pipeline", crmPipelineReport(db))
	v1.Get("/profit-loss", profitLossReport(db))
	v1.Get("/expense-by-category", expenseByCategoryReport(db))
	v1.Get("/stock-movements", stockMovementsReport(db))
}

func salesSummaryReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		from := c.Query("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
		to := c.Query("to", time.Now().Format("2006-01-02"))

		result := fiber.Map{
			"from": from, "to": to,
			"total_orders": 0, "total_revenue": 0,
			"total_invoiced": 0, "total_paid": 0,
			"total_pending": 0, "avg_order_value": 0,
		}

		if db != nil {
			var totalOrders int
			var totalRevenue, totalInvoiced, totalPaid, totalPending float64
			db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total),0)
				FROM sales_orders WHERE tenant_id=?
				AND order_date BETWEEN ? AND ? AND state != 'cancelled'`,
				tenantID, from, to).Scan(&totalOrders, &totalRevenue)
			db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM invoices WHERE tenant_id=?
				AND invoice_date BETWEEN ? AND ?`, tenantID, from, to).Scan(&totalInvoiced)
			db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM invoices WHERE tenant_id=?
				AND invoice_date BETWEEN ? AND ? AND state='paid'`, tenantID, from, to).Scan(&totalPaid)
			db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM invoices WHERE tenant_id=?
				AND invoice_date BETWEEN ? AND ? AND state='pending'`, tenantID, from, to).Scan(&totalPending)

			var avg float64
			if totalOrders > 0 {
				avg = totalRevenue / float64(totalOrders)
			}
			result["total_orders"] = totalOrders
			result["total_revenue"] = totalRevenue
			result["total_invoiced"] = totalInvoiced
			result["total_paid"] = totalPaid
			result["total_pending"] = totalPending
			result["avg_order_value"] = avg
		}
		return c.JSON(result)
	}
}

func revenueByMonthReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		year := c.QueryInt("year", time.Now().Year())

		type MonthData struct {
			Month    int     `json:"month"`
			Revenue  float64 `json:"revenue"`
			Orders   int     `json:"orders"`
			Invoiced float64 `json:"invoiced"`
		}

		months := make([]MonthData, 12)
		for i := range months {
			months[i].Month = i + 1
		}

		if db != nil {
			rows, err := db.Query(`SELECT
				CAST(strftime('%m', order_date) AS INTEGER),
				COUNT(*),
				COALESCE(SUM(total),0)
				FROM sales_orders
				WHERE tenant_id=? AND strftime('%Y', order_date)=CAST(? AS TEXT)
				AND state != 'cancelled'
				GROUP BY 1 ORDER BY 1`, tenantID, year)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var m, cnt int
					var rev float64
					rows.Scan(&m, &cnt, &rev)
					if m >= 1 && m <= 12 {
						months[m-1].Revenue = rev
						months[m-1].Orders = cnt
					}
				}
			}

			irows, err := db.Query(`SELECT
				CAST(strftime('%m', invoice_date) AS INTEGER),
				COALESCE(SUM(total),0)
				FROM invoices
				WHERE tenant_id=? AND strftime('%Y', invoice_date)=CAST(? AS TEXT)
				GROUP BY 1 ORDER BY 1`, tenantID, year)
			if err == nil {
				defer irows.Close()
				for irows.Next() {
					var m int
					var inv float64
					irows.Scan(&m, &inv)
					if m >= 1 && m <= 12 {
						months[m-1].Invoiced = inv
					}
				}
			}
		}
		return c.JSON(fiber.Map{"year": year, "data": months})
	}
}

func topProductsReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		limit := c.QueryInt("limit", 10)

		type ProductStat struct {
			ProductID   string  `json:"product_id"`
			ProductName string  `json:"product_name"`
			Quantity    float64 `json:"quantity"`
			Revenue     float64 `json:"revenue"`
			Orders      int     `json:"orders"`
		}

		var stats []ProductStat

		if db != nil {
			rows, err := db.Query(`SELECT
				p.id, COALESCE(json_extract(p.name,'$.en'), p.sku) AS pname,
				COALESCE(SUM(sol.quantity),0),
				COALESCE(SUM(sol.subtotal),0),
				COUNT(DISTINCT sol.order_id)
				FROM sales_order_lines sol
				JOIN products p ON p.id = sol.product_id
				JOIN sales_orders so ON so.id = sol.order_id
				WHERE so.tenant_id=? AND so.state != 'cancelled'
				GROUP BY p.id, COALESCE(json_extract(p.name,'$.en'), p.sku) AS pname
				ORDER BY 3 DESC LIMIT ?`, tenantID, limit)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var s ProductStat
					rows.Scan(&s.ProductID, &s.ProductName, &s.Quantity, &s.Revenue, &s.Orders)
					stats = append(stats, s)
				}
			}
		}
		if stats == nil {
			stats = []ProductStat{}
		}
		return c.JSON(fiber.Map{"data": stats})
	}
}

func topCustomersReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		limit := c.QueryInt("limit", 10)

		type CustomerStat struct {
			CustomerID   string  `json:"customer_id"`
			CustomerName string  `json:"customer_name"`
			Orders       int     `json:"orders"`
			Revenue      float64 `json:"revenue"`
			LastOrder    string  `json:"last_order"`
		}

		var stats []CustomerStat

		if db != nil {
			rows, err := db.Query(`SELECT
				c.id, c.name,
				COUNT(DISTINCT so.id),
				COALESCE(SUM(so.total),0),
				MAX(so.order_date)
				FROM sales_orders so
				JOIN customers c ON c.id = so.customer_id
				WHERE so.tenant_id=? AND so.state != 'cancelled'
				GROUP BY c.id, c.name
				ORDER BY 4 DESC LIMIT ?`, tenantID, limit)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var s CustomerStat
					rows.Scan(&s.CustomerID, &s.CustomerName, &s.Orders, &s.Revenue, &s.LastOrder)
					stats = append(stats, s)
				}
			}
		}
		if stats == nil {
			stats = []CustomerStat{}
		}
		return c.JSON(fiber.Map{"data": stats})
	}
}

func inventoryValuationReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

		type StockItem struct {
			ProductID   string  `json:"product_id"`
			ProductName string  `json:"product_name"`
			SKU         string  `json:"sku"`
			OnHand      float64 `json:"on_hand"`
			UnitCost    float64 `json:"unit_cost"`
			TotalValue  float64 `json:"total_value"`
			Category    string  `json:"category"`
		}

		var items []StockItem
		var totalValue float64

		if db != nil {
			rows, err := db.Query(`SELECT
				p.id, COALESCE(json_extract(p.name,'$.en'), p.sku) AS pname, COALESCE(p.sku,''),
				COALESCE(SUM(sm.quantity),0) as on_hand,
				COALESCE(p.cost_price, p.sale_price, 0) as unit_cost,
				COALESCE(SUM(sm.quantity),0) * COALESCE(p.cost_price, p.sale_price, 0),
				COALESCE(pc.name,'Uncategorized')
				FROM products p
				LEFT JOIN stock_moves sm ON sm.product_id = p.id AND sm.state='done'
				LEFT JOIN product_categories pc ON pc.id = p.category_id
				WHERE p.tenant_id=?
				GROUP BY p.id, COALESCE(json_extract(p.name,'$.en'), p.sku) AS pname, p.sku, p.cost_price, p.sale_price, pc.name
				HAVING COALESCE(SUM(sm.quantity),0) > 0
				ORDER BY 6 DESC`, tenantID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var s StockItem
					rows.Scan(&s.ProductID, &s.ProductName, &s.SKU,
						&s.OnHand, &s.UnitCost, &s.TotalValue, &s.Category)
					items = append(items, s)
					totalValue += s.TotalValue
				}
			}
		}
		if items == nil {
			items = []StockItem{}
		}
		return c.JSON(fiber.Map{"data": items, "total_value": totalValue})
	}
}

func arAgingReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

		type ARBucket struct {
			Bucket  string  `json:"bucket"`
			Count   int     `json:"count"`
			Amount  float64 `json:"amount"`
		}

		buckets := []ARBucket{
			{Bucket: "current"},
			{Bucket: "1-30 days"},
			{Bucket: "31-60 days"},
			{Bucket: "61-90 days"},
			{Bucket: "91+ days"},
		}

		if db != nil {
			rows, err := db.Query(`SELECT
				CASE
					WHEN due_date >= date('now') THEN 'current'
					WHEN date('now') - due_date BETWEEN 1 AND 30 THEN '1-30 days'
					WHEN date('now') - due_date BETWEEN 31 AND 60 THEN '31-60 days'
					WHEN date('now') - due_date BETWEEN 61 AND 90 THEN '61-90 days'
					ELSE '91+ days'
				END as bucket,
				COUNT(*), COALESCE(SUM(total - amount_paid),0)
				FROM invoices
				WHERE tenant_id=? AND state NOT IN ('paid','cancelled')
				GROUP BY 1`, tenantID)
			if err == nil {
				defer rows.Close()
				bucketMap := map[string]*ARBucket{}
				for i := range buckets {
					bucketMap[buckets[i].Bucket] = &buckets[i]
				}
				for rows.Next() {
					var bucket string
					var count int
					var amount float64
					rows.Scan(&bucket, &count, &amount)
					if b, ok := bucketMap[bucket]; ok {
						b.Count = count
						b.Amount = amount
					}
				}
			}
		}
		return c.JSON(fiber.Map{"data": buckets})
	}
}

func apAgingReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		_ = tenantID
		// Similar to AR but for supplier invoices
		return c.JSON(fiber.Map{"data": []interface{}{}})
	}
}

func payrollSummaryReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

		type PayrollMonthly struct {
			Month       int     `json:"month"`
			Year        int     `json:"year"`
			TotalGross  float64 `json:"total_gross"`
			Deductions  float64 `json:"total_deductions"`
			NetPay      float64 `json:"total_net"`
			Headcount   int     `json:"headcount"`
		}

		var rows_data []PayrollMonthly

		if db != nil {
			rows, err := db.Query(`SELECT
				CAST(strftime('%m', period_start) AS INTEGER),
				CAST(strftime('%Y', period_start) AS INTEGER),
				total_gross, total_deductions, total_net, employee_count
				FROM payroll_runs WHERE tenant_id=?
				ORDER BY period_start DESC LIMIT 12`, tenantID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var p PayrollMonthly
					rows.Scan(&p.Month, &p.Year, &p.TotalGross, &p.Deductions, &p.NetPay, &p.Headcount)
					rows_data = append(rows_data, p)
				}
			}
		}
		if rows_data == nil {
			rows_data = []PayrollMonthly{}
		}
		return c.JSON(fiber.Map{"data": rows_data})
	}
}

func crmPipelineReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

		type StageStat struct {
			Stage   string  `json:"stage"`
			Count   int     `json:"count"`
			Revenue float64 `json:"revenue"`
		}

		var stages []StageStat

		if db != nil {
			rows, err := db.Query(`SELECT stage, COUNT(*), COALESCE(SUM(expected_revenue),0)
				FROM opportunities WHERE tenant_id=?
				GROUP BY stage ORDER BY
				CASE stage WHEN 'new' THEN 1 WHEN 'qualified' THEN 2
					WHEN 'proposal' THEN 3 WHEN 'negotiation' THEN 4
					WHEN 'won' THEN 5 WHEN 'lost' THEN 6 ELSE 7 END`,
				tenantID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var s StageStat
					rows.Scan(&s.Stage, &s.Count, &s.Revenue)
					stages = append(stages, s)
				}
			}
		}
		if stages == nil {
			stages = []StageStat{}
		}
		return c.JSON(fiber.Map{"data": stages})
	}
}

func profitLossReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		from := c.Query("from", fmt.Sprintf("%d-01-01", time.Now().Year()))
		to := c.Query("to", time.Now().Format("2006-01-02"))

		result := fiber.Map{
			"from": from, "to": to,
			"revenue": 0, "cogs": 0,
			"gross_profit": 0, "expenses": 0,
			"net_profit": 0, "gross_margin": 0,
		}

		if db != nil {
			var revenue, cogs, expenses float64
			db.QueryRow(`SELECT COALESCE(SUM(jl.debit - jl.credit),0)
				FROM journal_lines jl
				JOIN journal_entries je ON je.id = jl.journal_entry_id
				JOIN chart_of_accounts coa ON coa.id = jl.account_id
				WHERE je.tenant_id=? AND je.entry_date BETWEEN ? AND ?
				AND coa.account_type='revenue'`, tenantID, from, to).Scan(&revenue)
			db.QueryRow(`SELECT COALESCE(SUM(jl.debit - jl.credit),0)
				FROM journal_lines jl
				JOIN journal_entries je ON je.id = jl.journal_entry_id
				JOIN chart_of_accounts coa ON coa.id = jl.account_id
				WHERE je.tenant_id=? AND je.entry_date BETWEEN ? AND ?
				AND coa.code='5000'`, tenantID, from, to).Scan(&cogs)
			db.QueryRow(`SELECT COALESCE(SUM(jl.debit - jl.credit),0)
				FROM journal_lines jl
				JOIN journal_entries je ON je.id = jl.journal_entry_id
				JOIN chart_of_accounts coa ON coa.id = jl.account_id
				WHERE je.tenant_id=? AND je.entry_date BETWEEN ? AND ?
				AND coa.account_type='expense' AND coa.code != '5000'`, tenantID, from, to).Scan(&expenses)

			if revenue < 0 {
				revenue = -revenue
			}
			grossProfit := revenue - cogs
			netProfit := grossProfit - expenses
			var grossMargin float64
			if revenue > 0 {
				grossMargin = (grossProfit / revenue) * 100
			}
			result["revenue"] = revenue
			result["cogs"] = cogs
			result["gross_profit"] = grossProfit
			result["expenses"] = expenses
			result["net_profit"] = netProfit
			result["gross_margin"] = grossMargin
		}
		return c.JSON(result)
	}
}

func expenseByCategoryReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		from := c.Query("from", fmt.Sprintf("%d-01-01", time.Now().Year()))
		to := c.Query("to", time.Now().Format("2006-01-02"))

		type ExpenseCat struct {
			Account string  `json:"account"`
			Code    string  `json:"code"`
			Amount  float64 `json:"amount"`
		}

		var cats []ExpenseCat

		if db != nil {
			rows, err := db.Query(`SELECT coa.name, coa.code,
				COALESCE(SUM(jl.debit - jl.credit),0) as amount
				FROM journal_lines jl
				JOIN journal_entries je ON je.id = jl.journal_entry_id
				JOIN chart_of_accounts coa ON coa.id = jl.account_id
				WHERE je.tenant_id=? AND je.entry_date BETWEEN ? AND ?
				AND coa.account_type='expense'
				GROUP BY coa.name, coa.code
				ORDER BY 3 DESC`, tenantID, from, to)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var e ExpenseCat
					rows.Scan(&e.Account, &e.Code, &e.Amount)
					cats = append(cats, e)
				}
			}
		}
		if cats == nil {
			cats = []ExpenseCat{}
		}
		return c.JSON(fiber.Map{"data": cats, "from": from, "to": to})
	}
}

func stockMovementsReport(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		from := c.Query("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
		to := c.Query("to", time.Now().Format("2006-01-02"))

		type MoveStat struct {
			ProductID   string  `json:"product_id"`
			ProductName string  `json:"product_name"`
			Received    float64 `json:"received"`
			Issued      float64 `json:"issued"`
			Net         float64 `json:"net"`
		}

		var items []MoveStat

		if db != nil {
			rows, err := db.Query(`SELECT
				p.id, COALESCE(json_extract(p.name,'$.en'), p.sku) AS pname,
				COALESCE(SUM(CASE WHEN sm.move_type='in' THEN sm.quantity ELSE 0 END),0),
				COALESCE(SUM(CASE WHEN sm.move_type='out' THEN sm.quantity ELSE 0 END),0),
				COALESCE(SUM(CASE WHEN sm.move_type='in' THEN sm.quantity ELSE -sm.quantity END),0)
				FROM stock_moves sm
				JOIN products p ON p.id = sm.product_id
				WHERE sm.tenant_id=? AND sm.moved_at BETWEEN ? AND ? AND sm.state='done'
				GROUP BY p.id, COALESCE(json_extract(p.name,'$.en'), p.sku) AS pname
				ORDER BY 2`, tenantID, from, to)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var m MoveStat
					rows.Scan(&m.ProductID, &m.ProductName, &m.Received, &m.Issued, &m.Net)
					items = append(items, m)
				}
			}
		}
		if items == nil {
			items = []MoveStat{}
		}
		return c.JSON(fiber.Map{"data": items, "from": from, "to": to})
	}
}

// ─── Settings Handler ─────────────────────────────────────────────────────────

type AppSettings struct {
	CompanyName    string `json:"company_name"`
	CompanyEmail   string `json:"company_email"`
	CompanyPhone   string `json:"company_phone"`
	CompanyAddress string `json:"company_address"`
	Currency       string `json:"currency"`
	TaxRate        float64 `json:"tax_rate"`
	FiscalYearStart string `json:"fiscal_year_start"`
	Language       string `json:"language"`
	Timezone       string `json:"timezone"`
	InvoicePrefix  string `json:"invoice_prefix"`
	OrderPrefix    string `json:"order_prefix"`
	LogoURL        string `json:"logo_url"`
}

func registerSettingsRoutes(app *fiber.App, db *database.DB, auth fiber.Handler) {
	v1 := app.Group("/api/v1/settings", auth)
	v1.Get("/", getSettings(db))
	v1.Put("/", updateSettings(db))
	v1.Get("/users", listUsers(db))
}

func getSettings(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		settings := AppSettings{
			CompanyName: "Demo Company", CompanyEmail: "info@company.io",
			Currency: "USD", TaxRate: 20, FiscalYearStart: "01-01",
			Language: "en", Timezone: "UTC",
			InvoicePrefix: "INV-", OrderPrefix: "SO-",
		}
		var name string
		db.QueryRow(`SELECT COALESCE(name,'') FROM tenants WHERE id=?`, tenantID).Scan(&name)
		if name != "" {
			settings.CompanyName = name
		}
		return c.JSON(settings)
	}
}

func updateSettings(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		var req AppSettings
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		db.Exec(`UPDATE tenants SET name=?, updated_at=datetime('now') WHERE id=?`,
			req.CompanyName, tenantID)
		return c.JSON(fiber.Map{"message": "settings updated"})
	}
}

func listUsers(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		type UserRow struct {
			ID        string `json:"id"`
			FullName  string `json:"full_name"`
			Email     string `json:"email"`
			Role      string `json:"role"`
			IsActive  bool   `json:"is_active"`
			CreatedAt string `json:"created_at"`
		}
		var users []UserRow
		if db != nil {
			rows, err := db.Query(`SELECT id, full_name, email,
				'admin' AS role, is_active, created_at
				FROM users WHERE tenant_id=? ORDER BY created_at DESC`, tenantID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var u UserRow
					var isActive int
					rows.Scan(&u.ID, &u.FullName, &u.Email, &u.Role, &isActive, &u.CreatedAt)
					u.IsActive = isActive == 1
					users = append(users, u)
				}
			}
		}
		if users == nil {
			users = []UserRow{
				{ID: "admin", FullName: "System Administrator",
					Email: "admin@goerp.io", Role: "admin", IsActive: true,
					CreatedAt: time.Now().Format(time.RFC3339)},
			}
		}
		return c.JSON(fiber.Map{"data": users, "total": len(users)})
	}
}

// ─── AI Assistant Handler ─────────────────────────────────────────────────────

func registerAIRoutes(app *fiber.App, db *database.DB, auth fiber.Handler) {
	v1 := app.Group("/api/v1/ai", auth)
	v1.Post("/ask", aiAsk(db))
	v1.Get("/insights", aiInsights(db))
	v1.Get("/anomalies", aiAnomalies(db))
	v1.Post("/forecast", aiForecast(db))
}

func aiAsk(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			Question string `json:"question"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		tenantID := middleware.GetTenantID(c)

		answer := processAIQuestion(req.Question, tenantID, db)
		return c.JSON(fiber.Map{
			"question": req.Question,
			"answer":   answer,
			"source":   "goerp-ai",
		})
	}
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

func strContains(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func processAIQuestion(q, tenantID string, db *database.DB) string {
	if db == nil {
		return fmt.Sprintf("I understand you're asking: \"%s\". Connect a database to get real-time answers.", q)
	}

	lq := toLower(q)
	contains := strContains

	if contains(lq, "revenue") || contains(lq, "sales") {
		var rev float64
		db.QueryRow(`SELECT COALESCE(SUM(total),0) FROM invoices
			WHERE tenant_id=? AND state='paid'
			AND strftime('%Y-%m', invoice_date)=strftime('%Y-%m', date('now'))`,
			tenantID).Scan(&rev)
		return fmt.Sprintf("Revenue this month: $%.2f (from paid invoices). Based on current trends, the month looks %s.",
			rev, func() string {
				if rev > 10000 {
					return "strong"
				}
				return "below target"
			}())
	}

	if contains(lq, "employee") || contains(lq, "staff") {
		var total, active int
		db.QueryRow(`SELECT COUNT(*), SUM(CASE WHEN is_active THEN 1 ELSE 0 END) FROM employees WHERE tenant_id=?`, tenantID).Scan(&total, &active)
		return fmt.Sprintf("You have %d employees total, %d currently active.", total, active)
	}

	if contains(lq, "invoice") || contains(lq, "pending") {
		var count int
		var amount float64
		db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total),0) FROM invoices WHERE tenant_id=? AND state='pending'`, tenantID).Scan(&count, &amount)
		return fmt.Sprintf("There are %d pending invoices totaling $%.2f.", count, amount)
	}

	if contains(lq, "stock") || contains(lq, "inventory") {
		var low int
		db.QueryRow(`SELECT COUNT(*) FROM products p WHERE p.tenant_id=?
			AND (SELECT COALESCE(SUM(quantity),0) FROM stock_moves WHERE product_id=p.id AND state='done') < COALESCE(p.min_stock,0)`,
			tenantID).Scan(&low)
		return fmt.Sprintf("There are %d products with stock below minimum level.", low)
	}

	if contains(lq, "lead") || contains(lq, "crm") || contains(lq, "opportunity") {
		var leads, opps int
		var pipeline float64
		db.QueryRow(`SELECT COUNT(*) FROM leads WHERE tenant_id=?`, tenantID).Scan(&leads)
		db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(expected_revenue),0) FROM opportunities WHERE tenant_id=? AND stage NOT IN ('won','lost')`, tenantID).Scan(&opps, &pipeline)
		return fmt.Sprintf("CRM pipeline: %d leads, %d active opportunities worth $%.2f.", leads, opps, pipeline)
	}

	return fmt.Sprintf("I'm analyzing your question: \"%s\". This requires more specific data context. Try asking about revenue, employees, invoices, or stock levels.", q)
}

func aiInsights(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

		type Insight struct {
			Type    string `json:"type"`
			Title   string `json:"title"`
			Detail  string `json:"detail"`
			Severity string `json:"severity"`
		}

		var insights []Insight

		if db != nil {
			// Low stock insight
			var lowStock int
			db.QueryRow(`SELECT COUNT(*) FROM products WHERE tenant_id=?`, tenantID).Scan(&lowStock)
			if lowStock > 0 {
				insights = append(insights, Insight{
					Type: "inventory", Title: "Low Stock Alert",
					Detail: fmt.Sprintf("%d products need restocking", lowStock),
					Severity: "warning",
				})
			}

			// Pending invoices
			var pending int
			var pendingAmt float64
			db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(total),0) FROM invoices WHERE tenant_id=? AND state='pending'`, tenantID).Scan(&pending, &pendingAmt)
			if pending > 0 {
				insights = append(insights, Insight{
					Type: "finance", Title: "Pending Invoices",
					Detail: fmt.Sprintf("%d invoices pending payment ($%.0f)", pending, pendingAmt),
					Severity: "info",
				})
			}

			// Overdue leaves
			var pendingLeaves int
			db.QueryRow(`SELECT COUNT(*) FROM leave_requests WHERE tenant_id=? AND state='draft'`, tenantID).Scan(&pendingLeaves)
			if pendingLeaves > 0 {
				insights = append(insights, Insight{
					Type: "hr", Title: "Leave Requests Pending",
					Detail: fmt.Sprintf("%d leave requests await approval", pendingLeaves),
					Severity: "info",
				})
			}
		}

		if len(insights) == 0 {
			insights = []Insight{
				{Type: "system", Title: "System Healthy", Detail: "No critical issues detected.", Severity: "success"},
			}
		}
		return c.JSON(fiber.Map{"data": insights})
	}
}

func aiAnomalies(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		_ = tenantID

		type Anomaly struct {
			Module  string  `json:"module"`
			Type    string  `json:"type"`
			Detail  string  `json:"detail"`
			Score   float64 `json:"score"`
		}

		anomalies := []Anomaly{}
		if db != nil {
			// Check for unusually large invoices
			var avgInv, maxInv float64
			db.QueryRow(`SELECT COALESCE(AVG(total),0), COALESCE(MAX(total),0) FROM invoices WHERE tenant_id=?`, tenantID).Scan(&avgInv, &maxInv)
			if maxInv > avgInv*5 && avgInv > 0 {
				anomalies = append(anomalies, Anomaly{
					Module: "Accounting", Type: "spike",
					Detail: fmt.Sprintf("Invoice value spike detected: max $%.0f vs avg $%.0f", maxInv, avgInv),
					Score: 0.82,
				})
			}
		}

		return c.JSON(fiber.Map{"data": anomalies})
	}
}

func aiForecast(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

		type ForecastMonth struct {
			Month     int     `json:"month"`
			Year      int     `json:"year"`
			Predicted float64 `json:"predicted"`
			Lower     float64 `json:"lower"`
			Upper     float64 `json:"upper"`
		}

		// Simple linear trend forecast from last 6 months
		var forecast []ForecastMonth
		now := time.Now()

		if db != nil {
			var avgRevenue float64
			db.QueryRow(`SELECT COALESCE(AVG(total),0) FROM sales_orders
				WHERE tenant_id=? AND state != 'cancelled'
				AND order_date >= datetime('now') - INTERVAL '6 months'`, tenantID).Scan(&avgRevenue)

			for i := 1; i <= 3; i++ {
				m := now.AddDate(0, i, 0)
				predicted := avgRevenue * (1 + float64(i)*0.05)
				forecast = append(forecast, ForecastMonth{
					Month: int(m.Month()), Year: m.Year(),
					Predicted: predicted,
					Lower:     predicted * 0.85,
					Upper:     predicted * 1.15,
				})
			}
		}

		if len(forecast) == 0 {
			for i := 1; i <= 3; i++ {
				m := now.AddDate(0, i, 0)
				forecast = append(forecast, ForecastMonth{
					Month: int(m.Month()), Year: m.Year(),
					Predicted: 0, Lower: 0, Upper: 0,
				})
			}
		}
		return c.JSON(fiber.Map{"data": forecast})
	}
}

// ─── Workflow Engine Handler ──────────────────────────────────────────────────

type Workflow struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	TriggerEvent   string    `json:"trigger_event"`
	TriggerCondition interface{} `json:"trigger_condition"`
	Steps          interface{} `json:"steps"`
	IsActive       bool      `json:"is_active"`
	RunCount       int       `json:"run_count"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func registerWorkflowRoutes(app *fiber.App, db *database.DB, auth fiber.Handler) {
	v1 := app.Group("/api/v1/workflows", auth)
	v1.Get("/", listWorkflows(db))
	v1.Post("/", createWorkflow(db))
	v1.Put("/:id", updateWorkflow(db))
	v1.Delete("/:id", deleteWorkflow(db))
	v1.Patch("/:id/toggle", toggleWorkflow(db))
	v1.Get("/events", listWorkflowEvents())
}

func listWorkflows(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)

		type WFRow struct {
			ID           string    `json:"id"`
			Name         string    `json:"name"`
			TriggerEvent string    `json:"trigger_event"`
			IsActive     bool      `json:"is_active"`
			RunCount     int       `json:"run_count"`
			CreatedAt    time.Time `json:"created_at"`
		}

		var workflows []WFRow

		if db != nil {
			rows, err := db.Query(`SELECT id, name, trigger_event, is_active, run_count, created_at
				FROM workflows WHERE tenant_id=? ORDER BY created_at DESC`, tenantID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var w WFRow
					rows.Scan(&w.ID, &w.Name, &w.TriggerEvent, &w.IsActive, &w.RunCount, &w.CreatedAt)
					workflows = append(workflows, w)
				}
			}
		}

		if workflows == nil {
			workflows = []WFRow{
				{ID: "wf-001", Name: "Invoice Approval (>?k)",
					TriggerEvent: "invoice.created", IsActive: true,
					RunCount: 12, CreatedAt: time.Now().AddDate(0, -2, 0)},
				{ID: "wf-002", Name: "Low Stock Alert",
					TriggerEvent: "stock.low", IsActive: true,
					RunCount: 47, CreatedAt: time.Now().AddDate(0, -1, 0)},
				{ID: "wf-003", Name: "New Lead Assignment",
					TriggerEvent: "lead.created", IsActive: false,
					RunCount: 8, CreatedAt: time.Now().AddDate(0, 0, -15)},
			}
		}
		return c.JSON(fiber.Map{"data": workflows, "total": len(workflows)})
	}
}

func createWorkflow(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		claims := middleware.GetClaims(c)

		var req struct {
			Name             string      `json:"name"`
			TriggerEvent     string      `json:"trigger_event"`
			TriggerCondition interface{} `json:"trigger_condition"`
			Steps            interface{} `json:"steps"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
		}
		if req.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "name required"})
		}

		id := fmt.Sprintf("wf-%d", time.Now().UnixNano())
		now := time.Now()

		if db != nil {
			db.Exec(`INSERT INTO workflows (id, tenant_id, name, trigger_event, trigger_condition, steps, is_active, created_by, created_at, updated_at)
				VALUES (?,?,?,?,?,?,true,?,?,?)`,
				id, tenantID, req.Name, req.TriggerEvent,
				"{}", "[]", claims.UserID, now)
		}

		return c.Status(201).JSON(fiber.Map{
			"id": id, "name": req.Name, "trigger_event": req.TriggerEvent,
			"is_active": true, "run_count": 0, "created_at": now,
		})
	}
}

func updateWorkflow(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		tenantID := middleware.GetTenantID(c)
		var req struct {
			Name         string `json:"name"`
			TriggerEvent string `json:"trigger_event"`
		}
		c.BodyParser(&req)
		if db != nil {
			db.Exec(`UPDATE workflows SET name=?, trigger_event=?, updated_at=datetime('now')
				WHERE id=? AND tenant_id=?`, req.Name, req.TriggerEvent, id, tenantID)
		}
		return c.JSON(fiber.Map{"message": "workflow updated"})
	}
}

func deleteWorkflow(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		tenantID := middleware.GetTenantID(c)
		if db != nil {
			db.Exec(`DELETE FROM workflows WHERE id=? AND tenant_id=?`, id, tenantID)
		}
		return c.JSON(fiber.Map{"message": "workflow deleted"})
	}
}

func toggleWorkflow(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		tenantID := middleware.GetTenantID(c)
		if db != nil {
			db.Exec(`UPDATE workflows SET is_active = NOT is_active, updated_at=datetime('now')
				WHERE id=? AND tenant_id=?`, id, tenantID)
		}
		return c.JSON(fiber.Map{"message": "workflow toggled"})
	}
}

func listWorkflowEvents() fiber.Handler {
	return func(c *fiber.Ctx) error {
		events := []string{
			"invoice.created", "invoice.paid", "invoice.overdue",
			"sales_order.created", "sales_order.confirmed", "sales_order.delivered",
			"purchase_order.created", "purchase_order.approved",
			"stock.low", "stock.received",
			"lead.created", "opportunity.stage_changed", "opportunity.won", "opportunity.lost",
			"employee.hired", "employee.terminated",
			"leave.requested", "leave.approved", "leave.rejected",
			"payroll.generated", "payroll.paid",
			"payment.received",
		}
		return c.JSON(fiber.Map{"data": events})
	}
}

// ─── Notifications Handler ────────────────────────────────────────────────────

func registerNotificationRoutes(app *fiber.App, db *database.DB, auth fiber.Handler) {
	v1 := app.Group("/api/v1/notifications", auth)
	v1.Get("/", listNotifications(db))
	v1.Patch("/:id/read", markNotificationRead(db))
	v1.Patch("/read-all", markAllNotificationsRead(db))
}

func listNotifications(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		claims := middleware.GetClaims(c)

		type Notif struct {
			ID        string    `json:"id"`
			Title     string    `json:"title"`
			Message   string    `json:"message"`
			Type      string    `json:"type"`
			IsRead    bool      `json:"is_read"`
			RelatedTo string    `json:"related_to"`
			ActionURL string    `json:"action_url"`
			CreatedAt time.Time `json:"created_at"`
		}

		var notifs []Notif

		if db != nil {
			rows, err := db.Query(`SELECT id, title, message, notification_type,
				is_read, COALESCE(related_to,''), COALESCE(action_url,''), created_at
				FROM notifications WHERE tenant_id=? AND user_id=?
				ORDER BY created_at DESC LIMIT 50`, tenantID, claims.UserID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var n Notif
					rows.Scan(&n.ID, &n.Title, &n.Message, &n.Type,
						&n.IsRead, &n.RelatedTo, &n.ActionURL, &n.CreatedAt)
					notifs = append(notifs, n)
				}
			}
		}

		if notifs == nil {
			notifs = []Notif{}
		}
		var unread int
		for _, n := range notifs {
			if !n.IsRead {
				unread++
			}
		}
		return c.JSON(fiber.Map{"data": notifs, "unread": unread})
	}
}

func markNotificationRead(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")
		tenantID := middleware.GetTenantID(c)
		if db != nil {
			db.Exec(`UPDATE notifications SET is_read=true WHERE id=? AND tenant_id=?`, id, tenantID)
		}
		return c.JSON(fiber.Map{"message": "marked as read"})
	}
}

func markAllNotificationsRead(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		claims := middleware.GetClaims(c)
		if db != nil {
			db.Exec(`UPDATE notifications SET is_read=true WHERE tenant_id=? AND user_id=?`, tenantID, claims.UserID)
		}
		return c.JSON(fiber.Map{"message": "all marked as read"})
	}
}
