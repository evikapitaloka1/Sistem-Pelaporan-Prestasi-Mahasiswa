package model

import (
	"github.com/google/uuid"
)

type RolePermission struct {

	RoleID       uuid.UUID `json:"roleId"`
	PermissionID uuid.UUID `json:"permissionId"`
}