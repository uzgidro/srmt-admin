package dto

type GetAllUsersFilters struct {
	OrganizationID *int64
	DepartmentID   *int64
	IsActive       *bool
}

// EditUserRequest - DTO для обновления пользователя
// (Обновляет *только* таблицу users)
type EditUserRequest struct {
	Login    *string
	IsActive *bool
	RoleIDs  []int64 // Optional: if provided, replaces all user roles with this list
	// (Пароль передается отдельно)
}
