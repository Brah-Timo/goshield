package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/goerp/goerp/configs"
	authdel "github.com/goerp/goerp/internal/auth/delivery"
	authrepo "github.com/goerp/goerp/internal/auth/repository"
	authuc "github.com/goerp/goerp/internal/auth/usecase"
	invdel "github.com/goerp/goerp/internal/inventory/delivery"
	invrepo "github.com/goerp/goerp/internal/inventory/repository"
	invuc "github.com/goerp/goerp/internal/inventory/usecase"
	salesdel "github.com/goerp/goerp/internal/sales/delivery"
	salesrepo "github.com/goerp/goerp/internal/sales/repository"
	salesuc "github.com/goerp/goerp/internal/sales/usecase"
	purchdel "github.com/goerp/goerp/internal/purchases/delivery"
	purchrepo "github.com/goerp/goerp/internal/purchases/repository"
	purchuc "github.com/goerp/goerp/internal/purchases/usecase"
	accdel "github.com/goerp/goerp/internal/accounting/delivery"
	accrepo "github.com/goerp/goerp/internal/accounting/repository"
	accuc "github.com/goerp/goerp/internal/accounting/usecase"
	crmdel "github.com/goerp/goerp/internal/crm/delivery"
	crmrepo "github.com/goerp/goerp/internal/crm/repository"
	crmuc "github.com/goerp/goerp/internal/crm/usecase"
	hrdel "github.com/goerp/goerp/internal/hr/delivery"
	hrrepo "github.com/goerp/goerp/internal/hr/repository"
	hruc "github.com/goerp/goerp/internal/hr/usecase"
	"github.com/goerp/goerp/internal/shared/database"
	"github.com/goerp/goerp/internal/shared/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberlog "github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// ── Load Config ──────────────────────────────────────────────────────────
	cfg, err := configs.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// ── Database ─────────────────────────────────────────────────────────────
	db, err := database.Connect(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	// ── Fiber App ─────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name + " v" + cfg.App.Version,
		ErrorHandler: errorHandler,
		BodyLimit:    10 * 1024 * 1024, // 10 MB
	})

	// ── Global Middleware ─────────────────────────────────────────────────────
	app.Use(recover.New())
	app.Use(fiberlog.New(fiberlog.Config{
		Format: "[${time}] ${status} ${method} ${path} ${latency}\n",
	}))
	app.Use(middleware.CORSMiddleware())
	app.Use(compress.New())

	// ── Static Files ─────────────────────────────────────────────────────────
	app.Static("/static", "./web/static", fiber.Static{
		Compress:  true,
		ByteRange: true,
	})
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("./web/static/favicon.ico")
	})

	// Serve the SPA for all non-API routes
	app.Get("/", serveApp)
	app.Get("/dashboard", serveApp)
	app.Get("/inventory/*", serveApp)
	app.Get("/sales/*", serveApp)
	app.Get("/purchases/*", serveApp)
	app.Get("/accounting/*", serveApp)
	app.Get("/crm/*", serveApp)
	app.Get("/hr/*", serveApp)
	app.Get("/reports/*", serveApp)
	app.Get("/settings/*", serveApp)
	app.Get("/ai/*", serveApp)
	app.Get("/workflows/*", serveApp)

	// ── Health ────────────────────────────────────────────────────────────────
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"version": cfg.App.Version,
			"app":     cfg.App.Name,
		})
	})

	// ── Wire up modules ───────────────────────────────────────────────────────
	wireModules(app, db, cfg)

	// ── Start ─────────────────────────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("GoERP v%s starting on http://%s", cfg.App.Version, addr)
	log.Printf("Default login: admin@goerp.io / admin123")

	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down GoERP...")
	_ = app.Shutdown()
}

func wireModules(app *fiber.App, db *database.DB, cfg *configs.Config) {
	secret := cfg.JWT.Secret
	ttl := cfg.JWT.AccessTokenTTL

	// Auth
	{
		userRepo := authrepo.NewUserRepository(db)
		tenantRepo := authrepo.NewTenantRepository(db)
		uc := authuc.NewAuthUsecase(userRepo, tenantRepo, secret, ttl)
		h := authdel.NewAuthHandler(uc)
		h.RegisterRoutes(app, secret)
	}

	// Protected routes
	auth := middleware.AuthMiddleware(secret)

	// Inventory
	{
		repo := invrepo.NewInventoryRepository(db)
		uc := invuc.NewInventoryUsecase(repo)
		h := invdel.NewInventoryHandler(uc)
		h.RegisterRoutes(app, auth)
	}

	// Sales
	{
		repo := salesrepo.NewSalesRepository(db)
		uc := salesuc.NewSalesUsecase(repo)
		h := salesdel.NewSalesHandler(uc)
		h.RegisterRoutes(app, auth)
	}

	// Purchases
	{
		repo := purchrepo.NewPurchasesRepository(db)
		uc := purchuc.NewPurchasesUsecase(repo)
		h := purchdel.NewPurchasesHandler(uc)
		h.RegisterRoutes(app, auth)
	}

	// Accounting
	{
		repo := accrepo.NewAccountingRepository(db)
		uc := accuc.NewAccountingUsecase(repo)
		h := accdel.NewAccountingHandler(uc)
		h.RegisterRoutes(app, auth)
	}

	// CRM - now uses *database.DB directly (not *sql.DB)
	{
		repo := crmrepo.NewCRMRepository(db)
		uc := crmuc.NewCRMUsecase(repo)
		h := crmdel.NewCRMHandler(uc)
		h.RegisterRoutes(app, auth)
	}

	// HR - now uses *database.DB directly (not *sql.DB)
	{
		repo := hrrepo.NewHRRepository(db)
		uc := hruc.NewHRUsecase(repo)
		h := hrdel.NewHRHandler(uc)
		h.RegisterRoutes(app, auth)
	}

	// Dashboard API
	app.Get("/api/v1/dashboard", auth, dashboardHandler(db))

	// Reports
	registerReportsRoutes(app, db, auth)

	// Settings
	registerSettingsRoutes(app, db, auth)

	// AI
	registerAIRoutes(app, db, auth)

	// Workflow
	registerWorkflowRoutes(app, db, auth)

	// Notifications
	registerNotificationRoutes(app, db, auth)
}

func serveApp(c *fiber.Ctx) error {
	return c.SendFile("./web/index.html")
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
		"code":  code,
	})
}

func dashboardHandler(db *database.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenantID := middleware.GetTenantID(c)
		if tenantID == "" {
			return c.Status(403).JSON(fiber.Map{"error": "no tenant"})
		}

		stats := fiber.Map{
			"revenue_this_month": 0,
			"orders_this_month":  0,
			"invoices_pending":   0,
			"stock_alerts":       0,
			"customers_total":    0,
			"suppliers_total":    0,
			"employees_total":    0,
			"open_opportunities": 0,
			"recent_sales":       []interface{}{},
			"recent_invoices":    []interface{}{},
			"revenue_chart":      []interface{}{},
			"top_products":       []interface{}{},
		}

		// Revenue this month (SQLite: strftime)
		var revenueMonth float64
		_ = db.QueryRow(`
			SELECT COALESCE(SUM(total),0) FROM invoices
			WHERE tenant_id=? AND state='paid'
			AND strftime('%Y-%m', invoice_date) = strftime('%Y-%m', 'now')
		`, tenantID).Scan(&revenueMonth)

		var ordersMonth int
		_ = db.QueryRow(`
			SELECT COUNT(*) FROM sales_orders
			WHERE tenant_id=? AND state NOT IN ('cancelled')
			AND strftime('%Y-%m', order_date) = strftime('%Y-%m', 'now')
		`, tenantID).Scan(&ordersMonth)

		var invoicesPending int
		_ = db.QueryRow(`SELECT COUNT(*) FROM invoices WHERE tenant_id=? AND state='pending'`, tenantID).Scan(&invoicesPending)

		var customersTotal int
		_ = db.QueryRow(`SELECT COUNT(*) FROM customers WHERE tenant_id=?`, tenantID).Scan(&customersTotal)

		var employeesTotal int
		_ = db.QueryRow(`SELECT COUNT(*) FROM employees WHERE tenant_id=? AND is_active=1`, tenantID).Scan(&employeesTotal)

		var openOpps int
		_ = db.QueryRow(`SELECT COUNT(*) FROM crm_opportunities WHERE tenant_id=? AND stage NOT IN ('won','lost')`, tenantID).Scan(&openOpps)

		var stockAlerts int
		_ = db.QueryRow(`
			SELECT COUNT(*) FROM products p
			WHERE p.tenant_id=? AND p.reorder_point > 0
			AND (
				SELECT COALESCE(
					SUM(CASE WHEN sm.to_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END) -
					SUM(CASE WHEN sm.from_location_id IS NOT NULL AND sm.state='done' THEN sm.quantity ELSE 0 END),
					0)
				FROM stock_moves sm WHERE sm.product_id = p.id
			) <= p.reorder_point
		`, tenantID).Scan(&stockAlerts)

		stats["revenue_this_month"] = revenueMonth
		stats["orders_this_month"] = ordersMonth
		stats["invoices_pending"] = invoicesPending
		stats["customers_total"] = customersTotal
		stats["employees_total"] = employeesTotal
		stats["open_opportunities"] = openOpps
		stats["stock_alerts"] = stockAlerts

		return c.JSON(stats)
	}
}
