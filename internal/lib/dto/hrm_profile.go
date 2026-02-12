package dto

type EditMyProfileRequest struct {
	Phone         *string `json:"phone"`
	InternalPhone *string `json:"internal_phone"`
	Email         *string `json:"email" validate:"omitempty,email"`
}
