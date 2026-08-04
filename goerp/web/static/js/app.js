/* ══════════════════════════════════════════════════════════════════════════
   GoERP — app.js   |   Full SPA Application
   ══════════════════════════════════════════════════════════════════════════ */

'use strict';

// ── API Client ──────────────────────────────────────────────────────────────
const API = {
  base: '/api/v1',
  token: null,

  setToken(t) { this.token = t; localStorage.setItem('goerp_token', t); },
  getToken() { return this.token || localStorage.getItem('goerp_token'); },
  clearToken() { this.token = null; localStorage.removeItem('goerp_token'); localStorage.removeItem('goerp_user'); },

  headers() {
    const h = { 'Content-Type': 'application/json' };
    const t = this.getToken();
    if (t) h['Authorization'] = 'Bearer ' + t;
    return h;
  },

  async request(method, path, body) {
    try {
      const opts = { method, headers: this.headers() };
      if (body) opts.body = JSON.stringify(body);
      const res = await fetch(this.base + path, opts);
      if (res.status === 401) { App.logout(); return null; }
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'Request failed');
      return data;
    } catch (e) {
      Toast.error(e.message);
      throw e;
    }
  },

  get(path)          { return this.request('GET', path); },
  post(path, body)   { return this.request('POST', path, body); },
  put(path, body)    { return this.request('PUT', path, body); },
  patch(path, body)  { return this.request('PATCH', path, body || {}); },
  delete(path)       { return this.request('DELETE', path); },
};

// ── Toast ───────────────────────────────────────────────────────────────────
const Toast = {
  show(msg, type = 'info', duration = 4000) {
    const icons = { success: 'fa-check-circle', error: 'fa-exclamation-circle', info: 'fa-info-circle' };
    const c = document.getElementById('toast-container');
    const t = document.createElement('div');
    t.className = `toast ${type}`;
    t.innerHTML = `<i class="fas ${icons[type] || icons.info}"></i><span>${msg}</span>`;
    c.appendChild(t);
    setTimeout(() => { t.style.animation = 'fadeOut .3s ease forwards'; setTimeout(() => t.remove(), 300); }, duration);
  },
  success(m) { this.show(m, 'success'); },
  error(m)   { this.show(m, 'error', 5000); },
  info(m)    { this.show(m, 'info'); },
};

// ── Helpers ─────────────────────────────────────────────────────────────────
const $ = id => document.getElementById(id);
const fmt = {
  currency(n) { return '$' + (parseFloat(n)||0).toLocaleString('en-US',{minimumFractionDigits:2,maximumFractionDigits:2}); },
  date(d) { if (!d) return '—'; return new Date(d).toLocaleDateString('en-US',{year:'numeric',month:'short',day:'numeric'}); },
  datetime(d) { if (!d) return '—'; return new Date(d).toLocaleString('en-US',{dateStyle:'short',timeStyle:'short'}); },
  percent(n) { return (parseFloat(n)||0).toFixed(1) + '%'; },
};

function statusBadge(status) {
  return `<span class="status-badge badge-${(status||'').toLowerCase().replace(/\s+/g,'-')}">${status||'—'}</span>`;
}
function actionBtns(...btns) {
  return `<div class="action-btns">${btns.join('')}</div>`;
}

// ── App Core ────────────────────────────────────────────────────────────────
const App = {
  currentPage: 'dashboard',
  charts: {},

  init() {
    const token = API.getToken();
    const userStr = localStorage.getItem('goerp_user');
    if (token && userStr) {
      try {
        const user = JSON.parse(userStr);
        this.showApp(user);
      } catch { this.showLogin(); }
    } else {
      this.showLogin();
    }
    this.bindGlobal();
  },

  showLogin() {
    $('login-page').style.display = 'flex';
    $('register-page').style.display = 'none';
    $('app').style.display = 'none';
  },

  showApp(user) {
    $('login-page').style.display = 'none';
    $('register-page').style.display = 'none';
    $('app').style.display = 'flex';
    $('user-name-display').textContent = user.full_name || user.email || 'User';
    $('user-role-display').textContent = user.is_super_admin ? 'Super Admin' : 'Admin';
    $('topbar-user-name').textContent = (user.full_name || 'Admin').split(' ')[0];
    this.navigate('dashboard');
  },

  navigate(page) {
    document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
    document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
    const pg = $('page-' + page);
    if (pg) pg.classList.add('active');
    const nav = document.querySelector(`[data-page="${page}"]`);
    if (nav) nav.classList.add('active');
    const titles = {
      dashboard:'Dashboard', inventory:'Inventory', sales:'Sales', purchases:'Purchases',
      accounting:'Accounting', crm:'CRM', hr:'Human Resources', invoices:'Invoices'
    };
    $('page-title').textContent = titles[page] || page;
    this.currentPage = page;
    this.loadPage(page);
  },

  loadPage(page) {
    switch(page) {
      case 'dashboard': Dashboard.load(); break;
      case 'inventory': Inventory.load(); break;
      case 'sales':     Sales.load(); break;
      case 'purchases': Purchases.load(); break;
      case 'accounting':Accounting.load(); break;
      case 'crm':       CRM.load(); break;
      case 'hr':        HR.load(); break;
      case 'invoices':  Sales.loadInvoices('all-invoices-body'); break;
    }
  },

  openModal(type, data) {
    const overlay = $('modal-overlay');
    const body = $('modal-body');
    const title = $('modal-title');
    const generators = {
      product:   () => Modals.product(data),
      order:     () => Modals.salesOrder(data),
      customer:  () => Modals.customer(data),
      supplier:  () => Modals.supplier(data),
      po:        () => Modals.purchaseOrder(data),
      journal:   () => Modals.journalEntry(data),
      lead:      () => Modals.lead(data),
      opportunity:()=> Modals.opportunity(data),
      activity:  () => Modals.activity(data),
      employee:  () => Modals.employee(data),
      leave:     () => Modals.leaveRequest(data),
      attendance:() => Modals.attendance(data),
      payroll:   () => Modals.payrollRun(data),
      stockmove: () => Modals.stockMove(data),
    };
    const fn = generators[type];
    if (!fn) return;
    const modal = fn();
    title.textContent = modal.title;
    body.innerHTML = modal.html;
    overlay.classList.add('open');
    if (modal.onOpen) setTimeout(modal.onOpen, 50);
  },

  closeModal() { $('modal-overlay').classList.remove('open'); },

  logout() {
    API.clearToken();
    this.showLogin();
    Toast.info('Signed out successfully.');
  },

  bindGlobal() {
    // Login form
    $('login-form').addEventListener('submit', async e => {
      e.preventDefault();
      const btn = $('login-btn');
      btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Signing in...';
      btn.disabled = true;
      try {
        const res = await API.post('/auth/login', {
          email: $('login-email').value,
          password: $('login-password').value,
          tenant_slug: 'demo'
        });
        if (res && res.access_token) {
          API.setToken(res.access_token);
          localStorage.setItem('goerp_user', JSON.stringify(res.user));
          this.showApp(res.user);
          Toast.success('Welcome back, ' + (res.user.full_name || 'Admin') + '!');
        }
      } catch(err) {
        $('login-error').textContent = err.message;
        $('login-error').style.display = 'block';
      } finally {
        btn.innerHTML = '<span>Sign In</span><i class="fas fa-arrow-right"></i>';
        btn.disabled = false;
      }
    });

    // Register form
    $('register-form').addEventListener('submit', async e => {
      e.preventDefault();
      try {
        const res = await API.post('/auth/register', {
          tenant_name: $('reg-company').value,
          full_name:   $('reg-name').value,
          email:       $('reg-email').value,
          password:    $('reg-password').value,
        });
        if (res && res.access_token) {
          API.setToken(res.access_token);
          localStorage.setItem('goerp_user', JSON.stringify(res.user));
          this.showApp(res.user);
          Toast.success('Organization created! Welcome to GoERP.');
        }
      } catch(err) {
        $('reg-error').textContent = err.message;
        $('reg-error').style.display = 'block';
      }
    });

    $('show-register').addEventListener('click', e => { e.preventDefault(); $('login-page').style.display='none'; $('register-page').style.display='flex'; });
    $('show-login').addEventListener('click',    e => { e.preventDefault(); $('register-page').style.display='none'; $('login-page').style.display='flex'; });

    // Navigation
    document.querySelectorAll('.nav-item').forEach(n => {
      n.addEventListener('click', e => { e.preventDefault(); this.navigate(n.dataset.page); });
    });

    // Sidebar toggle
    $('sidebar-toggle').addEventListener('click', () => $('sidebar').classList.toggle('collapsed'));
    $('mobile-toggle') && $('mobile-toggle').addEventListener('click', () => $('sidebar').classList.toggle('mobile-open'));

    // Logout
    $('logout-btn').addEventListener('click', () => this.logout());

    // Tab switching
    document.addEventListener('click', e => {
      if (e.target.matches('.tab')) {
        const tab = e.target;
        const tabId = tab.dataset.tab;
        const parent = tab.closest('.page') || tab.closest('.card-body');
        if (!parent) return;
        parent.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        parent.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
        tab.classList.add('active');
        const content = $(tabId);
        if (content) {
          content.classList.add('active');
          // Lazy load tab content
          if (tabId === 'inv-warehouses' && !content.dataset.loaded) { Inventory.loadWarehouses(); content.dataset.loaded = '1'; }
          if (tabId === 'inv-moves' && !content.dataset.loaded) { Inventory.loadMoves(); content.dataset.loaded = '1'; }
          if (tabId === 'inv-summary' && !content.dataset.loaded) { Inventory.loadSummary(); content.dataset.loaded = '1'; }
          if (tabId === 'sales-customers' && !content.dataset.loaded) { Sales.loadCustomers(); content.dataset.loaded = '1'; }
          if (tabId === 'sales-invoices' && !content.dataset.loaded) { Sales.loadInvoices('invoices-body'); content.dataset.loaded = '1'; }
          if (tabId === 'purch-suppliers' && !content.dataset.loaded) { Purchases.loadSuppliers(); content.dataset.loaded = '1'; }
          if (tabId === 'acc-accounts' && !content.dataset.loaded) { Accounting.loadAccounts(); content.dataset.loaded = '1'; }
          if (tabId === 'crm-opps' && !content.dataset.loaded) { CRM.loadOpportunities(); content.dataset.loaded = '1'; }
          if (tabId === 'crm-pipeline') { CRM.renderKanban(); }
          if (tabId === 'crm-activities' && !content.dataset.loaded) { CRM.loadActivities(); content.dataset.loaded = '1'; }
          if (tabId === 'hr-attendance' && !content.dataset.loaded) { HR.loadAttendance(); content.dataset.loaded = '1'; }
          if (tabId === 'hr-leaves' && !content.dataset.loaded) { HR.loadLeaves(); content.dataset.loaded = '1'; }
          if (tabId === 'hr-payroll-tab' && !content.dataset.loaded) { HR.loadPayroll(); content.dataset.loaded = '1'; }
        }
      }
    });

    // ESC key close modal
    document.addEventListener('keydown', e => { if (e.key === 'Escape') this.closeModal(); });
  }
};

// ── Dashboard ───────────────────────────────────────────────────────────────
const Dashboard = {
  async load() {
    try {
      const data = await API.get('/dashboard');
      if (!data) return;
      $('stat-revenue').textContent  = fmt.currency(data.revenue_this_month);
      $('stat-orders').textContent   = data.orders_this_month  || 0;
      $('stat-invoices').textContent = data.invoices_pending   || 0;
      $('stat-customers').textContent= data.customers_total    || 0;
      $('stat-employees').textContent= data.employees_total    || 0;
      $('stat-opps').textContent     = data.open_opportunities || 0;
      this.renderCharts();
    } catch(e) {
      this.renderCharts();
    }
  },

  renderCharts() {
    // Revenue chart
    const ctx = $('revenue-chart');
    if (ctx) {
      if (App.charts['revenue']) App.charts['revenue'].destroy();
      const months = ['Feb','Mar','Apr','May','Jun','Jul'];
      const rev = [42000,58000,51000,67000,73000,89000];
      App.charts['revenue'] = new Chart(ctx, {
        type: 'line',
        data: {
          labels: months,
          datasets: [{
            label: 'Revenue',
            data: rev,
            borderColor: '#2563eb',
            backgroundColor: 'rgba(37,99,235,.08)',
            borderWidth: 2.5,
            fill: true,
            tension: 0.4,
            pointBackgroundColor: '#2563eb',
            pointRadius: 4,
          }]
        },
        options: {
          responsive: true,
          plugins: { legend: { display: false } },
          scales: {
            y: { beginAtZero: false, grid: { color: '#f1f5f9' },
              ticks: { callback: v => '$' + (v/1000).toFixed(0) + 'k' }},
            x: { grid: { display: false }}
          }
        }
      });
    }

    // Pipeline donut
    const pctx = $('pipeline-chart');
    if (pctx) {
      if (App.charts['pipeline']) App.charts['pipeline'].destroy();
      App.charts['pipeline'] = new Chart(pctx, {
        type: 'doughnut',
        data: {
          labels: ['New','Qualified','Proposal','Negotiation','Won'],
          datasets: [{
            data: [8,14,22,18,36],
            backgroundColor: ['#94a3b8','#3b82f6','#f59e0b','#8b5cf6','#10b981'],
            borderWidth: 0,
          }]
        },
        options: {
          responsive: true,
          cutout: '65%',
          plugins: { legend: { position: 'right', labels: { boxWidth: 12, font: { size: 11 }}}}
        }
      });
    }
  }
};

// ── Inventory ───────────────────────────────────────────────────────────────
const Inventory = {
  products: [],

  async load() {
    await this.loadProducts();
  },

  async loadProducts() {
    const tbody = $('products-body');
    tbody.innerHTML = '<tr><td colspan="7" class="loading-row"><i class="fas fa-spinner fa-spin"></i> Loading...</td></tr>';
    try {
      const data = await API.get('/inventory/products');
      this.products = (data && data.data) ? data.data : (data || []);
      this.renderProducts(this.products);
    } catch { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">Failed to load products</td></tr>'; }
  },

  renderProducts(prods) {
    const tbody = $('products-body');
    if (!prods || !prods.length) { tbody.innerHTML = '<tr><td colspan="7" class="loading-row"><div class="empty-state"><i class="fas fa-boxes-stacked"></i><h4>No products yet</h4><p>Add your first product to get started</p></div></td></tr>'; return; }
    tbody.innerHTML = prods.map(p => `
      <tr>
        <td class="font-mono" style="font-size:12px;color:var(--text-muted)">${p.sku||'—'}</td>
        <td><strong>${p.name||p.names?.en||'—'}</strong></td>
        <td>${p.category||'—'}</td>
        <td>${p.uom||p.unit_of_measure||'—'}</td>
        <td class="text-right">${fmt.currency(p.base_price||p.sale_price||0)}</td>
        <td>${statusBadge(p.active===false?'inactive':'active')}</td>
        <td>${actionBtns(
          `<button class="btn-action edit" onclick='Inventory.editProduct(${JSON.stringify(p)})' title="Edit"><i class="fas fa-pen"></i></button>`,
          `<button class="btn-action delete" onclick="Inventory.deleteProduct('${p.id}')" title="Delete"><i class="fas fa-trash"></i></button>`
        )}</td>
      </tr>`).join('');
  },

  searchProducts(q) {
    const filtered = this.products.filter(p =>
      (p.name||p.names?.en||'').toLowerCase().includes(q.toLowerCase()) ||
      (p.sku||'').toLowerCase().includes(q.toLowerCase())
    );
    this.renderProducts(filtered);
  },

  async loadWarehouses() {
    const tbody = $('warehouses-body');
    try {
      const data = await API.get('/inventory/warehouses');
      const wh = (data && data.data) ? data.data : (data || []);
      if (!wh.length) { tbody.innerHTML = '<tr><td colspan="5" class="loading-row text-muted">No warehouses found</td></tr>'; return; }
      tbody.innerHTML = wh.map(w => `
        <tr>
          <td class="font-mono">${w.code||'—'}</td>
          <td><strong>${w.name}</strong></td>
          <td>${w.address||'—'}</td>
          <td>${statusBadge(w.active===false?'inactive':'active')}</td>
          <td>${actionBtns(`<button class="btn-action edit" title="Edit"><i class="fas fa-pen"></i></button>`)}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="5" class="loading-row text-muted">Failed to load warehouses</td></tr>'; }
  },

  async loadMoves() {
    const tbody = $('moves-body');
    try {
      const data = await API.get('/inventory/stock-moves');
      const moves = (data && data.data) ? data.data : (data || []);
      if (!moves.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No stock moves found</td></tr>'; return; }
      tbody.innerHTML = moves.map(m => `
        <tr>
          <td>${fmt.date(m.created_at)}</td>
          <td>${m.product_id||'—'}</td>
          <td>${m.from_location||'<i>—</i>'}</td>
          <td>${m.to_location||'—'}</td>
          <td class="text-right"><strong>${m.quantity}</strong></td>
          <td class="font-mono" style="font-size:11px">${m.reference||'—'}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load moves</td></tr>'; }
  },

  async loadSummary() {
    const tbody = $('summary-body');
    try {
      const data = await API.get('/inventory/stock-summary');
      const summary = (data && data.data) ? data.data : (data || []);
      if (!summary.length) { tbody.innerHTML = '<tr><td colspan="5" class="loading-row text-muted">No stock data</td></tr>'; return; }
      tbody.innerHTML = summary.map(s => `
        <tr>
          <td>${s.product_name||s.product_id||'—'}</td>
          <td>${s.warehouse_name||s.warehouse_id||'—'}</td>
          <td class="text-right"><strong>${s.quantity_on_hand||0}</strong></td>
          <td class="text-right">${s.quantity_reserved||0}</td>
          <td class="text-right" style="color:var(--success);font-weight:600">${(s.quantity_on_hand||0)-(s.quantity_reserved||0)}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="5" class="loading-row text-muted">Failed to load summary</td></tr>'; }
  },

  editProduct(p) { App.openModal('product', p); },
  async deleteProduct(id) {
    if (!confirm('Delete this product?')) return;
    await API.delete('/inventory/products/' + id);
    Toast.success('Product deleted'); this.loadProducts();
  }
};

// ── Sales ────────────────────────────────────────────────────────────────────
const Sales = {
  async load() {
    await this.loadOrders();
  },

  async loadOrders() {
    const tbody = $('orders-body');
    try {
      const data = await API.get('/sales/orders');
      const orders = (data && data.data) ? data.data : (data || []);
      if (!orders.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No orders yet</td></tr>'; return; }
      tbody.innerHTML = orders.map(o => `
        <tr>
          <td class="font-mono">${o.order_number||o.id?.slice(0,8)||'—'}</td>
          <td>${o.customer_name||o.customer_id||'—'}</td>
          <td>${fmt.date(o.order_date||o.created_at)}</td>
          <td class="text-right">${fmt.currency(o.total||0)}</td>
          <td>${statusBadge(o.state||o.status||'draft')}</td>
          <td>${actionBtns(
            `<button class="btn-action view" title="View"><i class="fas fa-eye"></i></button>`,
            o.state==='draft' ? `<button class="btn-action confirm" onclick="Sales.confirmOrder('${o.id}')" title="Confirm"><i class="fas fa-check"></i></button>` : '',
            `<button class="btn-action delete" onclick="Sales.cancelOrder('${o.id}')" title="Cancel"><i class="fas fa-ban"></i></button>`
          )}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load orders</td></tr>'; }
  },

  async loadCustomers() {
    const tbody = $('customers-body');
    try {
      const data = await API.get('/sales/customers');
      const custs = (data && data.data) ? data.data : (data || []);
      if (!custs.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No customers yet</td></tr>'; return; }
      tbody.innerHTML = custs.map(c => `
        <tr>
          <td><strong>${c.name}</strong></td>
          <td>${c.email||'—'}</td>
          <td>${c.phone||'—'}</td>
          <td>${c.city||'—'}</td>
          <td class="text-right">${fmt.currency(c.credit_limit||0)}</td>
          <td>${actionBtns(`<button class="btn-action edit" title="Edit"><i class="fas fa-pen"></i></button>`)}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load customers</td></tr>'; }
  },

  async loadInvoices(tbodyId) {
    const tbody = $(tbodyId);
    if (!tbody) return;
    try {
      const data = await API.get('/sales/invoices');
      const invs = (data && data.data) ? data.data : (data || []);
      if (!invs.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No invoices yet</td></tr>'; return; }
      tbody.innerHTML = invs.map(i => `
        <tr>
          <td class="font-mono">${i.invoice_number||i.id?.slice(0,8)||'—'}</td>
          <td>${i.customer_name||i.customer_id||'—'}</td>
          <td>${fmt.date(i.invoice_date||i.created_at)}</td>
          <td class="text-right"><strong>${fmt.currency(i.total||0)}</strong></td>
          <td>${statusBadge(i.state||'pending')}</td>
          <td>${actionBtns(`<button class="btn-action view" title="View"><i class="fas fa-eye"></i></button>`)}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load invoices</td></tr>'; }
  },

  async confirmOrder(id) {
    await API.patch('/sales/orders/' + id + '/confirm');
    Toast.success('Order confirmed!'); this.loadOrders();
  },
  async cancelOrder(id) {
    if (!confirm('Cancel this order?')) return;
    await API.patch('/sales/orders/' + id + '/cancel');
    Toast.info('Order cancelled'); this.loadOrders();
  }
};

// ── Purchases ────────────────────────────────────────────────────────────────
const Purchases = {
  async load() { await this.loadOrders(); },

  async loadOrders() {
    const tbody = $('po-body');
    try {
      const data = await API.get('/purchases/orders');
      const pos = (data && data.data) ? data.data : (data || []);
      if (!pos.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No purchase orders yet</td></tr>'; return; }
      tbody.innerHTML = pos.map(p => `
        <tr>
          <td class="font-mono">${p.po_number||p.id?.slice(0,8)||'—'}</td>
          <td>${p.supplier_name||p.supplier_id||'—'}</td>
          <td>${fmt.date(p.order_date||p.created_at)}</td>
          <td class="text-right">${fmt.currency(p.total||0)}</td>
          <td>${statusBadge(p.state||p.status||'draft')}</td>
          <td>${actionBtns(`<button class="btn-action view" title="View"><i class="fas fa-eye"></i></button>`)}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load orders</td></tr>'; }
  },

  async loadSuppliers() {
    const tbody = $('suppliers-body');
    try {
      const data = await API.get('/purchases/suppliers');
      const supps = (data && data.data) ? data.data : (data || []);
      if (!supps.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No suppliers yet</td></tr>'; return; }
      tbody.innerHTML = supps.map(s => `
        <tr>
          <td><strong>${s.name}</strong></td>
          <td>${s.email||'—'}</td>
          <td>${s.phone||'—'}</td>
          <td>${s.city||'—'}</td>
          <td>${s.payment_terms||'Net 30'}</td>
          <td>${actionBtns(`<button class="btn-action edit" title="Edit"><i class="fas fa-pen"></i></button>`)}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load suppliers</td></tr>'; }
  }
};

// ── Accounting ───────────────────────────────────────────────────────────────
const Accounting = {
  async load() { await this.loadJournalEntries(); },

  async loadJournalEntries() {
    const tbody = $('journal-body');
    try {
      const data = await API.get('/accounting/journal-entries');
      const entries = (data && data.data) ? data.data : (data || []);
      if (!entries.length) { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">No journal entries yet</td></tr>'; return; }
      tbody.innerHTML = entries.map(e => `
        <tr>
          <td class="font-mono" style="font-size:11px">${e.reference||e.id?.slice(0,8)||'—'}</td>
          <td>${fmt.date(e.date||e.created_at)}</td>
          <td>${e.description||'—'}</td>
          <td class="text-right">${fmt.currency(e.total_debit||0)}</td>
          <td class="text-right">${fmt.currency(e.total_credit||0)}</td>
          <td>${statusBadge(e.state||e.status||'draft')}</td>
          <td>${actionBtns(
            e.state!=='posted' ? `<button class="btn-action confirm" onclick="Accounting.postEntry('${e.id}')" title="Post"><i class="fas fa-check-double"></i></button>` : '',
            `<button class="btn-action view" title="View"><i class="fas fa-eye"></i></button>`
          )}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">Failed to load journal entries</td></tr>'; }
  },

  async loadAccounts() {
    const tbody = $('accounts-body');
    try {
      const data = await API.get('/accounting/accounts');
      const accs = (data && data.data) ? data.data : (data || []);
      if (!accs.length) { tbody.innerHTML = '<tr><td colspan="5" class="loading-row text-muted">No accounts found</td></tr>'; return; }
      tbody.innerHTML = accs.map(a => `
        <tr>
          <td class="font-mono">${a.code}</td>
          <td><strong>${a.name}</strong></td>
          <td>${a.account_type||a.type||'—'}</td>
          <td>${a.normal_balance||'—'}</td>
          <td>${statusBadge(a.is_active===false?'inactive':'active')}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="5" class="loading-row text-muted">Failed to load accounts</td></tr>'; }
  },

  async postEntry(id) {
    await API.patch('/accounting/journal-entries/' + id + '/post');
    Toast.success('Journal entry posted!'); this.loadJournalEntries();
  }
};

// ── CRM ──────────────────────────────────────────────────────────────────────
const CRM = {
  opps: [],

  async load() {
    await this.loadStats();
    await this.loadLeads();
  },

  async loadStats() {
    try {
      const s = await API.get('/crm/pipeline/stats');
      if (!s) return;
      $('crm-total-leads').textContent = s.total_leads || 0;
      $('crm-total-opps').textContent  = s.total_opportunities || 0;
      $('crm-revenue').textContent     = fmt.currency(s.total_revenue || 0);
      $('crm-win-rate').textContent    = fmt.percent(s.win_rate || 0);
    } catch {}
  },

  async loadLeads(status) {
    const tbody = $('leads-body');
    try {
      const path = '/crm/leads' + (status ? '?status=' + status : '');
      const data = await API.get(path);
      const leads = (data && data.data) ? data.data : (data || []);
      if (!leads.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No leads yet</td></tr>'; return; }
      tbody.innerHTML = leads.map(l => `
        <tr>
          <td><strong>${l.name}</strong></td>
          <td>${l.company||'—'}</td>
          <td>${l.email||'—'}</td>
          <td><span class="status-badge badge-${l.source||'other'}">${l.source||'—'}</span></td>
          <td>${statusBadge(l.status||'new')}</td>
          <td>${actionBtns(
            `<button class="btn-action confirm" onclick="CRM.convertLead('${l.id}')" title="Convert to Opportunity"><i class="fas fa-arrow-up-right-from-square"></i></button>`,
            `<button class="btn-action edit" onclick='App.openModal("lead",${JSON.stringify(l)})' title="Edit"><i class="fas fa-pen"></i></button>`,
            `<button class="btn-action delete" onclick="CRM.deleteLead('${l.id}')" title="Delete"><i class="fas fa-trash"></i></button>`
          )}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load leads</td></tr>'; }
  },

  async loadOpportunities() {
    const tbody = $('opps-body');
    try {
      const data = await API.get('/crm/opportunities');
      this.opps = (data && data.data) ? data.data : (data || []);
      if (!this.opps.length) { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">No opportunities yet</td></tr>'; return; }
      tbody.innerHTML = this.opps.map(o => `
        <tr>
          <td><strong>${o.name}</strong></td>
          <td>${o.company||'—'}</td>
          <td>${statusBadge(o.stage||'new')}</td>
          <td class="text-right">${fmt.currency(o.expected_revenue||0)}</td>
          <td class="text-right">${fmt.percent(o.probability||0)}</td>
          <td>${fmt.date(o.expected_close)}</td>
          <td>${actionBtns(
            `<button class="btn-action approve" onclick="CRM.markWon('${o.id}')" title="Mark Won"><i class="fas fa-trophy"></i></button>`,
            `<button class="btn-action reject" onclick="CRM.markLost('${o.id}')" title="Mark Lost"><i class="fas fa-times"></i></button>`,
            `<button class="btn-action edit" title="Edit"><i class="fas fa-pen"></i></button>`
          )}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">Failed to load opportunities</td></tr>'; }
  },

  renderKanban() {
    const board = $('kanban-board');
    const stages = ['new','qualified','proposal','negotiation','won','lost'];
    const labels  = {new:'New',qualified:'Qualified',proposal:'Proposal',negotiation:'Negotiation',won:'Won ✓',lost:'Lost ✗'};
    const colOpps = {};
    stages.forEach(s => colOpps[s] = []);
    this.opps.forEach(o => { if (colOpps[o.stage]) colOpps[o.stage].push(o); });
    board.innerHTML = stages.map(s => `
      <div class="kanban-col">
        <div class="kanban-col-header">
          <h4>${labels[s]}</h4>
          <span class="kanban-count">${colOpps[s].length}</span>
        </div>
        <div class="kanban-cards">
          ${colOpps[s].map(o => `
            <div class="kanban-card">
              <div class="kanban-card-title">${o.name}</div>
              <div class="kanban-card-meta">
                <span><i class="fas fa-building"></i> ${o.company||'—'}</span>
                <span><i class="fas fa-percent"></i> ${fmt.percent(o.probability)}</span>
              </div>
              <div class="kanban-card-value">${fmt.currency(o.expected_revenue)}</div>
            </div>`).join('') || '<p style="text-align:center;color:var(--text-muted);font-size:11px;padding:16px">Empty</p>'}
        </div>
      </div>`).join('');
  },

  async loadActivities() {
    const tbody = $('activities-body');
    try {
      const data = await API.get('/crm/activities');
      const acts = (data && data.data) ? data.data : (data || []);
      if (!acts.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No activities yet</td></tr>'; return; }
      tbody.innerHTML = acts.map(a => `
        <tr>
          <td><span class="status-badge">${a.type||'—'}</span></td>
          <td><strong>${a.title}</strong></td>
          <td>${a.assigned_to||'—'}</td>
          <td>${fmt.date(a.due_date)}</td>
          <td>${statusBadge(a.is_done?'done':'pending')}</td>
          <td>${!a.is_done ? actionBtns(`<button class="btn-action approve" onclick="CRM.completeActivity('${a.id}')" title="Mark Done"><i class="fas fa-check"></i></button>`) : '—'}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load activities</td></tr>'; }
  },

  async markWon(id) { await API.patch('/crm/opportunities/'+id+'/won'); Toast.success('Opportunity marked as Won!'); this.loadOpportunities(); },
  async markLost(id) {
    const reason = prompt('Reason for losing (optional):');
    await API.patch('/crm/opportunities/'+id+'/lost', { reason: reason||'' });
    Toast.info('Opportunity marked as Lost'); this.loadOpportunities();
  },
  async convertLead(id) {
    const name = prompt('Opportunity name:');
    if (!name) return;
    await API.post('/crm/leads/'+id+'/convert', { name, stage:'new', probability:20 });
    Toast.success('Lead converted to opportunity!'); this.loadLeads(); this.loadOpportunities();
  },
  async deleteLead(id) {
    if (!confirm('Delete this lead?')) return;
    await API.delete('/crm/leads/'+id);
    Toast.info('Lead deleted'); this.loadLeads();
  },
  async completeActivity(id) {
    await API.patch('/crm/activities/'+id+'/complete');
    Toast.success('Activity marked as done!'); this.loadActivities();
  }
};

// ── HR ───────────────────────────────────────────────────────────────────────
const HR = {
  async load() {
    await this.loadStats();
    await this.loadEmployees();
  },

  async loadStats() {
    try {
      const s = await API.get('/hr/stats');
      if (!s) return;
      $('hr-total').textContent         = s.total_employees   || 0;
      $('hr-active').textContent        = s.active_employees  || 0;
      $('hr-leave').textContent         = s.on_leave          || 0;
      $('hr-pending-leaves').textContent= s.pending_leaves    || 0;
      $('hr-payroll').textContent       = fmt.currency(s.total_payroll || 0);
    } catch {}
  },

  async loadEmployees(dept) {
    const tbody = $('employees-body');
    try {
      const path = '/hr/employees' + (dept ? '?department='+dept : '');
      const data = await API.get(path);
      const emps = (data && data.data) ? data.data : (data || []);
      if (!emps.length) { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">No employees yet</td></tr>'; return; }
      tbody.innerHTML = emps.map(e => `
        <tr>
          <td class="font-mono" style="font-size:11px">${e.employee_number||'—'}</td>
          <td><strong>${e.full_name||e.first_name+' '+e.last_name}</strong></td>
          <td>${e.department||'—'}</td>
          <td>${e.job_title||'—'}</td>
          <td>${e.email}</td>
          <td>${statusBadge(e.status||'active')}</td>
          <td>${actionBtns(
            `<button class="btn-action edit" onclick='App.openModal("employee",${JSON.stringify(e)})' title="Edit"><i class="fas fa-pen"></i></button>`,
            `<button class="btn-action view" onclick="HR.viewContract('${e.id}')" title="Contract"><i class="fas fa-file-contract"></i></button>`
          )}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">Failed to load employees</td></tr>'; }
  },

  async loadAttendance() {
    const tbody = $('attendance-body');
    try {
      const data = await API.get('/hr/attendance');
      const atts = (data && data.data) ? data.data : (data || []);
      if (!atts.length) { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">No attendance records</td></tr>'; return; }
      tbody.innerHTML = atts.map(a => `
        <tr>
          <td>${a.employee_name||a.employee_id||'—'}</td>
          <td>${fmt.date(a.date)}</td>
          <td>${a.check_in ? fmt.datetime(a.check_in) : '—'}</td>
          <td>${a.check_out ? fmt.datetime(a.check_out) : '—'}</td>
          <td class="text-right">${(a.work_hours||0).toFixed(1)}h</td>
          <td class="text-right">${(a.overtime_hours||0).toFixed(1)}h</td>
          <td>${statusBadge(a.status||'present')}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">Failed to load attendance</td></tr>'; }
  },

  async loadLeaves(status) {
    const tbody = $('leaves-body');
    try {
      const path = '/hr/leaves' + (status ? '?status='+status : '');
      const data = await API.get(path);
      const leaves = (data && data.data) ? data.data : (data || []);
      if (!leaves.length) { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">No leave requests</td></tr>'; return; }
      tbody.innerHTML = leaves.map(l => `
        <tr>
          <td>${l.employee_name||l.employee_id||'—'}</td>
          <td>${statusBadge(l.leave_type||'annual')}</td>
          <td>${fmt.date(l.start_date)}</td>
          <td>${fmt.date(l.end_date)}</td>
          <td class="text-right">${l.total_days||0}d</td>
          <td>${statusBadge(l.status||'pending')}</td>
          <td>${l.status==='pending' ? actionBtns(
            `<button class="btn-action approve" onclick="HR.approveLeave('${l.id}')" title="Approve"><i class="fas fa-check"></i></button>`,
            `<button class="btn-action reject" onclick="HR.rejectLeave('${l.id}')" title="Reject"><i class="fas fa-times"></i></button>`
          ) : '—'}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="7" class="loading-row text-muted">Failed to load leaves</td></tr>'; }
  },

  async loadPayroll() {
    const tbody = $('payroll-body');
    try {
      const data = await API.get('/hr/payroll');
      const runs = (data && data.data) ? data.data : (data || []);
      if (!runs.length) { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">No payroll runs yet</td></tr>'; return; }
      const months = ['','Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
      tbody.innerHTML = runs.map(p => `
        <tr>
          <td><strong>${months[p.month]||p.month} ${p.year}</strong></td>
          <td class="text-right">${p.employee_count||0}</td>
          <td class="text-right">${fmt.currency(p.total_gross||0)}</td>
          <td class="text-right" style="color:var(--danger)">${fmt.currency(p.total_deductions||0)}</td>
          <td class="text-right" style="color:var(--success);font-weight:700">${fmt.currency(p.total_net||0)}</td>
          <td>${statusBadge(p.status||'draft')}</td>
        </tr>`).join('');
    } catch { tbody.innerHTML = '<tr><td colspan="6" class="loading-row text-muted">Failed to load payroll</td></tr>'; }
  },

  async approveLeave(id) { await API.patch('/hr/leaves/'+id+'/approve'); Toast.success('Leave approved!'); this.loadLeaves(); },
  async rejectLeave(id)  {
    const reason = prompt('Rejection reason:');
    await API.patch('/hr/leaves/'+id+'/reject', { reason: reason||'Rejected' });
    Toast.info('Leave rejected'); this.loadLeaves();
  },
  async viewContract(empId) {
    const c = await API.get('/hr/employees/'+empId+'/contract');
    if (c) Toast.info(`Contract: ${c.contract_type} | Salary: ${fmt.currency(c.total_salary)}`);
    else Toast.error('No active contract found');
  }
};

// ── Modals ───────────────────────────────────────────────────────────────────
const Modals = {
  product(data) {
    const isEdit = !!data;
    return {
      title: isEdit ? 'Edit Product' : 'New Product',
      html: `
        <div class="form-row">
          <div class="form-group"><label>SKU</label><input type="text" id="m-sku" value="${data?.sku||''}"/></div>
          <div class="form-group"><label>Product Name *</label><input type="text" id="m-name" value="${data?.name||data?.names?.en||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Category</label><input type="text" id="m-cat" value="${data?.category||''}"/></div>
          <div class="form-group"><label>Unit of Measure</label>
            <select id="m-uom"><option value="pcs"${(data?.uom||data?.unit_of_measure)==='pcs'?' selected':''}>Pieces</option><option value="kg">Kilogram</option><option value="liter">Liter</option><option value="meter">Meter</option></select>
          </div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Sale Price</label><input type="number" id="m-price" step="0.01" value="${data?.base_price||data?.sale_price||0}"/></div>
          <div class="form-group"><label>Cost Price</label><input type="number" id="m-cost" step="0.01" value="${data?.cost_price||0}"/></div>
        </div>
        <div class="form-group"><label>Description</label><textarea id="m-desc">${data?.description||''}</textarea></div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveProduct('${data?.id||''}')"><i class="fas fa-save"></i> Save</button>
        </div>`,
    };
  },
  async saveProduct(id) {
    const body = { sku: $('m-sku').value, name: $('m-name').value, category: $('m-cat').value,
      uom: $('m-uom').value, base_price: parseFloat($('m-price').value)||0, cost_price: parseFloat($('m-cost').value)||0,
      description: $('m-desc').value };
    if (!body.name) { Toast.error('Product name is required'); return; }
    try {
      if (id) await API.put('/inventory/products/'+id, body);
      else await API.post('/inventory/products', body);
      Toast.success(id ? 'Product updated!' : 'Product created!');
      App.closeModal(); Inventory.loadProducts();
    } catch {}
  },

  customer(data) {
    return {
      title: data ? 'Edit Customer' : 'New Customer',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Name *</label><input type="text" id="m-cname" value="${data?.name||''}"/></div>
          <div class="form-group"><label>Email</label><input type="email" id="m-cemail" value="${data?.email||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Phone</label><input type="text" id="m-cphone" value="${data?.phone||''}"/></div>
          <div class="form-group"><label>City</label><input type="text" id="m-ccity" value="${data?.city||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Credit Limit</label><input type="number" id="m-climit" value="${data?.credit_limit||0}"/></div>
          <div class="form-group"><label>Payment Terms</label>
            <select id="m-cterms"><option>Net 30</option><option>Net 60</option><option>Immediate</option></select>
          </div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveCustomer('${data?.id||''}')"><i class="fas fa-save"></i> Save</button>
        </div>`,
    };
  },
  async saveCustomer(id) {
    const body = { name: $('m-cname').value, email: $('m-cemail').value, phone: $('m-cphone').value,
      city: $('m-ccity').value, credit_limit: parseFloat($('m-climit').value)||0 };
    if (!body.name) { Toast.error('Customer name is required'); return; }
    try {
      if (id) await API.put('/sales/customers/'+id, body);
      else await API.post('/sales/customers', body);
      Toast.success(id ? 'Customer updated!' : 'Customer created!');
      App.closeModal(); Sales.loadCustomers();
    } catch {}
  },

  supplier(data) {
    return {
      title: data ? 'Edit Supplier' : 'New Supplier',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Name *</label><input type="text" id="m-sname" value="${data?.name||''}"/></div>
          <div class="form-group"><label>Email</label><input type="email" id="m-semail" value="${data?.email||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Phone</label><input type="text" id="m-sphone" value="${data?.phone||''}"/></div>
          <div class="form-group"><label>City</label><input type="text" id="m-scity" value="${data?.city||''}"/></div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveSupplier('${data?.id||''}')"><i class="fas fa-save"></i> Save</button>
        </div>`,
    };
  },
  async saveSupplier(id) {
    const body = { name: $('m-sname').value, email: $('m-semail').value, phone: $('m-sphone').value, city: $('m-scity').value };
    if (!body.name) { Toast.error('Supplier name is required'); return; }
    try {
      if (id) await API.put('/purchases/suppliers/'+id, body);
      else await API.post('/purchases/suppliers', body);
      Toast.success('Supplier saved!'); App.closeModal(); Purchases.loadSuppliers();
    } catch {}
  },

  salesOrder() {
    return {
      title: 'New Sales Order',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Customer Name *</label><input type="text" id="m-ocust" placeholder="Customer name"/></div>
          <div class="form-group"><label>Order Date</label><input type="date" id="m-odate" value="${new Date().toISOString().split('T')[0]}"/></div>
        </div>
        <div class="form-group"><label>Notes</label><textarea id="m-onotes" placeholder="Order notes..."></textarea></div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveSalesOrder()"><i class="fas fa-save"></i> Create Order</button>
        </div>`,
    };
  },
  async saveSalesOrder() {
    const body = { customer_name: $('m-ocust').value, order_date: $('m-odate').value, notes: $('m-onotes').value, lines: [] };
    if (!body.customer_name) { Toast.error('Customer name is required'); return; }
    try {
      await API.post('/sales/orders', body);
      Toast.success('Sales order created!'); App.closeModal(); Sales.loadOrders();
    } catch {}
  },

  purchaseOrder() {
    return {
      title: 'New Purchase Order',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Supplier Name *</label><input type="text" id="m-posup" placeholder="Supplier name"/></div>
          <div class="form-group"><label>Order Date</label><input type="date" id="m-podate" value="${new Date().toISOString().split('T')[0]}"/></div>
        </div>
        <div class="form-group"><label>Notes</label><textarea id="m-ponotes" placeholder="PO notes..."></textarea></div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.savePurchaseOrder()"><i class="fas fa-save"></i> Create PO</button>
        </div>`,
    };
  },
  async savePurchaseOrder() {
    const body = { supplier_name: $('m-posup').value, order_date: $('m-podate').value, notes: $('m-ponotes').value, lines: [] };
    if (!body.supplier_name) { Toast.error('Supplier name is required'); return; }
    try {
      await API.post('/purchases/orders', body);
      Toast.success('Purchase order created!'); App.closeModal(); Purchases.loadOrders();
    } catch {}
  },

  journalEntry() {
    return {
      title: 'New Journal Entry',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Date</label><input type="date" id="m-jdate" value="${new Date().toISOString().split('T')[0]}"/></div>
          <div class="form-group"><label>Reference</label><input type="text" id="m-jref" placeholder="JE-001"/></div>
        </div>
        <div class="form-group"><label>Description *</label><input type="text" id="m-jdesc" placeholder="Journal entry description"/></div>
        <div class="divider"></div>
        <p style="font-size:12px;color:var(--text-muted);font-weight:600">Lines (Debit = Credit)</p>
        <div class="form-row">
          <div class="form-group"><label>Account (Debit)</label><input type="text" id="m-jdebitacc" placeholder="Account code"/></div>
          <div class="form-group"><label>Debit Amount</label><input type="number" id="m-jdebit" step="0.01" placeholder="0.00"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Account (Credit)</label><input type="text" id="m-jcreditacc" placeholder="Account code"/></div>
          <div class="form-group"><label>Credit Amount</label><input type="number" id="m-jcredit" step="0.01" placeholder="0.00"/></div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveJournalEntry()"><i class="fas fa-save"></i> Save Entry</button>
        </div>`,
    };
  },
  async saveJournalEntry() {
    const body = {
      date: $('m-jdate').value, reference: $('m-jref').value, description: $('m-jdesc').value,
      lines: [
        { account_code: $('m-jdebitacc').value, debit: parseFloat($('m-jdebit').value)||0, credit: 0, description: 'Debit' },
        { account_code: $('m-jcreditacc').value, debit: 0, credit: parseFloat($('m-jcredit').value)||0, description: 'Credit' }
      ]
    };
    if (!body.description) { Toast.error('Description is required'); return; }
    try {
      await API.post('/accounting/journal-entries', body);
      Toast.success('Journal entry created!'); App.closeModal(); Accounting.loadJournalEntries();
    } catch {}
  },

  lead(data) {
    return {
      title: data ? 'Edit Lead' : 'New Lead',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Full Name *</label><input type="text" id="m-lname" value="${data?.name||''}"/></div>
          <div class="form-group"><label>Company</label><input type="text" id="m-lcomp" value="${data?.company||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Email</label><input type="email" id="m-lemail" value="${data?.email||''}"/></div>
          <div class="form-group"><label>Phone</label><input type="text" id="m-lphone" value="${data?.phone||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Source</label>
            <select id="m-lsource">
              <option value="website">Website</option><option value="referral">Referral</option>
              <option value="cold_call">Cold Call</option><option value="email">Email</option><option value="other">Other</option>
            </select>
          </div>
          <div class="form-group"><label>Status</label>
            <select id="m-lstatus">
              <option value="new">New</option><option value="contacted">Contacted</option>
              <option value="qualified">Qualified</option><option value="lost">Lost</option>
            </select>
          </div>
        </div>
        <div class="form-group"><label>Notes</label><textarea id="m-lnotes">${data?.notes||''}</textarea></div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveLead('${data?.id||''}')"><i class="fas fa-save"></i> Save</button>
        </div>`,
    };
  },
  async saveLead(id) {
    const body = { name: $('m-lname').value, company: $('m-lcomp').value, email: $('m-lemail').value,
      phone: $('m-lphone').value, source: $('m-lsource').value, status: $('m-lstatus').value, notes: $('m-lnotes').value };
    if (!body.name) { Toast.error('Name is required'); return; }
    try {
      if (id) await API.put('/crm/leads/'+id, body);
      else await API.post('/crm/leads', body);
      Toast.success('Lead saved!'); App.closeModal(); CRM.loadLeads();
    } catch {}
  },

  opportunity(data) {
    return {
      title: 'New Opportunity',
      html: `
        <div class="form-group"><label>Opportunity Name *</label><input type="text" id="m-oname" value="${data?.name||''}"/></div>
        <div class="form-row">
          <div class="form-group"><label>Customer Name *</label><input type="text" id="m-ocname" value="${data?.customer_name||''}"/></div>
          <div class="form-group"><label>Company</label><input type="text" id="m-occomp" value="${data?.company||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Expected Revenue</label><input type="number" id="m-orev" value="${data?.expected_revenue||0}"/></div>
          <div class="form-group"><label>Probability %</label><input type="number" id="m-oprob" value="${data?.probability||20}" min="0" max="100"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Stage</label>
            <select id="m-ostage">
              <option value="new">New</option><option value="qualified">Qualified</option>
              <option value="proposal">Proposal</option><option value="negotiation">Negotiation</option>
            </select>
          </div>
          <div class="form-group"><label>Expected Close</label><input type="date" id="m-oclose" value="${new Date().toISOString().split('T')[0]}"/></div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveOpportunity('${data?.id||''}')"><i class="fas fa-save"></i> Save</button>
        </div>`,
    };
  },
  async saveOpportunity(id) {
    const body = { name: $('m-oname').value, customer_name: $('m-ocname').value, company: $('m-occomp').value,
      expected_revenue: parseFloat($('m-orev').value)||0, probability: parseFloat($('m-oprob').value)||0,
      stage: $('m-ostage').value, expected_close: $('m-oclose').value };
    if (!body.name || !body.customer_name) { Toast.error('Name and customer are required'); return; }
    try {
      if (id) await API.put('/crm/opportunities/'+id, body);
      else await API.post('/crm/opportunities', body);
      Toast.success('Opportunity saved!'); App.closeModal(); CRM.loadOpportunities();
    } catch {}
  },

  activity(data) {
    return {
      title: 'New Activity',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Type</label>
            <select id="m-atype">
              <option value="call">Call</option><option value="email">Email</option>
              <option value="meeting">Meeting</option><option value="task">Task</option>
            </select>
          </div>
          <div class="form-group"><label>Due Date</label><input type="date" id="m-adue" value="${new Date().toISOString().split('T')[0]}"/></div>
        </div>
        <div class="form-group"><label>Title *</label><input type="text" id="m-atitle" placeholder="Activity title"/></div>
        <div class="form-group"><label>Description</label><textarea id="m-adesc" placeholder="Details..."></textarea></div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveActivity()"><i class="fas fa-save"></i> Save</button>
        </div>`,
    };
  },
  async saveActivity() {
    const body = { type: $('m-atype').value, title: $('m-atitle').value, description: $('m-adesc').value, due_date: $('m-adue').value };
    if (!body.title) { Toast.error('Title is required'); return; }
    try {
      await API.post('/crm/activities', body);
      Toast.success('Activity created!'); App.closeModal(); CRM.loadActivities();
    } catch {}
  },

  employee(data) {
    return {
      title: data ? 'Edit Employee' : 'New Employee',
      html: `
        <div class="form-row">
          <div class="form-group"><label>First Name *</label><input type="text" id="m-efname" value="${data?.first_name||''}"/></div>
          <div class="form-group"><label>Last Name *</label><input type="text" id="m-elname" value="${data?.last_name||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Email *</label><input type="email" id="m-eemail" value="${data?.email||''}"/></div>
          <div class="form-group"><label>Phone</label><input type="text" id="m-ephone" value="${data?.phone||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Department</label><input type="text" id="m-edept" value="${data?.department||''}"/></div>
          <div class="form-group"><label>Job Title</label><input type="text" id="m-ejobtitle" value="${data?.job_title||''}"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Employee Number</label><input type="text" id="m-enum" value="${data?.employee_number||''}"/></div>
          <div class="form-group"><label>Hire Date</label><input type="date" id="m-ehire" value="${(data?.hire_date||new Date().toISOString()).split('T')[0]}"/></div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveEmployee('${data?.id||''}')"><i class="fas fa-save"></i> Save</button>
        </div>`,
    };
  },
  async saveEmployee(id) {
    const body = { first_name: $('m-efname').value, last_name: $('m-elname').value, email: $('m-eemail').value,
      phone: $('m-ephone').value, department: $('m-edept').value, job_title: $('m-ejobtitle').value,
      employee_number: $('m-enum').value, hire_date: $('m-ehire').value+'T00:00:00Z' };
    if (!body.first_name || !body.last_name) { Toast.error('First and last name are required'); return; }
    try {
      if (id) await API.put('/hr/employees/'+id, body);
      else await API.post('/hr/employees', body);
      Toast.success('Employee saved!'); App.closeModal(); HR.loadEmployees();
    } catch {}
  },

  leaveRequest(data) {
    return {
      title: 'New Leave Request',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Leave Type</label>
            <select id="m-lltype">
              <option value="annual">Annual Leave</option><option value="sick">Sick Leave</option>
              <option value="unpaid">Unpaid Leave</option><option value="maternity">Maternity</option><option value="paternity">Paternity</option>
            </select>
          </div>
          <div class="form-group"><label>Employee ID</label><input type="text" id="m-llempid" placeholder="emp-001"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Start Date *</label><input type="date" id="m-llstart" value="${new Date().toISOString().split('T')[0]}"/></div>
          <div class="form-group"><label>End Date *</label><input type="date" id="m-llend" value="${new Date().toISOString().split('T')[0]}"/></div>
        </div>
        <div class="form-group"><label>Reason</label><textarea id="m-llreason" placeholder="Reason for leave..."></textarea></div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveLeave()"><i class="fas fa-save"></i> Submit Request</button>
        </div>`,
    };
  },
  async saveLeave() {
    const body = { leave_type: $('m-lltype').value, employee_id: $('m-llempid').value,
      start_date: $('m-llstart').value+'T00:00:00Z', end_date: $('m-llend').value+'T00:00:00Z', reason: $('m-llreason').value };
    if (!body.employee_id) { Toast.error('Employee ID is required'); return; }
    try {
      await API.post('/hr/leaves', body);
      Toast.success('Leave request submitted!'); App.closeModal(); HR.loadLeaves();
    } catch {}
  },

  attendance() {
    return {
      title: 'Record Attendance',
      html: `
        <div class="form-group"><label>Employee ID *</label><input type="text" id="m-attempid" placeholder="emp-001"/></div>
        <div class="form-row">
          <div class="form-group"><label>Status</label>
            <select id="m-attstatus">
              <option value="present">Present</option><option value="absent">Absent</option>
              <option value="late">Late</option><option value="half_day">Half Day</option>
            </select>
          </div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveAttendance()"><i class="fas fa-save"></i> Record</button>
        </div>`,
    };
  },
  async saveAttendance() {
    const body = { employee_id: $('m-attempid').value, status: $('m-attstatus').value };
    if (!body.employee_id) { Toast.error('Employee ID is required'); return; }
    try {
      await API.post('/hr/attendance', body);
      Toast.success('Attendance recorded!'); App.closeModal(); HR.loadAttendance();
    } catch {}
  },

  payrollRun() {
    const now = new Date();
    return {
      title: 'Generate Payroll',
      html: `
        <div class="form-row">
          <div class="form-group"><label>Month</label>
            <select id="m-prmonth">
              ${['January','February','March','April','May','June','July','August','September','October','November','December']
                .map((m,i)=>`<option value="${i+1}"${now.getMonth()===i?' selected':''}>${m}</option>`).join('')}
            </select>
          </div>
          <div class="form-group"><label>Year</label><input type="number" id="m-pryear" value="${now.getFullYear()}"/></div>
        </div>
        <div class="form-group"><label>Notes</label><textarea id="m-prnotes" placeholder="Payroll notes..."></textarea></div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.savePayrollRun()"><i class="fas fa-cogs"></i> Generate</button>
        </div>`,
    };
  },
  async savePayrollRun() {
    const body = { month: parseInt($('m-prmonth').value), year: parseInt($('m-pryear').value), notes: $('m-prnotes').value };
    try {
      await API.post('/hr/payroll/generate', body);
      Toast.success('Payroll generated!'); App.closeModal(); HR.loadPayroll();
    } catch {}
  },

  stockMove() {
    return {
      title: 'Record Stock Move',
      html: `
        <div class="form-group"><label>Product ID *</label><input type="text" id="m-smprod" placeholder="Product ID"/></div>
        <div class="form-row">
          <div class="form-group"><label>From Location</label><input type="text" id="m-smfrom" placeholder="e.g. WH1/Input"/></div>
          <div class="form-group"><label>To Location *</label><input type="text" id="m-smto" placeholder="e.g. WH1/Stock"/></div>
        </div>
        <div class="form-row">
          <div class="form-group"><label>Quantity *</label><input type="number" id="m-smqty" value="1" min="0.01" step="0.01"/></div>
          <div class="form-group"><label>Reference</label><input type="text" id="m-smref" placeholder="SO-001"/></div>
        </div>
        <div class="form-actions">
          <button class="btn btn-outline" onclick="App.closeModal()">Cancel</button>
          <button class="btn btn-primary" onclick="Modals.saveStockMove()"><i class="fas fa-save"></i> Record Move</button>
        </div>`,
    };
  },
  async saveStockMove() {
    const body = { product_id: $('m-smprod').value, from_location: $('m-smfrom').value,
      to_location: $('m-smto').value, quantity: parseFloat($('m-smqty').value)||0,
      reference: $('m-smref').value };
    if (!body.product_id || !body.quantity) { Toast.error('Product and quantity are required'); return; }
    try {
      await API.post('/inventory/stock-moves', body);
      Toast.success('Stock move recorded!'); App.closeModal(); Inventory.loadMoves();
    } catch {}
  },
};

// ── Boot ─────────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => App.init());
