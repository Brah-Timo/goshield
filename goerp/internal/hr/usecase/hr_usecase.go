package usecase

import (
	"fmt"
	"time"

	"github.com/goerp/goerp/internal/hr/domain"
	"github.com/goerp/goerp/internal/hr/repository"
)

type HRUsecase struct {
	repo *repository.HRRepository
}

func NewHRUsecase(repo *repository.HRRepository) *HRUsecase {
	return &HRUsecase{repo: repo}
}

// ─── EMPLOYEES ────────────────────────────────────────────────────────────────

func (uc *HRUsecase) ListEmployees(filter domain.EmployeeFilter) ([]domain.Employee, int, error) {
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	return uc.repo.ListEmployees(filter)
}

func (uc *HRUsecase) GetEmployee(id, tenantID string) (*domain.Employee, error) {
	return uc.repo.GetEmployee(id, tenantID)
}

func (uc *HRUsecase) CreateEmployee(req domain.CreateEmployeeRequest, tenantID, createdBy string) (*domain.Employee, error) {
	if req.FirstName == "" || req.LastName == "" {
		return nil, fmt.Errorf("first name and last name are required")
	}
	if req.Email == "" {
		return nil, fmt.Errorf("employee email is required")
	}

	emp := &domain.Employee{
		TenantID:         tenantID,
		EmployeeNumber:   req.EmployeeNumber,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		FullName:         req.FirstName + " " + req.LastName,
		Email:            req.Email,
		Phone:            req.Phone,
		Department:       req.Department,
		JobTitle:         req.JobTitle,
		ManagerID:        req.ManagerID,
		Status:           domain.EmployeeStatusActive,
		HireDate:         req.HireDate,
		BirthDate:        req.BirthDate,
		NationalID:       req.NationalID,
		Address:          req.Address,
		EmergencyContact: req.EmergencyContact,
		CreatedBy:        createdBy,
	}

	if emp.HireDate.IsZero() {
		emp.HireDate = time.Now()
	}

	if err := uc.repo.CreateEmployee(emp); err != nil {
		return nil, fmt.Errorf("failed to create employee: %w", err)
	}
	return emp, nil
}

func (uc *HRUsecase) UpdateEmployee(id string, req domain.UpdateEmployeeRequest, tenantID string) (*domain.Employee, error) {
	emp, err := uc.repo.GetEmployee(id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}

	if req.FirstName != "" {
		emp.FirstName = req.FirstName
	}
	if req.LastName != "" {
		emp.LastName = req.LastName
	}
	emp.FullName = emp.FirstName + " " + emp.LastName
	if req.Email != "" {
		emp.Email = req.Email
	}
	if req.Phone != "" {
		emp.Phone = req.Phone
	}
	if req.Department != "" {
		emp.Department = req.Department
	}
	if req.JobTitle != "" {
		emp.JobTitle = req.JobTitle
	}
	if req.Status != "" {
		emp.Status = req.Status
	}
	if req.Address != "" {
		emp.Address = req.Address
	}
	if req.EmergencyContact != "" {
		emp.EmergencyContact = req.EmergencyContact
	}

	if err := uc.repo.UpdateEmployee(emp); err != nil {
		return nil, fmt.Errorf("failed to update employee: %w", err)
	}
	return emp, nil
}

// ─── CONTRACTS ────────────────────────────────────────────────────────────────

func (uc *HRUsecase) GetEmployeeContract(employeeID, tenantID string) (*domain.Contract, error) {
	return uc.repo.GetEmployeeContract(employeeID, tenantID)
}

func (uc *HRUsecase) CreateContract(req domain.CreateContractRequest, tenantID, createdBy string) (*domain.Contract, error) {
	if req.EmployeeID == "" {
		return nil, fmt.Errorf("employee ID is required")
	}
	if req.BasicSalary <= 0 {
		return nil, fmt.Errorf("basic salary must be greater than 0")
	}

	// Verify employee exists
	if _, err := uc.repo.GetEmployee(req.EmployeeID, tenantID); err != nil {
		return nil, fmt.Errorf("employee not found")
	}

	contract := &domain.Contract{
		TenantID:           tenantID,
		EmployeeID:         req.EmployeeID,
		ContractType:       req.ContractType,
		BasicSalary:        req.BasicSalary,
		HousingAllowance:   req.HousingAllowance,
		TransportAllowance: req.TransportAllowance,
		OtherAllowances:    req.OtherAllowances,
		Currency:           req.Currency,
		StartDate:          req.StartDate,
		EndDate:            req.EndDate,
		Notes:              req.Notes,
		IsActive:           true,
		CreatedBy:          createdBy,
	}

	if contract.Currency == "" {
		contract.Currency = "USD"
	}
	if contract.ContractType == "" {
		contract.ContractType = domain.ContractTypePermanent
	}

	contract.TotalSalary = contract.BasicSalary + contract.HousingAllowance +
		contract.TransportAllowance + contract.OtherAllowances

	if err := uc.repo.CreateContract(contract); err != nil {
		return nil, fmt.Errorf("failed to create contract: %w", err)
	}
	return contract, nil
}

// ─── ATTENDANCE ───────────────────────────────────────────────────────────────

func (uc *HRUsecase) ListAttendance(filter domain.AttendanceFilter) ([]domain.Attendance, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 31
	}
	return uc.repo.ListAttendance(filter)
}

func (uc *HRUsecase) RecordAttendance(employeeID, tenantID, status string, checkIn, checkOut *time.Time) (*domain.Attendance, error) {
	if employeeID == "" {
		return nil, fmt.Errorf("employee ID is required")
	}

	var workHours float64
	var overtime float64
	if checkIn != nil && checkOut != nil {
		diff := checkOut.Sub(*checkIn).Hours()
		workHours = diff
		if workHours > 8 {
			overtime = workHours - 8
			workHours = 8
		}
	}

	att := &domain.Attendance{
		TenantID:      tenantID,
		EmployeeID:    employeeID,
		Date:          time.Now(),
		CheckIn:       checkIn,
		CheckOut:      checkOut,
		WorkHours:     workHours,
		OvertimeHours: overtime,
		Status:        status,
	}
	if att.Status == "" {
		att.Status = "present"
	}

	if err := uc.repo.RecordAttendance(att); err != nil {
		return nil, fmt.Errorf("failed to record attendance: %w", err)
	}
	return att, nil
}

// ─── LEAVE REQUESTS ───────────────────────────────────────────────────────────

func (uc *HRUsecase) ListLeaveRequests(filter domain.LeaveFilter) ([]domain.LeaveRequest, int, error) {
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	return uc.repo.ListLeaveRequests(filter)
}

func (uc *HRUsecase) CreateLeaveRequest(req domain.CreateLeaveRequest, tenantID, createdBy string) (*domain.LeaveRequest, error) {
	if req.EmployeeID == "" {
		return nil, fmt.Errorf("employee ID is required")
	}
	if req.StartDate.After(req.EndDate) {
		return nil, fmt.Errorf("start date must be before end date")
	}

	// Calculate total days
	days := int(req.EndDate.Sub(req.StartDate).Hours()/24) + 1

	leave := &domain.LeaveRequest{
		TenantID:   tenantID,
		EmployeeID: req.EmployeeID,
		LeaveType:  req.LeaveType,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		TotalDays:  days,
		Reason:     req.Reason,
		Status:     domain.LeaveStatusPending,
		CreatedBy:  createdBy,
	}

	if leave.LeaveType == "" {
		leave.LeaveType = domain.LeaveTypeAnnual
	}

	if err := uc.repo.CreateLeaveRequest(leave); err != nil {
		return nil, fmt.Errorf("failed to create leave request: %w", err)
	}
	return leave, nil
}

func (uc *HRUsecase) ApproveLeave(id, tenantID, approvedBy string) error {
	return uc.repo.ApproveLeave(id, tenantID, approvedBy)
}

func (uc *HRUsecase) RejectLeave(id, tenantID, reason string) error {
	return uc.repo.RejectLeave(id, tenantID, reason)
}

// ─── PAYROLL ──────────────────────────────────────────────────────────────────

func (uc *HRUsecase) ListPayrollRuns(tenantID string) ([]domain.PayrollRun, error) {
	return uc.repo.ListPayrollRuns(tenantID)
}

func (uc *HRUsecase) GeneratePayroll(req domain.CreatePayrollRunRequest, tenantID, createdBy string) (*domain.PayrollRun, error) {
	if req.Month < 1 || req.Month > 12 {
		return nil, fmt.Errorf("invalid month: must be 1-12")
	}
	if req.Year < 2020 {
		return nil, fmt.Errorf("invalid year")
	}

	// Get all active employees
	emps, _, err := uc.repo.ListEmployees(domain.EmployeeFilter{
		TenantID: tenantID,
		Status:   domain.EmployeeStatusActive,
		Limit:    1000,
		Page:     1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch employees: %w", err)
	}

	var lines []domain.PayrollLine
	var totalGross, totalDeductions, totalNet float64

	for _, emp := range emps {
		contract, err := uc.repo.GetEmployeeContract(emp.ID, tenantID)
		if err != nil {
			continue // Skip if no active contract
		}

		gross := contract.BasicSalary + contract.HousingAllowance +
			contract.TransportAllowance + contract.OtherAllowances

		// Simple tax/SS calculation (9% social security, 15% income tax on net)
		ss := gross * 0.09
		tax := (gross - ss) * 0.15
		totalDed := ss + tax
		net := gross - totalDed

		lines = append(lines, domain.PayrollLine{
			EmployeeID:         emp.ID,
			EmployeeName:       emp.FullName,
			BasicSalary:        contract.BasicSalary,
			HousingAllowance:   contract.HousingAllowance,
			TransportAllowance: contract.TransportAllowance,
			OtherAllowances:    contract.OtherAllowances,
			GrossSalary:        gross,
			SocialSecurity:     ss,
			IncomeTax:          tax,
			TotalDeductions:    totalDed,
			NetSalary:          net,
			WorkDays:           22,
		})
		totalGross += gross
		totalDeductions += totalDed
		totalNet += net
	}

	run := &domain.PayrollRun{
		TenantID:        tenantID,
		Month:           req.Month,
		Year:            req.Year,
		Status:          domain.PayrollStatusDraft,
		TotalGross:      totalGross,
		TotalDeductions: totalDeductions,
		TotalNet:        totalNet,
		EmployeeCount:   len(emps),
		Notes:           req.Notes,
		CreatedBy:       createdBy,
	}

	_ = lines // lines calculated for totals; repo computes from contracts
	if err := uc.repo.CreatePayrollRun(run); err != nil {
		return nil, fmt.Errorf("failed to create payroll run: %w", err)
	}
	return run, nil
}

func (uc *HRUsecase) GetHRStats(tenantID string) (*domain.HRStats, error) {
	return uc.repo.GetHRStats(tenantID)
}
