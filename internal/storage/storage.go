package storage

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
	ErrRoleExists   = errors.New("role already exists")
)
