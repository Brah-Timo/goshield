package domain

import "time"

// Employee statuses
const (
	EmployeeStatusActive     = "active"
	EmployeeStatusInactive   = "inactive"
	EmployeeStatusOnLeave    = "on_leave"
	EmployeeStatusTerminated = "terminated"
)

// Contract types
const (
	ContractTypePermanent = "permanent"
	ContractTypeFixed     = "fixed_term"
	ContractTypePartTime  = "part_time"
	ContractTypeFreelance = "freelance"
)

// Leave types
const (
	LeaveTypeAnnual  = "annual"
	LeaveTypeSick    = "sick"
	LeaveTypeUnpaid  = "unpaid"
	LeaveTypeMaternity = "maternity"
	LeaveTypePaternity = "paternity"
)

// Leave request statuses
const (
	LeaveStatusPending  = "pending"
	LeaveStatusApproved = "approved"
	LeaveStatusRejected = "rejected"
)

// Payroll statuses
const (
	PayrollStatusDraft     = "draft"
	PayrollStatusConfirmed = "confirmed"
	PayrollStatusPaid      = "paid"
)

// Employee represents a company employee
type Employee struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	EmployeeNumber string    `json:"employee_number"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	FullName       string    `json:"full_name"`
	Email          string    `json:"email"`
	Phone          string    `json:"phone"`
	Department     string    `json:"department"`
	JobTitle       string    `json:"job_title"`
	ManagerID      string    `json:"manager_id,omitempty"`
	Status         string    `json:"status"`
	HireDate       time.Time `json:"hire_date"`
	BirthDate      *time.Time `json:"birth_date,omitempty"`
	NationalID     string    `json:"national_id"`
	Address        string    `json:"address"`
	EmergencyContact string  `json:"emergency_contact"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Contract represents an employment contract
type Contract struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	EmployeeID   string     `json:"employee_id"`
	EmployeeName string     `json:"employee_name,omitempty"`
	ContractType string     `json:"contract_type"`
	BasicSalary  float64    `json:"basic_salary"`
	HousingAllowance float64 `json:"housing_allowance"`
	TransportAllowance float64 `json:"transport_allowance"`
	OtherAllowances float64 `json:"other_allowances"`
	TotalSalary  float64    `json:"total_salary"`
	Currency     string     `json:"currency"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	IsActive     bool       `json:"is_active"`
	Notes        string     `json:"notes"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// Attendance records employee check-in/out
type Attendance struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	EmployeeID  string     `json:"employee_id"`
	EmployeeName string    `json:"employee_name,omitempty"`
	Date        time.Time  `json:"date"`
	CheckIn     *time.Time `json:"check_in,omitempty"`
	CheckOut    *time.Time `json:"check_out,omitempty"`
	WorkHours   float64    `json:"work_hours"`
	OvertimeHours float64  `json:"overtime_hours"`
	Status      string     `json:"status"` // present, absent, late, half_day
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
}

// LeaveRequest represents an employee leave request
type LeaveRequest struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	EmployeeID   string     `json:"employee_id"`
	EmployeeName string     `json:"employee_name,omitempty"`
	LeaveType    string     `json:"leave_type"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      time.Time  `json:"end_date"`
	TotalDays    int        `json:"total_days"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"`
	ApprovedBy   string     `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	RejectionNote string    `json:"rejection_note,omitempty"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// PayrollRun represents a monthly payroll calculation
type PayrollRun struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	Month            int       `json:"month"`
	Year             int       `json:"year"`
	Status           string    `json:"status"`
	TotalGross       float64   `json:"total_gross"`
	TotalDeductions  float64   `json:"total_deductions"`
	TotalNet         float64   `json:"total_net"`
	EmployeeCount    int       `json:"employee_count"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	Notes            string    `json:"notes"`
	CreatedBy        string    `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// PayrollLine is one employee's payslip within a run
type PayrollLine struct {
	ID              string  `json:"id"`
	TenantID        string  `json:"tenant_id"`
	PayrollRunID    string  `json:"payroll_run_id"`
	EmployeeID      string  `json:"employee_id"`
	EmployeeName    string  `json:"employee_name"`
	BasicSalary     float64 `json:"basic_salary"`
	HousingAllowance float64 `json:"housing_allowance"`
	TransportAllowance float64 `json:"transport_allowance"`
	OtherAllowances float64 `json:"other_allowances"`
	GrossSalary     float64 `json:"gross_salary"`
	SocialSecurity  float64 `json:"social_security"`
	IncomeTax       float64 `json:"income_tax"`
	OtherDeductions float64 `json:"other_deductions"`
	TotalDeductions float64 `json:"total_deductions"`
	NetSalary       float64 `json:"net_salary"`
	WorkDays        int     `json:"work_days"`
	AbsentDays      int     `json:"absent_days"`
}

// Department represents an organizational unit
type Department struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Name       string    `json:"name"`
	Code       string    `json:"code"`
	ManagerID  string    `json:"manager_id,omitempty"`
	ParentID   string    `json:"parent_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// HR Stats
type HRStats struct {
	TotalEmployees  int     `json:"total_employees"`
	ActiveEmployees int     `json:"active_employees"`
	OnLeave         int     `json:"on_leave"`
	PendingLeaves   int     `json:"pending_leaves"`
	TotalPayroll    float64 `json:"total_payroll"`
	Departments     int     `json:"departments"`
}

// Filters
type EmployeeFilter struct {
	TenantID   string
	Status     string
	Department string
	Page       int
	Limit      int
}

type AttendanceFilter struct {
	TenantID   string
	EmployeeID string
	DateFrom   *time.Time
	DateTo     *time.Time
	Page       int
	Limit      int
}

type LeaveFilter struct {
	TenantID   string
	EmployeeID string
	Status     string
	Page       int
	Limit      int
}

// Request types
type CreateEmployeeRequest struct {
	EmployeeNumber   string     `json:"employee_number"`
	FirstName        string     `json:"first_name"`
	LastName         string     `json:"last_name"`
	Email            string     `json:"email"`
	Phone            string     `json:"phone"`
	Department       string     `json:"department"`
	JobTitle         string     `json:"job_title"`
	ManagerID        string     `json:"manager_id"`
	HireDate         time.Time  `json:"hire_date"`
	BirthDate        *time.Time `json:"birth_date"`
	NationalID       string     `json:"national_id"`
	Address          string     `json:"address"`
	EmergencyContact string     `json:"emergency_contact"`
}

type UpdateEmployeeRequest struct {
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	Email            string  `json:"email"`
	Phone            string  `json:"phone"`
	Department       string  `json:"department"`
	JobTitle         string  `json:"job_title"`
	ManagerID        string  `json:"manager_id"`
	Status           string  `json:"status"`
	Address          string  `json:"address"`
	EmergencyContact string  `json:"emergency_contact"`
}

type CreateContractRequest struct {
	EmployeeID         string     `json:"employee_id"`
	ContractType       string     `json:"contract_type"`
	BasicSalary        float64    `json:"basic_salary"`
	HousingAllowance   float64    `json:"housing_allowance"`
	TransportAllowance float64    `json:"transport_allowance"`
	OtherAllowances    float64    `json:"other_allowances"`
	Currency           string     `json:"currency"`
	StartDate          time.Time  `json:"start_date"`
	EndDate            *time.Time `json:"end_date"`
	Notes              string     `json:"notes"`
}

type CreateLeaveRequest struct {
	EmployeeID string    `json:"employee_id"`
	LeaveType  string    `json:"leave_type"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Reason     string    `json:"reason"`
}

type CreatePayrollRunRequest struct {
	Month int    `json:"month"`
	Year  int    `json:"year"`
	Notes string `json:"notes"`
}
