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
	// (Пароль передается отдельно)
}
