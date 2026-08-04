package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps sql.DB with convenience methods
type DB struct {
	*sql.DB
}

// Connect opens a SQLite database. dsn must be a modernc.org/sqlite DSN,
// e.g. "file:./goerp.db?_journal=WAL&_timeout=5000&_foreign_keys=on"
func Connect(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}

	// Apply SQLite pragmas for production-grade performance
	pragmas := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous  = NORMAL;`,
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA cache_size   = -65536;`, // 64 MB page cache
		`PRAGMA temp_store   = MEMORY;`,
		`PRAGMA mmap_size    = 268435456;`, // 256 MB mmap
		`PRAGMA busy_timeout = 5000;`,
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			log.Printf("[DB] pragma warning (%s): %v", p, err)
		}
	}

	// Single writer is safe for SQLite WAL; pool of readers ok
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	log.Println("[DB] SQLite connected successfully")
	return &DB{db}, nil
}

// AutoMigrate creates all tables then seeds default data.
// Each statement is idempotent (IF NOT EXISTS / INSERT OR IGNORE).
func (d *DB) AutoMigrate() error {
	stmts := []string{
		ddlTenants,
		ddlUsers,
		ddlProductCategories,
		ddlProducts,
		ddlProductVariants,
		ddlWarehouses,
		ddlStockLocations,
		ddlStockMoves,
		ddlBatches,
		ddlCustomers,
		ddlSalesOrders,
		ddlSalesOrderLines,
		ddlInvoices,
		ddlInvoiceLines,
		ddlSuppliers,
		ddlPurchaseOrders,
		ddlEmployees,
		ddlContracts,
		ddlLeaveRequests,
		ddlAttendance,
		ddlPayrollRuns,
		ddlNotifications,
		ddlCRMLeads,
		ddlCRMOpportunities,
		ddlCRMActivities,
		seedData,
	}
	for _, s := range stmts {
		if _, err := d.Exec(s); err != nil {
			return fmt.Errorf("migration: %w  [prefix: %.100s]", err, s)
		}
	}
	log.Println("[DB] SQLite schema migrated and seed data applied")
	return nil
}

// ─── DDL ──────────────────────────────────────────────────────────────────────

const ddlTenants = `
CREATE TABLE IF NOT EXISTS tenants (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    slug       TEXT UNIQUE NOT NULL,
    plan       TEXT DEFAULT 'community',
    is_active  INTEGER DEFAULT 1,
    settings   TEXT DEFAULT '{}',
    created_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);`

const ddlUsers = `
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    tenant_id     TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    avatar        TEXT DEFAULT '',
    is_active     INTEGER DEFAULT 1,
    is_superadmin INTEGER DEFAULT 0,
    last_login    TEXT,
    created_at    TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at    TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(tenant_id, email)
);
CREATE INDEX IF NOT EXISTS idx_users_email  ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);`

const ddlProductCategories = `
CREATE TABLE IF NOT EXISTS product_categories (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    parent_id   TEXT REFERENCES product_categories(id),
    description TEXT DEFAULT '',
    created_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_cats_tenant ON product_categories(tenant_id);`

const ddlProducts = `
CREATE TABLE IF NOT EXISTS products (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    sku             TEXT NOT NULL,
    name            TEXT NOT NULL DEFAULT '{"en":""}',
    description     TEXT DEFAULT '',
    category_id     TEXT REFERENCES product_categories(id),
    unit_of_measure TEXT DEFAULT 'unit',
    base_price      REAL DEFAULT 0,
    cost_price      REAL DEFAULT 0,
    sale_price      REAL DEFAULT 0,
    tax_rate        REAL DEFAULT 0,
    barcode         TEXT DEFAULT '',
    qr_code         TEXT DEFAULT '',
    track_inventory INTEGER DEFAULT 1,
    track_batch     INTEGER DEFAULT 0,
    track_serial    INTEGER DEFAULT 0,
    has_expiry      INTEGER DEFAULT 0,
    min_stock_level REAL DEFAULT 0,
    reorder_point   REAL DEFAULT 0,
    is_active       INTEGER DEFAULT 1,
    image_url       TEXT DEFAULT '',
    tags            TEXT DEFAULT '[]',
    created_at      TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at      TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(tenant_id, sku)
);
CREATE INDEX IF NOT EXISTS idx_products_tenant  ON products(tenant_id);
CREATE INDEX IF NOT EXISTS idx_products_sku     ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_barcode ON products(barcode);
CREATE INDEX IF NOT EXISTS idx_products_cat     ON products(category_id);`

const ddlProductVariants = `
CREATE TABLE IF NOT EXISTS product_variants (
    id             TEXT PRIMARY KEY,
    product_id     TEXT REFERENCES products(id) ON DELETE CASCADE,
    sku            TEXT NOT NULL,
    attributes     TEXT DEFAULT '{}',
    price_modifier REAL DEFAULT 0,
    stock_qty      REAL DEFAULT 0,
    barcode        TEXT DEFAULT '',
    is_active      INTEGER DEFAULT 1,
    created_at     TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_variants_product ON product_variants(product_id);`

const ddlWarehouses = `
CREATE TABLE IF NOT EXISTS warehouses (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    code       TEXT NOT NULL,
    address    TEXT DEFAULT '',
    city       TEXT DEFAULT '',
    country    TEXT DEFAULT '',
    manager_id TEXT DEFAULT '',
    is_active  INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(tenant_id, code)
);
CREATE INDEX IF NOT EXISTS idx_warehouses_tenant ON warehouses(tenant_id);`

const ddlStockLocations = `
CREATE TABLE IF NOT EXISTS stock_locations (
    id            TEXT PRIMARY KEY,
    warehouse_id  TEXT REFERENCES warehouses(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    code          TEXT DEFAULT '',
    location_type TEXT DEFAULT 'internal',
    parent_id     TEXT REFERENCES stock_locations(id),
    created_at    TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_locs_warehouse ON stock_locations(warehouse_id);`

const ddlStockMoves = `
CREATE TABLE IF NOT EXISTS stock_moves (
    id               TEXT PRIMARY KEY,
    tenant_id        TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    product_id       TEXT REFERENCES products(id),
    variant_id       TEXT REFERENCES product_variants(id),
    from_location_id TEXT REFERENCES stock_locations(id),
    to_location_id   TEXT REFERENCES stock_locations(id),
    quantity         REAL NOT NULL DEFAULT 0,
    unit_cost        REAL DEFAULT 0,
    batch_number     TEXT DEFAULT '',
    serial_number    TEXT DEFAULT '',
    expiry_date      TEXT DEFAULT '',
    move_type        TEXT NOT NULL DEFAULT 'in',
    reference        TEXT DEFAULT '',
    state            TEXT DEFAULT 'draft',
    notes            TEXT DEFAULT '',
    created_by       TEXT DEFAULT '',
    created_at       TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    done_at          TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_moves_product ON stock_moves(product_id, state);
CREATE INDEX IF NOT EXISTS idx_moves_tenant  ON stock_moves(tenant_id);
CREATE INDEX IF NOT EXISTS idx_moves_type    ON stock_moves(move_type);
CREATE INDEX IF NOT EXISTS idx_moves_created ON stock_moves(created_at);`

const ddlBatches = `
CREATE TABLE IF NOT EXISTS batches (
    id               TEXT PRIMARY KEY,
    tenant_id        TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    product_id       TEXT REFERENCES products(id),
    batch_number     TEXT NOT NULL,
    manufacture_date TEXT DEFAULT '',
    expiry_date      TEXT DEFAULT '',
    qty_received     REAL DEFAULT 0,
    qty_remaining    REAL DEFAULT 0,
    supplier_id      TEXT DEFAULT '',
    created_at       TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_batches_product ON batches(product_id);
CREATE INDEX IF NOT EXISTS idx_batches_tenant  ON batches(tenant_id);`

const ddlCustomers = `
CREATE TABLE IF NOT EXISTS customers (
    id           TEXT PRIMARY KEY,
    tenant_id    TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    email        TEXT DEFAULT '',
    phone        TEXT DEFAULT '',
    company_name TEXT DEFAULT '',
    address      TEXT DEFAULT '',
    city         TEXT DEFAULT '',
    country      TEXT DEFAULT '',
    credit_limit REAL DEFAULT 0,
    currency     TEXT DEFAULT 'USD',
    is_active    INTEGER DEFAULT 1,
    notes        TEXT DEFAULT '',
    created_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_customers_tenant ON customers(tenant_id);`

const ddlSalesOrders = `
CREATE TABLE IF NOT EXISTS sales_orders (
    id           TEXT PRIMARY KEY,
    tenant_id    TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    order_number TEXT NOT NULL,
    customer_id  TEXT REFERENCES customers(id),
    state        TEXT DEFAULT 'draft',
    order_date   TEXT DEFAULT (date('now')),
    delivery_date TEXT DEFAULT '',
    subtotal     REAL DEFAULT 0,
    tax_amount   REAL DEFAULT 0,
    total        REAL DEFAULT 0,
    notes        TEXT DEFAULT '',
    created_by   TEXT DEFAULT '',
    created_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(tenant_id, order_number)
);
CREATE INDEX IF NOT EXISTS idx_orders_tenant   ON sales_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_orders_customer ON sales_orders(customer_id);`

const ddlSalesOrderLines = `
CREATE TABLE IF NOT EXISTS sales_order_lines (
    id         TEXT PRIMARY KEY,
    order_id   TEXT REFERENCES sales_orders(id) ON DELETE CASCADE,
    product_id TEXT REFERENCES products(id),
    quantity   REAL NOT NULL DEFAULT 1,
    unit_price REAL NOT NULL DEFAULT 0,
    discount   REAL DEFAULT 0,
    tax_rate   REAL DEFAULT 0,
    subtotal   REAL DEFAULT 0,
    total      REAL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_order_lines_order ON sales_order_lines(order_id);`

const ddlInvoices = `
CREATE TABLE IF NOT EXISTS invoices (
    id             TEXT PRIMARY KEY,
    tenant_id      TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_number TEXT NOT NULL,
    customer_id    TEXT REFERENCES customers(id),
    order_id       TEXT REFERENCES sales_orders(id),
    state          TEXT DEFAULT 'draft',
    invoice_date   TEXT DEFAULT (date('now')),
    due_date       TEXT DEFAULT '',
    subtotal       REAL DEFAULT 0,
    tax_amount     REAL DEFAULT 0,
    total          REAL DEFAULT 0,
    amount_paid    REAL DEFAULT 0,
    amount_due     REAL DEFAULT 0,
    currency       TEXT DEFAULT 'USD',
    notes          TEXT DEFAULT '',
    created_by     TEXT DEFAULT '',
    created_at     TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at     TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(tenant_id, invoice_number)
);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant   ON invoices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_invoices_customer ON invoices(customer_id);`

const ddlInvoiceLines = `
CREATE TABLE IF NOT EXISTS invoice_lines (
    id         TEXT PRIMARY KEY,
    invoice_id TEXT REFERENCES invoices(id) ON DELETE CASCADE,
    product_id TEXT REFERENCES products(id),
    quantity   REAL NOT NULL DEFAULT 1,
    unit_price REAL NOT NULL DEFAULT 0,
    discount   REAL DEFAULT 0,
    tax_rate   REAL DEFAULT 0,
    subtotal   REAL DEFAULT 0,
    total      REAL DEFAULT 0
);`

const ddlSuppliers = `
CREATE TABLE IF NOT EXISTS suppliers (
    id           TEXT PRIMARY KEY,
    tenant_id    TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    email        TEXT DEFAULT '',
    phone        TEXT DEFAULT '',
    company_name TEXT DEFAULT '',
    address      TEXT DEFAULT '',
    city         TEXT DEFAULT '',
    country      TEXT DEFAULT '',
    currency     TEXT DEFAULT 'USD',
    payment_terms INTEGER DEFAULT 30,
    is_active    INTEGER DEFAULT 1,
    notes        TEXT DEFAULT '',
    created_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at   TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_suppliers_tenant ON suppliers(tenant_id);`

const ddlPurchaseOrders = `
CREATE TABLE IF NOT EXISTS purchase_orders (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    po_number   TEXT NOT NULL,
    supplier_id TEXT REFERENCES suppliers(id),
    state       TEXT DEFAULT 'draft',
    order_date  TEXT DEFAULT (date('now')),
    expected_date TEXT DEFAULT '',
    subtotal    REAL DEFAULT 0,
    tax_amount  REAL DEFAULT 0,
    total       REAL DEFAULT 0,
    notes       TEXT DEFAULT '',
    created_by  TEXT DEFAULT '',
    created_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(tenant_id, po_number)
);
CREATE INDEX IF NOT EXISTS idx_po_tenant   ON purchase_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_po_supplier ON purchase_orders(supplier_id);`

const ddlEmployees = `
CREATE TABLE IF NOT EXISTS employees (
    id                TEXT PRIMARY KEY,
    tenant_id         TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    employee_number   TEXT DEFAULT '',
    first_name        TEXT NOT NULL,
    last_name         TEXT NOT NULL,
    email             TEXT DEFAULT '',
    phone             TEXT DEFAULT '',
    department        TEXT DEFAULT '',
    position          TEXT DEFAULT '',
    manager_id        TEXT DEFAULT '',
    hire_date         TEXT DEFAULT (date('now')),
    national_id       TEXT DEFAULT '',
    address           TEXT DEFAULT '',
    emergency_contact TEXT DEFAULT '',
    user_id           TEXT DEFAULT '',
    is_active         INTEGER DEFAULT 1,
    created_at        TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at        TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_employees_tenant ON employees(tenant_id);`

const ddlContracts = `
CREATE TABLE IF NOT EXISTS contracts (
    id                  TEXT PRIMARY KEY,
    tenant_id           TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id         TEXT REFERENCES employees(id),
    contract_type       TEXT NOT NULL DEFAULT 'permanent',
    start_date          TEXT NOT NULL,
    end_date            TEXT DEFAULT '',
    basic_salary        REAL NOT NULL DEFAULT 0,
    housing_allowance   REAL DEFAULT 0,
    transport_allowance REAL DEFAULT 0,
    other_allowances    REAL DEFAULT 0,
    currency            TEXT DEFAULT 'USD',
    is_active           INTEGER DEFAULT 1,
    notes               TEXT DEFAULT '',
    created_at          TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_contracts_employee ON contracts(employee_id);`

const ddlLeaveRequests = `
CREATE TABLE IF NOT EXISTS leave_requests (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id TEXT REFERENCES employees(id),
    leave_type  TEXT NOT NULL DEFAULT 'annual',
    start_date  TEXT NOT NULL,
    end_date    TEXT NOT NULL,
    days_count  INTEGER DEFAULT 0,
    state       TEXT DEFAULT 'draft',
    reason      TEXT DEFAULT '',
    approved_by TEXT DEFAULT '',
    approved_at TEXT DEFAULT '',
    created_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_leaves_employee ON leave_requests(employee_id);
CREATE INDEX IF NOT EXISTS idx_leaves_tenant   ON leave_requests(tenant_id);`

const ddlAttendance = `
CREATE TABLE IF NOT EXISTS attendance (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    employee_id TEXT REFERENCES employees(id),
    check_in   TEXT,
    check_out  TEXT DEFAULT '',
    work_hours REAL DEFAULT 0,
    status     TEXT DEFAULT 'present',
    notes      TEXT DEFAULT '',
    created_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_attend_employee ON attendance(employee_id);
CREATE INDEX IF NOT EXISTS idx_attend_tenant   ON attendance(tenant_id);`

const ddlPayrollRuns = `
CREATE TABLE IF NOT EXISTS payroll_runs (
    id               TEXT PRIMARY KEY,
    tenant_id        TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    period_start     TEXT NOT NULL,
    period_end       TEXT NOT NULL,
    state            TEXT DEFAULT 'draft',
    total_gross      REAL DEFAULT 0,
    total_deductions REAL DEFAULT 0,
    total_net        REAL DEFAULT 0,
    employee_count   INTEGER DEFAULT 0,
    processed_at     TEXT DEFAULT '',
    processed_by     TEXT DEFAULT '',
    notes            TEXT DEFAULT '',
    created_at       TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_payroll_tenant ON payroll_runs(tenant_id);`

const ddlNotifications = `
CREATE TABLE IF NOT EXISTS notifications (
    id                TEXT PRIMARY KEY,
    tenant_id         TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    user_id           TEXT REFERENCES users(id),
    title             TEXT NOT NULL,
    message           TEXT NOT NULL,
    notification_type TEXT DEFAULT 'info',
    is_read           INTEGER DEFAULT 0,
    created_at        TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_notif_user   ON notifications(user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_notif_tenant ON notifications(tenant_id);`

const ddlCRMLeads = `
CREATE TABLE IF NOT EXISTS crm_leads (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    email       TEXT DEFAULT '',
    phone       TEXT DEFAULT '',
    company     TEXT DEFAULT '',
    source      TEXT DEFAULT '',
    status      TEXT DEFAULT 'new',
    assigned_to TEXT DEFAULT '',
    notes       TEXT DEFAULT '',
    created_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_leads_tenant ON crm_leads(tenant_id);`

const ddlCRMOpportunities = `
CREATE TABLE IF NOT EXISTS crm_opportunities (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    customer_id     TEXT REFERENCES customers(id),
    stage           TEXT DEFAULT 'prospecting',
    probability     REAL DEFAULT 0,
    expected_revenue REAL DEFAULT 0,
    close_date      TEXT DEFAULT '',
    assigned_to     TEXT DEFAULT '',
    notes           TEXT DEFAULT '',
    is_won          INTEGER DEFAULT 0,
    is_lost         INTEGER DEFAULT 0,
    created_at      TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at      TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_opp_tenant ON crm_opportunities(tenant_id);`

const ddlCRMActivities = `
CREATE TABLE IF NOT EXISTS crm_activities (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT REFERENCES tenants(id) ON DELETE CASCADE,
    activity_type   TEXT DEFAULT 'call',
    title           TEXT NOT NULL,
    description     TEXT DEFAULT '',
    lead_id         TEXT REFERENCES crm_leads(id),
    opportunity_id  TEXT REFERENCES crm_opportunities(id),
    customer_id     TEXT REFERENCES customers(id),
    assigned_to     TEXT DEFAULT '',
    due_date        TEXT DEFAULT '',
    is_done         INTEGER DEFAULT 0,
    created_at      TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_activities_tenant ON crm_activities(tenant_id);`

// ─── Seed Data ────────────────────────────────────────────────────────────────
// Production seed: 1 tenant, 1 admin user, 2 warehouses, 9 locations,
// 5 categories, 18 products, 34 stock moves (17 IN + 17 OUT giving real qty),
// 3 batches, 2 variants, 3 customers, 3 suppliers.
// Password: admin@goerp.io / admin123 (bcrypt cost-12 hash)

const seedData = `
-- ── Tenant ────────────────────────────────────────────────────────────────────
INSERT OR IGNORE INTO tenants (id, name, slug, plan, is_active)
VALUES ('00000000-0000-0000-0000-000000000001','Demo Company','demo','enterprise',1);

-- ── Admin user (password = admin123, bcrypt cost-12) ─────────────────────────
INSERT OR IGNORE INTO users (id, tenant_id, email, password_hash, full_name, is_superadmin, is_active)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'admin@goerp.io',
    '$2a$12$3E.cxLde9ca4ZhEJdbv/YOzhTbkifweD8ibFTVTv7fq.hESMaoHAS',
    'System Administrator',
    1, 1
);

-- ── Warehouses ────────────────────────────────────────────────────────────────
INSERT OR IGNORE INTO warehouses (id, tenant_id, name, code, address, city, country, is_active)
VALUES
    ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000001',
     'Main Warehouse','WH-001','123 Industrial Ave','New York','US',1),
    ('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001',
     'West Coast DC','WH-002','456 Logistics Blvd','Los Angeles','US',1);

-- ── Stock Locations ───────────────────────────────────────────────────────────
INSERT OR IGNORE INTO stock_locations (id, warehouse_id, name, code, location_type)
VALUES
    ('00000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000001','Zone A - Rack 1','A-R1','internal'),
    ('00000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000001','Zone A - Rack 2','A-R2','internal'),
    ('00000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000001','Zone B - Bulk Storage','B-BULK','internal'),
    ('00000000-0000-0000-0000-000000000013','00000000-0000-0000-0000-000000000001','Receiving Dock','RECV','internal'),
    ('00000000-0000-0000-0000-000000000014','00000000-0000-0000-0000-000000000001','Dispatch Dock','DISP','internal'),
    ('00000000-0000-0000-0000-000000000015','00000000-0000-0000-0000-000000000002','Shelf C1','C1','internal'),
    ('00000000-0000-0000-0000-000000000016','00000000-0000-0000-0000-000000000002','Shelf C2','C2','internal'),
    ('00000000-0000-0000-0000-000000000017','00000000-0000-0000-0000-000000000002','Shelf C3','C3','internal'),
    ('00000000-0000-0000-0000-000000000018','00000000-0000-0000-0000-000000000002','West Receiving','W-RECV','internal');

-- ── Product Categories (5) ────────────────────────────────────────────────────
INSERT OR IGNORE INTO product_categories (id, tenant_id, name, description)
VALUES
    ('00000000-0000-0000-0001-000000000001','00000000-0000-0000-0000-000000000001',
     'Electronics','Computers, monitors, storage devices and electronic components'),
    ('00000000-0000-0000-0001-000000000002','00000000-0000-0000-0000-000000000001',
     'Accessories','Cables, adapters, peripherals and add-on accessories'),
    ('00000000-0000-0000-0001-000000000003','00000000-0000-0000-0000-000000000001',
     'Office Supplies','Stationery, paper, consumables and office essentials'),
    ('00000000-0000-0000-0001-000000000004','00000000-0000-0000-0000-000000000001',
     'Networking','Switches, routers, access points and networking equipment'),
    ('00000000-0000-0000-0001-000000000005','00000000-0000-0000-0000-000000000001',
     'Software','Software licenses, subscriptions and digital products');

-- ── Products (18 items) ───────────────────────────────────────────────────────
INSERT OR IGNORE INTO products
    (id,tenant_id,sku,name,description,category_id,unit_of_measure,
     base_price,cost_price,sale_price,tax_rate,barcode,
     track_inventory,track_batch,track_serial,has_expiry,
     min_stock_level,reorder_point,is_active)
VALUES
-- Electronics
('00000000-0000-0000-0002-000000000001','00000000-0000-0000-0000-000000000001',
 'LAPTOP-PRO-15','{"en":"Laptop Pro 15"}',
 'Professional 15-inch laptop, Intel Core i7-1355U, 16GB RAM, 512GB NVMe SSD',
 '00000000-0000-0000-0001-000000000001','unit',
 1299.99,950.00,1299.99,20,'8901234567890',1,1,1,0,5,10,1),

('00000000-0000-0000-0002-000000000002','00000000-0000-0000-0000-000000000001',
 'LAPTOP-ULTRA-14','{"en":"UltraBook 14 Pro"}',
 'Premium ultrabook 14-inch, AMD Ryzen 7 7840U, 32GB RAM, 1TB NVMe',
 '00000000-0000-0000-0001-000000000001','unit',
 1599.99,1150.00,1599.99,20,'8901234567891',1,0,1,0,3,6,1),

('00000000-0000-0000-0002-000000000003','00000000-0000-0000-0000-000000000001',
 'MON-27-4K','{"en":"4K Monitor 27 Inch"}',
 '27-inch 4K IPS display, 144Hz refresh rate, HDR400, USB-C 65W PD',
 '00000000-0000-0000-0001-000000000001','unit',
 599.99,380.00,599.99,20,'8901234567892',1,0,0,0,3,5,1),

('00000000-0000-0000-0002-000000000004','00000000-0000-0000-0000-000000000001',
 'MON-34-UWQHD','{"en":"Ultrawide Monitor 34 Inch"}',
 '34-inch UWQHD IPS curved, 165Hz, 1ms, G-Sync compatible',
 '00000000-0000-0000-0001-000000000001','unit',
 749.99,490.00,749.99,20,'8901234567893',1,0,0,0,2,4,1),

('00000000-0000-0000-0002-000000000005','00000000-0000-0000-0000-000000000001',
 'SSD-1TB-NVMe','{"en":"NVMe SSD 1TB"}',
 'PCIe 4.0 NVMe M.2 SSD, 7400MB/s read, 6900MB/s write, 5-year warranty',
 '00000000-0000-0000-0001-000000000001','unit',
 99.99,58.00,99.99,20,'8901234567894',1,0,1,0,15,25,1),

('00000000-0000-0000-0002-000000000006','00000000-0000-0000-0000-000000000001',
 'RAM-DDR5-32GB','{"en":"DDR5 RAM 32GB Kit"}',
 '32GB (2x16GB) DDR5-6000 CL30, XMP 3.0, aluminum heat spreader',
 '00000000-0000-0000-0001-000000000001','unit',
 129.99,78.00,129.99,20,'8901234567895',1,0,0,0,8,12,1),

('00000000-0000-0000-0002-000000000007','00000000-0000-0000-0000-000000000001',
 'KBRD-MECH-TKL','{"en":"Mechanical Keyboard TKL"}',
 'Tenkeyless mechanical keyboard, Cherry MX Red switches, RGB backlit',
 '00000000-0000-0000-0001-000000000001','unit',
 89.99,45.00,89.99,20,'8901234567896',1,0,0,0,10,15,1),

('00000000-0000-0000-0002-000000000008','00000000-0000-0000-0000-000000000001',
 'MOUSE-WL-PRO','{"en":"Wireless Mouse Pro X200"}',
 'Ergonomic wireless mouse, 25600 DPI, 100-hour battery, USB-A and USB-C',
 '00000000-0000-0000-0001-000000000001','unit',
 49.99,22.00,49.99,20,'8901234567897',1,0,0,0,20,35,1),

-- Accessories
('00000000-0000-0000-0002-000000000009','00000000-0000-0000-0000-000000000001',
 'HUB-USBC-9P','{"en":"USB-C Hub 9-Port"}',
 '9-in-1 USB-C hub: 4K HDMI x2, DP, 3xUSB-A 3.2, USB-C PD 100W, SD/microSD',
 '00000000-0000-0000-0001-000000000002','unit',
 69.99,31.00,69.99,20,'8901234567898',1,0,0,0,12,20,1),

('00000000-0000-0000-0002-000000000010','00000000-0000-0000-0000-000000000001',
 'CABLE-USBC-2M','{"en":"USB-C Cable 2m 240W"}',
 'USB 4 certified cable, 240W charging, 40Gbps data, 8K video pass-through',
 '00000000-0000-0000-0001-000000000002','unit',
 19.99,6.50,19.99,20,'8901234567899',1,0,0,0,50,100,1),

('00000000-0000-0000-0002-000000000011','00000000-0000-0000-0000-000000000001',
 'DOCKING-TBT4','{"en":"Thunderbolt 4 Docking Station"}',
 'Thunderbolt 4 dock, 96W host charge, 2x4K display, 3xUSB-A, 2xUSB-C, SD, 2.5G LAN',
 '00000000-0000-0000-0001-000000000002','unit',
 299.99,175.00,299.99,20,'8901234567900',1,0,0,0,5,8,1),

('00000000-0000-0000-0002-000000000012','00000000-0000-0000-0000-000000000001',
 'STAND-LAPTOP-ADJ','{"en":"Adjustable Laptop Stand"}',
 'Aluminum laptop stand, 6 angle settings, foldable, fits 10-17 inch laptops',
 '00000000-0000-0000-0001-000000000002','unit',
 34.99,12.00,34.99,20,'8901234567901',1,0,0,0,25,40,1),

-- Office Supplies
('00000000-0000-0000-0002-000000000013','00000000-0000-0000-0000-000000000001',
 'PAPER-A4-REAM','{"en":"A4 Copy Paper 80gsm Ream"}',
 'A4 white copy paper, 80gsm, 500 sheets per ream, acid-free, CIE 161 brightness',
 '00000000-0000-0000-0001-000000000003','ream',
 8.99,3.20,8.99,10,'8901234567902',1,0,0,0,100,200,1),

('00000000-0000-0000-0002-000000000014','00000000-0000-0000-0000-000000000001',
 'TONER-HP-BLACK','{"en":"HP LaserJet Black Toner CF217A"}',
 'HP 17A black laser toner cartridge, 1600-page yield, compatible with M102/M130',
 '00000000-0000-0000-0001-000000000003','unit',
 44.99,18.00,44.99,10,'8901234567903',1,1,0,1,10,20,1),

('00000000-0000-0000-0002-000000000015','00000000-0000-0000-0000-000000000001',
 'CHAIR-ERGO-PRO','{"en":"Ergonomic Office Chair Pro"}',
 'Ergonomic mesh office chair, lumbar support, adjustable armrests, 5-year warranty',
 '00000000-0000-0000-0001-000000000003','unit',
 399.99,195.00,399.99,20,'8901234567904',1,0,0,0,5,8,1),

-- Networking
('00000000-0000-0000-0002-000000000016','00000000-0000-0000-0000-000000000001',
 'SWITCH-24P-GBE','{"en":"24-Port Gigabit Switch Managed"}',
 '24-port managed gigabit switch, VLAN, QoS, SNMP, 4x SFP uplinks, rack-mountable',
 '00000000-0000-0000-0001-000000000004','unit',
 349.99,198.00,349.99,20,'8901234567905',1,0,1,0,3,5,1),

('00000000-0000-0000-0002-000000000017','00000000-0000-0000-0000-000000000001',
 'AP-WIFI6E-TRI','{"en":"Wi-Fi 6E Tri-Band Access Point"}',
 'Wi-Fi 6E access point, 6GHz+5GHz+2.4GHz, 7.8Gbps, 300+ clients, PoE++',
 '00000000-0000-0000-0001-000000000004','unit',
 299.99,168.00,299.99,20,'8901234567906',1,0,1,0,4,8,1),

-- Software (no physical stock)
('00000000-0000-0000-0002-000000000018','00000000-0000-0000-0000-000000000001',
 'SW-OFFICE365-1Y','{"en":"Microsoft 365 Business 1-Year"}',
 'Microsoft 365 Business Standard, 1 user, 1 year subscription, 1TB OneDrive',
 '00000000-0000-0000-0001-000000000005','license',
 149.99,92.00,149.99,20,'',0,0,0,0,0,0,1);

-- ── Stock Moves: 17 IN moves + 17 OUT moves = real on-hand quantities ─────────
-- in moves set to_location_id; out moves leave to_location_id NULL (from_location NULL for in).
-- Net qty per product: IN - OUT = current_stock
INSERT OR IGNORE INTO stock_moves
    (id,tenant_id,product_id,quantity,unit_cost,move_type,reference,state,notes,
     to_location_id,from_location_id,created_at,done_at)
VALUES
-- Laptop Pro 15: 50 in - 5 out = 45
('00000000-0000-0000-0003-000000000001','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000001',50,950.00,'in','PO-2026-0001','done','Initial purchase',
 '00000000-0000-0000-0000-000000000010',NULL,'2026-01-15T09:00:00Z','2026-01-15T09:00:00Z'),
('00000000-0000-0000-0003-000000000002','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000001',5,1299.99,'out','SO-2026-0041','done','Sale Acme Corp',
 NULL,'00000000-0000-0000-0000-000000000010','2026-02-10T14:00:00Z','2026-02-10T14:00:00Z'),

-- UltraBook 14 Pro: 20 in - 2 out = 18
('00000000-0000-0000-0003-000000000003','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000002',20,1150.00,'in','PO-2026-0002','done','Initial stock',
 '00000000-0000-0000-0000-000000000010',NULL,'2026-01-20T10:00:00Z','2026-01-20T10:00:00Z'),
('00000000-0000-0000-0003-000000000004','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000002',2,1599.99,'out','SO-2026-0055','done','Sale TechStart',
 NULL,'00000000-0000-0000-0000-000000000010','2026-02-18T11:00:00Z','2026-02-18T11:00:00Z'),

-- 4K Monitor 27: 15 in - 3 out = 12
('00000000-0000-0000-0003-000000000005','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000003',15,380.00,'in','PO-2026-0003','done','Q1 2026 purchase',
 '00000000-0000-0000-0000-000000000011',NULL,'2026-01-22T08:30:00Z','2026-01-22T08:30:00Z'),
('00000000-0000-0000-0003-000000000006','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000003',3,599.99,'out','SO-2026-0062','done','Sale Global Media',
 NULL,'00000000-0000-0000-0000-000000000011','2026-02-25T09:00:00Z','2026-02-25T09:00:00Z'),

-- Ultrawide Monitor 34: 10 in - 2 out = 8
('00000000-0000-0000-0003-000000000007','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000004',10,490.00,'in','PO-2026-0004','done','Ultrawide order',
 '00000000-0000-0000-0000-000000000011',NULL,'2026-01-28T13:00:00Z','2026-01-28T13:00:00Z'),
('00000000-0000-0000-0003-000000000008','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000004',2,749.99,'out','SO-2026-0078','done','Sale order',
 NULL,'00000000-0000-0000-0000-000000000011','2026-03-05T10:00:00Z','2026-03-05T10:00:00Z'),

-- NVMe SSD 1TB: 50 in - 8 out = 42
('00000000-0000-0000-0003-000000000009','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000005',50,58.00,'in','PO-2026-0005','done','SSD bulk order',
 '00000000-0000-0000-0000-000000000012',NULL,'2026-01-10T07:00:00Z','2026-01-10T07:00:00Z'),
('00000000-0000-0000-0003-000000000010','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000005',8,99.99,'out','SO-2026-0091','done','SSD sales batch',
 NULL,'00000000-0000-0000-0000-000000000012','2026-02-14T16:00:00Z','2026-02-14T16:00:00Z'),

-- DDR5 RAM 32GB: 15 in - 11 out = 4 (LOW STOCK - below reorder 12)
('00000000-0000-0000-0003-000000000011','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000006',15,78.00,'in','PO-2026-0006','done','RAM kit order',
 '00000000-0000-0000-0000-000000000012',NULL,'2026-01-12T09:00:00Z','2026-01-12T09:00:00Z'),
('00000000-0000-0000-0003-000000000012','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000006',11,129.99,'out','SO-2026-0099','done','RAM kits sold',
 NULL,'00000000-0000-0000-0000-000000000012','2026-02-28T14:00:00Z','2026-02-28T14:00:00Z'),

-- Mechanical Keyboard TKL: 40 in - 7 out = 33
('00000000-0000-0000-0003-000000000013','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000007',40,45.00,'in','PO-2026-0007','done','Keyboard order',
 '00000000-0000-0000-0000-000000000011',NULL,'2026-01-18T10:00:00Z','2026-01-18T10:00:00Z'),
('00000000-0000-0000-0003-000000000014','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000007',7,89.99,'out','SO-2026-0110','done','Keyboard sales',
 NULL,'00000000-0000-0000-0000-000000000011','2026-03-01T11:00:00Z','2026-03-01T11:00:00Z'),

-- Wireless Mouse Pro: 100 in - 12 out = 88
('00000000-0000-0000-0003-000000000015','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000008',100,22.00,'in','PO-2026-0008','done','Mouse bulk',
 '00000000-0000-0000-0000-000000000012',NULL,'2026-01-08T08:00:00Z','2026-01-08T08:00:00Z'),
('00000000-0000-0000-0003-000000000016','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000008',12,49.99,'out','SO-2026-0125','done','Mouse sales',
 NULL,'00000000-0000-0000-0000-000000000012','2026-02-20T15:00:00Z','2026-02-20T15:00:00Z'),

-- USB-C Hub 9-Port: 30 in - 3 out = 27
('00000000-0000-0000-0003-000000000017','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000009',30,31.00,'in','PO-2026-0009','done','Hub order',
 '00000000-0000-0000-0000-000000000015',NULL,'2026-01-25T09:00:00Z','2026-01-25T09:00:00Z'),
('00000000-0000-0000-0003-000000000018','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000009',3,69.99,'out','SO-2026-0140','done','Hub sales',
 NULL,'00000000-0000-0000-0000-000000000015','2026-03-10T10:00:00Z','2026-03-10T10:00:00Z'),

-- USB-C Cable 2m: 200 in - 55 out = 145
('00000000-0000-0000-0003-000000000019','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000010',200,6.50,'in','PO-2026-0010','done','Cable bulk stock',
 '00000000-0000-0000-0000-000000000012',NULL,'2026-01-05T07:00:00Z','2026-01-05T07:00:00Z'),
('00000000-0000-0000-0003-000000000020','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000010',55,19.99,'out','SO-2026-0155','done','Cable sales',
 NULL,'00000000-0000-0000-0000-000000000012','2026-02-15T09:00:00Z','2026-02-15T09:00:00Z'),

-- Thunderbolt 4 Dock: 18 in - 4 out = 14
('00000000-0000-0000-0003-000000000021','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000011',18,175.00,'in','PO-2026-0011','done','Dock order',
 '00000000-0000-0000-0000-000000000015',NULL,'2026-01-30T11:00:00Z','2026-01-30T11:00:00Z'),
('00000000-0000-0000-0003-000000000022','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000011',4,299.99,'out','SO-2026-0168','done','Dock sales',
 NULL,'00000000-0000-0000-0000-000000000015','2026-03-08T14:00:00Z','2026-03-08T14:00:00Z'),

-- Laptop Stand: 75 in - 13 out = 62
('00000000-0000-0000-0003-000000000023','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000012',75,12.00,'in','PO-2026-0012','done','Stand stock',
 '00000000-0000-0000-0000-000000000016',NULL,'2026-01-14T08:00:00Z','2026-01-14T08:00:00Z'),
('00000000-0000-0000-0003-000000000024','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000012',13,34.99,'out','SO-2026-0180','done','Stand sales',
 NULL,'00000000-0000-0000-0000-000000000016','2026-02-22T10:00:00Z','2026-02-22T10:00:00Z'),

-- A4 Paper Ream: 500 in - 120 out = 380
('00000000-0000-0000-0003-000000000025','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000013',500,3.20,'in','PO-2026-0013','done','Paper supply',
 '00000000-0000-0000-0000-000000000017',NULL,'2026-01-03T07:00:00Z','2026-01-03T07:00:00Z'),
('00000000-0000-0000-0003-000000000026','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000013',120,8.99,'out','SO-2026-0195','done','Paper to offices',
 NULL,'00000000-0000-0000-0000-000000000017','2026-02-05T09:00:00Z','2026-02-05T09:00:00Z'),

-- HP Toner: 30 in - 8 out = 22
('00000000-0000-0000-0003-000000000027','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000014',30,18.00,'in','PO-2026-0014','done','Toner order',
 '00000000-0000-0000-0000-000000000017',NULL,'2026-01-20T10:00:00Z','2026-01-20T10:00:00Z'),
('00000000-0000-0000-0003-000000000028','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000014',8,44.99,'out','SO-2026-0210','done','Toner sales',
 NULL,'00000000-0000-0000-0000-000000000017','2026-03-12T14:00:00Z','2026-03-12T14:00:00Z'),

-- Ergonomic Chair: 10 in - 3 out = 7
('00000000-0000-0000-0003-000000000029','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000015',10,195.00,'in','PO-2026-0015','done','Chair order',
 '00000000-0000-0000-0000-000000000018',NULL,'2026-01-26T09:00:00Z','2026-01-26T09:00:00Z'),
('00000000-0000-0000-0003-000000000030','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000015',3,399.99,'out','SO-2026-0225','done','Chair sales',
 NULL,'00000000-0000-0000-0000-000000000018','2026-02-28T16:00:00Z','2026-02-28T16:00:00Z'),

-- 24-Port Switch: 12 in - 3 out = 9
('00000000-0000-0000-0003-000000000031','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000016',12,198.00,'in','PO-2026-0016','done','Switch order',
 '00000000-0000-0000-0000-000000000015',NULL,'2026-01-16T11:00:00Z','2026-01-16T11:00:00Z'),
('00000000-0000-0000-0003-000000000032','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000016',3,349.99,'out','SO-2026-0240','done','Network switch sales',
 NULL,'00000000-0000-0000-0000-000000000015','2026-03-15T10:00:00Z','2026-03-15T10:00:00Z'),

-- Wi-Fi 6E AP: 15 in - 4 out = 11
('00000000-0000-0000-0003-000000000033','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000017',15,168.00,'in','PO-2026-0017','done','AP order',
 '00000000-0000-0000-0000-000000000015',NULL,'2026-01-22T13:00:00Z','2026-01-22T13:00:00Z'),
('00000000-0000-0000-0003-000000000034','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000017',4,299.99,'out','SO-2026-0255','done','AP sales',
 NULL,'00000000-0000-0000-0000-000000000015','2026-03-18T09:00:00Z','2026-03-18T09:00:00Z');

-- ── Batches ───────────────────────────────────────────────────────────────────
INSERT OR IGNORE INTO batches
    (id,tenant_id,product_id,batch_number,manufacture_date,expiry_date,
     qty_received,qty_remaining,created_at)
VALUES
('00000000-0000-0000-0004-000000000001','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000001',
 'BATCH-LAPTOP-2025-001','2025-06-01','',30,28,'2025-06-15T08:00:00Z'),
('00000000-0000-0000-0004-000000000002','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000001',
 'BATCH-LAPTOP-2026-001','2026-01-10','',20,17,'2026-01-15T09:00:00Z'),
('00000000-0000-0000-0004-000000000003','00000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0002-000000000014',
 'BATCH-TONER-2026-001','2026-01-15','2028-01-15',30,22,'2026-01-20T10:00:00Z');

-- ── Product Variants (for Laptop Pro 15) ─────────────────────────────────────
INSERT OR IGNORE INTO product_variants
    (id,product_id,sku,attributes,price_modifier,stock_qty,barcode,is_active)
VALUES
('00000000-0000-0000-0005-000000000001',
 '00000000-0000-0000-0002-000000000001',
 'LAPTOP-PRO-15-BLK-512',
 '{"color":"Space Black","storage":"512GB","memory":"16GB"}',
 0,25,'8901234560101',1),
('00000000-0000-0000-0005-000000000002',
 '00000000-0000-0000-0002-000000000001',
 'LAPTOP-PRO-15-SLV-1TB',
 '{"color":"Silver","storage":"1TB","memory":"32GB"}',
 200,20,'8901234560102',1);

-- ── Customers (3) ─────────────────────────────────────────────────────────────
INSERT OR IGNORE INTO customers (id,tenant_id,name,email,phone,company_name,city,country,is_active)
VALUES
('00000000-0000-0000-0006-000000000001','00000000-0000-0000-0000-000000000001',
 'Acme Corporation','buyer@acme.com','+1-212-555-0100','Acme Corp','New York','US',1),
('00000000-0000-0000-0006-000000000002','00000000-0000-0000-0000-000000000001',
 'TechStart Inc','orders@techstart.io','+1-415-555-0200','TechStart Inc','San Francisco','US',1),
('00000000-0000-0000-0006-000000000003','00000000-0000-0000-0000-000000000001',
 'Global Media Ltd','procurement@globalmedia.com','+44-20-7946-0300','Global Media','London','GB',1);

-- ── Suppliers (3) ─────────────────────────────────────────────────────────────
INSERT OR IGNORE INTO suppliers (id,tenant_id,name,email,phone,company_name,city,country,is_active)
VALUES
('00000000-0000-0000-0007-000000000001','00000000-0000-0000-0000-000000000001',
 'Tech Distributors Inc','sales@techdist.com','+1-800-555-0110','Tech Distributors','Austin','US',1),
('00000000-0000-0000-0007-000000000002','00000000-0000-0000-0000-000000000001',
 'Office Essentials Co','order@officeess.com','+1-800-555-0220','Office Essentials','Chicago','US',1),
('00000000-0000-0000-0007-000000000003','00000000-0000-0000-0000-000000000001',
 'Network Pro Supply','supply@netpro.com','+1-800-555-0330','Network Pro Supply','Seattle','US',1);
`
