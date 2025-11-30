package repository

import (
	"context"
	db "uas/database/postgres"
	model "uas/app/model/postgres"
	"errors"
	"github.com/lib/pq"
	"github.com/google/uuid"
)

// ================= INTERFACE =================
type AuthRepository interface {
	GetByUsername(ctx context.Context, username string) (*model.UserData, string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserData, error)
}

// ================= IMPLEMENTATION =================
type authRepo struct{}

// constructor
func NewAuthRepository() AuthRepository {
	return &authRepo{}
}

// ================= METHODS =================

// GetByUsername mengambil user berdasarkan username, mengembalikan user dan password hash
func (r *authRepo) GetByUsername(ctx context.Context, username string) (*model.UserData, string, error) {
	query := `
		SELECT u.id, u.username, u.full_name, r.name AS role_name,
		       ARRAY(
		         SELECT p.name
		         FROM permissions p
		         JOIN role_permissions rp ON rp.permission_id = p.id
		         WHERE rp.role_id = u.role_id
		       ) AS permissions,
		       u.password_hash
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.username=$1;
	`

	var u model.UserData
	var passwordHash string

	err := db.GetDB().QueryRowContext(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.FullName,
		&u.Role,
		pq.Array(&u.Permissions), // gunakan github.com/lib/pq untuk array
		&passwordHash,
	)
	if err != nil {
		return nil, "", errors.New("user tidak ditemukan")
	}

	return &u, passwordHash, nil
}

// GetByID mengambil user berdasarkan UUID
func (r *authRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.UserData, error) {
	query := `
		SELECT u.id, u.username, u.full_name, r.name AS role_name,
		       ARRAY(
		         SELECT p.name
		         FROM permissions p
		         JOIN role_permissions rp ON rp.permission_id = p.id
		         WHERE rp.role_id = u.role_id
		       ) AS permissions
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id=$1;
	`

	var u model.UserData

	err := db.GetDB().QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.FullName,
		&u.Role,
		pq.Array(&u.Permissions),
	)
	if err != nil {
		return nil, errors.New("user tidak ditemukan")
	}

	return &u, nil
}
