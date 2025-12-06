package repository

import (
	"context"
	"database/sql"
	"errors"

	db "uas/database/postgres"
	model "uas/app/model/postgres"

	"github.com/google/uuid"
)

type UserRepository interface {
	ListUsers(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	CreateUser(ctx context.Context, req model.CreateUserRequest) (uuid.UUID, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	UpdateUserRole(ctx context.Context, id uuid.UUID, roleID uuid.UUID) error
}

type userRepo struct{}

func NewUserRepository() UserRepository {
	return &userRepo{}
}

func (r *userRepo) ListUsers(ctx context.Context) ([]model.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.full_name,
			   u.role_id, r.name AS role_name, u.is_active,
			   u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		ORDER BY u.created_at DESC;
	`

	rows, err := db.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User

	for rows.Next() {
		var u model.User
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.PasswordHash,
			&u.FullName,
			&u.RoleID,
			&u.RoleName,
			&u.IsActive,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if createdAt.Valid {
			u.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			u.UpdatedAt = updatedAt.Time
		}

		users = append(users, u)
	}

	return users, nil
}

func (r *userRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.full_name,
			   u.role_id, r.name AS role_name, u.is_active,
			   u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1;
	`

	var u model.User
	var createdAt, updatedAt sql.NullTime

	err := db.GetDB().QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.RoleID,
		&u.RoleName,
		&u.IsActive,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		u.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		u.UpdatedAt = updatedAt.Time
	}

	return &u, nil
}

func (r *userRepo) CreateUser(ctx context.Context, req model.CreateUserRequest) (uuid.UUID, error) {
	query := `
		INSERT INTO users (username, email, password_hash, full_name, role_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
		RETURNING id;
	`

	var id uuid.UUID
	err := db.GetDB().QueryRowContext(ctx, query,
		req.Username,
		req.Email,
		req.Password,
		req.FullName,
		req.RoleID,
	).Scan(&id)

	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) error {
	query := `
		UPDATE users
		SET username=$1,
			email=$2,
			full_name=$3,
			updated_at=NOW()
		WHERE id=$4;
	`

	res, err := db.GetDB().ExecContext(ctx, query,
		req.Username,
		req.Email,
		req.FullName,
		id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("user tidak ditemukan atau tidak ada perubahan")
	}

	return nil
}

func (r *userRepo) DeleteUser(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id=$1;`
	_, err := db.GetDB().ExecContext(ctx, query, id)
	return err
}

func (r *userRepo) UpdateUserRole(ctx context.Context, id uuid.UUID, roleID uuid.UUID) error {
	query := `UPDATE users SET role_id=$1 WHERE id=$2;`
	_, err := db.GetDB().ExecContext(ctx, query, roleID, id)
	return err
}