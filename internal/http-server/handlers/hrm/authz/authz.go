package authz

import (
	"context"

	"srmt-admin/internal/token"
)

// EmployeeAccessChecker provides methods for checking employee data access
type EmployeeAccessChecker interface {
	GetEmployeeIDByUserID(ctx context.Context, userID int64) (int64, error)
	IsManagerOf(ctx context.Context, managerEmployeeID, employeeID int64) (bool, error)
}

// AccessLevel represents the level of access a user has
type AccessLevel int

const (
	// AccessNone means no access
	AccessNone AccessLevel = iota
	// AccessSelf means user can access their own data
	AccessSelf
	// AccessSubordinate means user can access their subordinate's data
	AccessSubordinate
	// AccessDepartment means user can access department data (HR role)
	AccessDepartment
	// AccessAll means user can access all data (admin/hr role)
	AccessAll
)

// CanAccessEmployeeData checks if the user can access another employee's data
// Returns true if:
// 1. User is the employee themselves
// 2. User is a manager of the employee (directly or indirectly)
// 3. User has HR or admin role
func CanAccessEmployeeData(ctx context.Context, claims *token.Claims, targetEmployeeID int64, checker EmployeeAccessChecker) (bool, AccessLevel, error) {
	// Check for privileged roles first
	for _, role := range claims.Roles {
		if role == "admin" || role == "hr" {
			return true, AccessAll, nil
		}
	}

	// Get the current user's employee ID
	currentEmployeeID, err := checker.GetEmployeeIDByUserID(ctx, claims.UserID)
	if err != nil {
		// User might not be an employee (admin without employee record)
		return false, AccessNone, nil
	}

	// Check if user is accessing their own data
	if currentEmployeeID == targetEmployeeID {
		return true, AccessSelf, nil
	}

	// Check if user is a manager of the target employee
	isManager, err := checker.IsManagerOf(ctx, currentEmployeeID, targetEmployeeID)
	if err != nil {
		return false, AccessNone, err
	}

	if isManager {
		return true, AccessSubordinate, nil
	}

	return false, AccessNone, nil
}

// CanAccessSalaryData checks if the user can access salary data
// More restrictive than general employee data - requires HR role or being the employee themselves
func CanAccessSalaryData(ctx context.Context, claims *token.Claims, targetEmployeeID int64, checker EmployeeAccessChecker) (bool, error) {
	// Only HR and admin can see others' salary
	for _, role := range claims.Roles {
		if role == "admin" || role == "hr" {
			return true, nil
		}
	}

	// Get the current user's employee ID
	currentEmployeeID, err := checker.GetEmployeeIDByUserID(ctx, claims.UserID)
	if err != nil {
		return false, nil
	}

	// Users can only see their own salary
	return currentEmployeeID == targetEmployeeID, nil
}

// HasAnyRole checks if the user has any of the specified roles
func HasAnyRole(claims *token.Claims, roles ...string) bool {
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	for _, userRole := range claims.Roles {
		if _, ok := roleSet[userRole]; ok {
			return true
		}
	}
	return false
}
