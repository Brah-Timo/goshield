package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goerp/goerp/internal/hr/domain"
	"github.com/goerp/goerp/internal/shared/database"
)

type HRRepository struct {
	db *database.DB
}

func NewHRRepository(db *database.DB) *HRRepository {
	return &HRRepository{db: db}
}

// ---- helpers ----------------------------------------------------------------

func hrParseTime(s string) time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Now()
}

func hrParseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := hrParseTime(s)
	return &t
}

func hrNullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ─── EMPLOYEES ────────────────────────────────────────────────────────────────

func (r *HRRepository) ListEmployees(filter domain.EmployeeFilter) ([]domain.Employee, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	query := `SELECT e.id, e.tenant_id, COALESCE(e.employee_number,''),
		e.first_name, e.last_name,
		(e.first_name || ' ' || e.last_name) AS full_name,
		COALESCE(e.email,''), COALESCE(e.phone,''),
		COALESCE(e.department,''), COALESCE(e.job_title,''),
		COALESCE(e.manager_id,''),
		CASE WHEN e.is_active=1 THEN 'active' ELSE 'inactive' END,
		COALESCE(e.hire_date, date('now')),
		COALESCE(e.national_id,''), COALESCE(e.address,''),
		COALESCE(e.emergency_contact,''),
		COALESCE(e.created_by,''), e.created_at, e.updated_at
		FROM employees e WHERE e.tenant_id=?`
	args := []interface{}{filter.TenantID}

	if filter.Status == "active" {
		query += " AND e.is_active=1"
	} else if filter.Status == "inactive" {
		query += " AND e.is_active=0"
	}
	if filter.Department != "" {
		query += " AND e.department=?"
		args = append(args, filter.Department)
	}

	query += " ORDER BY e.first_name ASC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var emps []domain.Employee
	for rows.Next() {
		var e domain.Employee
		var hireDateStr, createdStr, updatedStr string
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.EmployeeNumber,
			&e.FirstName, &e.LastName, &e.FullName,
			&e.Email, &e.Phone, &e.Department, &e.JobTitle,
			&e.ManagerID, &e.Status, &hireDateStr,
			&e.NationalID, &e.Address, &e.EmergencyContact,
			&e.CreatedBy, &createdStr, &updatedStr,
		); err != nil {
			continue
		}
		e.HireDate = hrParseTime(hireDateStr)
		e.CreatedAt = hrParseTime(createdStr)
		e.UpdatedAt = hrParseTime(updatedStr)
		emps = append(emps, e)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM employees WHERE tenant_id=?`, filter.TenantID).Scan(&total)
	return emps, total, nil
}

func (r *HRRepository) GetEmployee(id, tenantID string) (*domain.Employee, error) {
	var e domain.Employee
	var hireDateStr, createdStr, updatedStr string
	err := r.db.QueryRow(`
		SELECT id, tenant_id, COALESCE(employee_number,''),
		       first_name, last_name,
		       (first_name || ' ' || last_name) AS full_name,
		       COALESCE(email,''), COALESCE(phone,''),
		       COALESCE(department,''), COALESCE(job_title,''),
		       COALESCE(manager_id,''),
		       CASE WHEN is_active=1 THEN 'active' ELSE 'inactive' END,
		       COALESCE(hire_date, date('now')),
		       COALESCE(national_id,''), COALESCE(address,''),
		       COALESCE(emergency_contact,''),
		       COALESCE(created_by,''), created_at, updated_at
		FROM employees WHERE id=? AND tenant_id=?`, id, tenantID).Scan(
		&e.ID, &e.TenantID, &e.EmployeeNumber,
		&e.FirstName, &e.LastName, &e.FullName,
		&e.Email, &e.Phone, &e.Department, &e.JobTitle,
		&e.ManagerID, &e.Status, &hireDateStr,
		&e.NationalID, &e.Address, &e.EmergencyContact,
		&e.CreatedBy, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}
	e.HireDate = hrParseTime(hireDateStr)
	e.CreatedAt = hrParseTime(createdStr)
	e.UpdatedAt = hrParseTime(updatedStr)
	return &e, nil
}

func (r *HRRepository) CreateEmployee(e *domain.Employee) error {
	e.ID = uuid.New().String()
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()

	hireDate := e.HireDate.Format("2006-01-02")
	if e.HireDate.IsZero() {
		hireDate = time.Now().Format("2006-01-02")
	}

	_, err := r.db.Exec(`
		INSERT INTO employees
		  (id, tenant_id, employee_number, first_name, last_name,
		   email, phone, department, job_title, hire_date,
		   national_id, address, emergency_contact, is_active,
		   created_by, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?,?)`,
		e.ID, e.TenantID, e.EmployeeNumber,
		e.FirstName, e.LastName,
		e.Email, e.Phone, e.Department, e.JobTitle, hireDate,
		e.NationalID, e.Address, e.EmergencyContact,
		e.CreatedBy,
		e.CreatedAt.Format(time.RFC3339),
		e.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *HRRepository) UpdateEmployee(e *domain.Employee) error {
	e.UpdatedAt = time.Now()
	isActive := 1
	if e.Status == "inactive" || e.Status == "terminated" {
		isActive = 0
	}
	_, err := r.db.Exec(`
		UPDATE employees SET
		       first_name=?, last_name=?, email=?, phone=?,
		       department=?, job_title=?, address=?, emergency_contact=?,
		       is_active=?, updated_at=?
		WHERE id=? AND tenant_id=?`,
		e.FirstName, e.LastName, e.Email, e.Phone,
		e.Department, e.JobTitle, e.Address, e.EmergencyContact,
		isActive, e.UpdatedAt.Format(time.RFC3339),
		e.ID, e.TenantID,
	)
	return err
}

// ─── CONTRACTS ────────────────────────────────────────────────────────────────

func (r *HRRepository) GetEmployeeContract(employeeID, tenantID string) (*domain.Contract, error) {
	var c domain.Contract
	var endDateStr, createdStr, empName string
	var isActive int

	err := r.db.QueryRow(`
		SELECT c.id, c.tenant_id, c.employee_id,
		       (e.first_name || ' ' || e.last_name),
		       c.contract_type, c.basic_salary,
		       c.housing_allowance, c.transport_allowance, c.other_allowances,
		       (c.basic_salary + c.housing_allowance + c.transport_allowance + c.other_allowances),
		       c.currency, c.start_date, COALESCE(c.end_date,''),
		       c.is_active, COALESCE(c.notes,''), c.created_at
		FROM contracts c
		JOIN employees e ON e.id = c.employee_id
		WHERE c.employee_id=? AND c.tenant_id=? AND c.is_active=1
		ORDER BY c.created_at DESC LIMIT 1`,
		employeeID, tenantID,
	).Scan(
		&c.ID, &c.TenantID, &c.EmployeeID, &empName,
		&c.ContractType, &c.BasicSalary,
		&c.HousingAllowance, &c.TransportAllowance, &c.OtherAllowances, &c.TotalSalary,
		&c.Currency, &c.StartDate, &endDateStr,
		&isActive, &c.Notes, &createdStr,
	)
	if err != nil {
		return nil, err
	}
	c.EmployeeName = empName
	c.IsActive = isActive == 1
	c.CreatedAt = hrParseTime(createdStr)
	c.UpdatedAt = c.CreatedAt
	if endDateStr != "" {
		t := hrParseTime(endDateStr)
		c.EndDate = &t
	}
	return &c, nil
}

func (r *HRRepository) CreateContract(c *domain.Contract) error {
	c.ID = uuid.New().String()
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	c.TotalSalary = c.BasicSalary + c.HousingAllowance + c.TransportAllowance + c.OtherAllowances

	// Deactivate existing contracts
	_, _ = r.db.Exec(`UPDATE contracts SET is_active=0 WHERE employee_id=? AND tenant_id=?`,
		c.EmployeeID, c.TenantID)

	var endDate interface{}
	if c.EndDate != nil && !c.EndDate.IsZero() {
		endDate = c.EndDate.Format("2006-01-02")
	}

	startDate := ""
	if !c.StartDate.IsZero() {
		startDate = c.StartDate.Format("2006-01-02")
	} else {
		startDate = time.Now().Format("2006-01-02")
	}

	_, err := r.db.Exec(`
		INSERT INTO contracts
		  (id, tenant_id, employee_id, contract_type, basic_salary,
		   housing_allowance, transport_allowance, other_allowances,
		   currency, start_date, end_date, is_active, notes, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,1,?,?)`,
		c.ID, c.TenantID, c.EmployeeID, c.ContractType, c.BasicSalary,
		c.HousingAllowance, c.TransportAllowance, c.OtherAllowances,
		c.Currency, startDate, endDate,
		c.Notes, c.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// ─── ATTENDANCE ───────────────────────────────────────────────────────────────

func (r *HRRepository) ListAttendance(filter domain.AttendanceFilter) ([]domain.Attendance, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 31
	}
	offset := (filter.Page - 1) * filter.Limit

	query := `SELECT a.id, a.tenant_id, a.employee_id,
		(e.first_name || ' ' || e.last_name),
		COALESCE(a.date, date(a.check_in)),
		COALESCE(a.check_in,''), COALESCE(a.check_out,''),
		COALESCE(a.work_hours,0), COALESCE(a.overtime_hours,0),
		a.status, COALESCE(a.notes,''), a.created_at
		FROM attendance a
		JOIN employees e ON e.id = a.employee_id
		WHERE a.tenant_id=?`
	args := []interface{}{filter.TenantID}

	if filter.EmployeeID != "" {
		query += " AND a.employee_id=?"
		args = append(args, filter.EmployeeID)
	}
	if filter.DateFrom != nil {
		query += " AND a.date >= ?"
		args = append(args, filter.DateFrom.Format("2006-01-02"))
	}
	if filter.DateTo != nil {
		query += " AND a.date < ?"
		args = append(args, filter.DateTo.Format("2006-01-02"))
	}

	query += " ORDER BY a.date DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var atts []domain.Attendance
	for rows.Next() {
		var a domain.Attendance
		var dateStr, checkInStr, checkOutStr, createdStr string
		if err := rows.Scan(
			&a.ID, &a.TenantID, &a.EmployeeID, &a.EmployeeName,
			&dateStr, &checkInStr, &checkOutStr,
			&a.WorkHours, &a.OvertimeHours,
			&a.Status, &a.Notes, &createdStr,
		); err != nil {
			continue
		}
		a.Date = hrParseTime(dateStr)
		a.CreatedAt = hrParseTime(createdStr)
		if checkInStr != "" {
			t := hrParseTime(checkInStr)
			a.CheckIn = &t
		}
		if checkOutStr != "" {
			t := hrParseTime(checkOutStr)
			a.CheckOut = &t
		}
		atts = append(atts, a)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM attendance WHERE tenant_id=?`, filter.TenantID).Scan(&total)
	return atts, total, nil
}

func (r *HRRepository) RecordAttendance(a *domain.Attendance) error {
	a.ID = uuid.New().String()
	a.CreatedAt = time.Now()

	today := time.Now().Format("2006-01-02")

	var checkIn, checkOut interface{}
	if a.CheckIn != nil {
		checkIn = a.CheckIn.Format(time.RFC3339)
	}
	if a.CheckOut != nil {
		checkOut = a.CheckOut.Format(time.RFC3339)
	}

	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO attendance
		  (id, tenant_id, employee_id, date, check_in, check_out,
		   work_hours, overtime_hours, status, notes, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.TenantID, a.EmployeeID, today,
		checkIn, checkOut, a.WorkHours, a.OvertimeHours,
		a.Status, a.Notes,
		a.CreatedAt.Format(time.RFC3339),
	)
	return err
}

// ─── LEAVE REQUESTS ───────────────────────────────────────────────────────────

func (r *HRRepository) ListLeaveRequests(filter domain.LeaveFilter) ([]domain.LeaveRequest, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	query := `SELECT lr.id, lr.tenant_id, lr.employee_id,
		(e.first_name || ' ' || e.last_name),
		lr.leave_type, lr.start_date, lr.end_date,
		COALESCE(lr.days_count, 0),
		COALESCE(lr.reason,''), lr.state,
		COALESCE(lr.approved_by,''), COALESCE(lr.approved_at,''),
		'',
		lr.created_at, lr.created_at
		FROM leave_requests lr
		JOIN employees e ON e.id = lr.employee_id
		WHERE lr.tenant_id=?`
	args := []interface{}{filter.TenantID}

	if filter.EmployeeID != "" {
		query += " AND lr.employee_id=?"
		args = append(args, filter.EmployeeID)
	}
	if filter.Status != "" {
		query += " AND lr.state=?"
		args = append(args, filter.Status)
	}

	query += " ORDER BY lr.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var leaves []domain.LeaveRequest
	for rows.Next() {
		var lr domain.LeaveRequest
		var startStr, endStr, approvedAtStr, createdStr, updatedStr, approverName string
		if err := rows.Scan(
			&lr.ID, &lr.TenantID, &lr.EmployeeID, &lr.EmployeeName,
			&lr.LeaveType, &startStr, &endStr,
			&lr.TotalDays, &lr.Reason, &lr.Status,
			&lr.ApprovedBy, &approvedAtStr, &approverName,
			&createdStr, &updatedStr,
		); err != nil {
			continue
		}
		lr.StartDate = hrParseTime(startStr)
		lr.EndDate = hrParseTime(endStr)
		lr.CreatedAt = hrParseTime(createdStr)
		lr.UpdatedAt = hrParseTime(updatedStr)
		lr.ApprovedAt = hrParseTimePtr(approvedAtStr)
		leaves = append(leaves, lr)
	}

	var total int
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM leave_requests WHERE tenant_id=?`, filter.TenantID).Scan(&total)
	return leaves, total, nil
}

func (r *HRRepository) CreateLeaveRequest(lr *domain.LeaveRequest) error {
	lr.ID = uuid.New().String()
	lr.CreatedAt = time.Now()
	lr.UpdatedAt = time.Now()
	if lr.Status == "" {
		lr.Status = domain.LeaveStatusPending
	}

	_, err := r.db.Exec(`
		INSERT INTO leave_requests
		  (id, tenant_id, employee_id, leave_type, start_date, end_date,
		   days_count, reason, state, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		lr.ID, lr.TenantID, lr.EmployeeID, lr.LeaveType,
		lr.StartDate.Format("2006-01-02"),
		lr.EndDate.Format("2006-01-02"),
		lr.TotalDays, lr.Reason, lr.Status,
		lr.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *HRRepository) ApproveLeave(id, tenantID, approvedBy string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`
		UPDATE leave_requests SET state='approved', approved_by=?, approved_at=?
		WHERE id=? AND tenant_id=? AND state='pending'`,
		approvedBy, now, id, tenantID,
	)
	return err
}

func (r *HRRepository) RejectLeave(id, tenantID, reason string) error {
	_, err := r.db.Exec(`
		UPDATE leave_requests SET state='refused'
		WHERE id=? AND tenant_id=?`, id, tenantID,
	)
	return err
}

// ─── PAYROLL ──────────────────────────────────────────────────────────────────

func (r *HRRepository) ListPayrollRuns(tenantID string) ([]domain.PayrollRun, error) {
	rows, err := r.db.Query(`
		SELECT id, tenant_id,
		       CAST(strftime('%m', period_start) AS INTEGER),
		       CAST(strftime('%Y', period_start) AS INTEGER),
		       state, total_gross, total_deductions, total_net,
		       employee_count, COALESCE(processed_at,''), COALESCE(notes,''),
		       COALESCE(processed_by,''), created_at, created_at
		FROM payroll_runs WHERE tenant_id=?
		ORDER BY period_start DESC LIMIT 24`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []domain.PayrollRun
	for rows.Next() {
		var pr domain.PayrollRun
		var paidAtStr, createdStr, updatedStr string
		if err := rows.Scan(
			&pr.ID, &pr.TenantID,
			&pr.Month, &pr.Year, &pr.Status,
			&pr.TotalGross, &pr.TotalDeductions, &pr.TotalNet,
			&pr.EmployeeCount, &paidAtStr, &pr.Notes,
			&pr.CreatedBy, &createdStr, &updatedStr,
		); err != nil {
			continue
		}
		pr.CreatedAt = hrParseTime(createdStr)
		pr.UpdatedAt = hrParseTime(updatedStr)
		pr.PaidAt = hrParseTimePtr(paidAtStr)
		runs = append(runs, pr)
	}
	return runs, nil
}

func (r *HRRepository) CreatePayrollRun(pr *domain.PayrollRun) error {
	pr.ID = uuid.New().String()
	pr.CreatedAt = time.Now()
	pr.UpdatedAt = time.Now()

	// Calculate totals from employee contracts in this tenant
	var totalGross, totalNet, totalDed float64
	var empCount int

	empRows, err := r.db.Query(`
		SELECT c.basic_salary + c.housing_allowance + c.transport_allowance + c.other_allowances
		FROM contracts c
		JOIN employees e ON e.id = c.employee_id
		WHERE c.tenant_id=? AND c.is_active=1 AND e.is_active=1`,
		pr.TenantID,
	)
	if err == nil {
		defer empRows.Close()
		for empRows.Next() {
			var gross float64
			empRows.Scan(&gross)
			ss := gross * 0.09
			tax := (gross - ss) * 0.15
			ded := ss + tax
			totalGross += gross
			totalDed += ded
			totalNet += gross - ded
			empCount++
		}
	}

	if pr.TotalGross > 0 {
		totalGross = pr.TotalGross
		totalDed = pr.TotalDeductions
		totalNet = pr.TotalNet
		empCount = pr.EmployeeCount
	}

	periodStart := fmt.Sprintf("%04d-%02d-01", pr.Year, pr.Month)

	_, insertErr := r.db.Exec(`
		INSERT INTO payroll_runs
		  (id, tenant_id, period_start, state,
		   total_gross, total_deductions, total_net,
		   employee_count, notes, created_by, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		pr.ID, pr.TenantID, periodStart, pr.Status,
		totalGross, totalDed, totalNet,
		empCount, pr.Notes, pr.CreatedBy,
		pr.CreatedAt.Format(time.RFC3339),
	)
	if insertErr == nil {
		pr.TotalGross = totalGross
		pr.TotalDeductions = totalDed
		pr.TotalNet = totalNet
		pr.EmployeeCount = empCount
	}
	return insertErr
}

// ─── HR STATS ─────────────────────────────────────────────────────────────────

func (r *HRRepository) GetHRStats(tenantID string) (*domain.HRStats, error) {
	stats := &domain.HRStats{}

	_ = r.db.QueryRow(`SELECT COUNT(*) FROM employees WHERE tenant_id=? AND is_active=1`, tenantID).Scan(&stats.TotalEmployees)
	stats.ActiveEmployees = stats.TotalEmployees
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM leave_requests WHERE tenant_id=? AND state='approved'`, tenantID).Scan(&stats.OnLeave)
	_ = r.db.QueryRow(`SELECT COUNT(*) FROM leave_requests WHERE tenant_id=? AND state='pending'`, tenantID).Scan(&stats.PendingLeaves)
	_ = r.db.QueryRow(`
		SELECT COALESCE(SUM(basic_salary + housing_allowance + transport_allowance + other_allowances),0)
		FROM contracts c JOIN employees e ON e.id=c.employee_id
		WHERE c.tenant_id=? AND c.is_active=1 AND e.is_active=1`, tenantID).Scan(&stats.TotalPayroll)
	_ = r.db.QueryRow(`SELECT COUNT(DISTINCT COALESCE(department,'General')) FROM employees WHERE tenant_id=? AND is_active=1`, tenantID).Scan(&stats.Departments)

	return stats, nil
}
