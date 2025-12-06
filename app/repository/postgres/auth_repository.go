package repository

import (
	"context"
	"database/sql"
	"sync"

	db "uas/database/postgres"
	model "uas/app/model/postgres"

	"errors"
	"github.com/lib/pq"
	"github.com/google/uuid"
)

// ================= IN-MEMORY BLACKLIST STORAGE =================
var blacklist map[string]bool
var mu sync.RWMutex

func init() {
	blacklist = make(map[string]bool)
}

// ================= INTERFACE =================
type AuthRepository interface {
	GetByUsername(ctx context.Context, username string) (*model.UserData, string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserData, error)
	BlacklistToken(ctx context.Context, jti string) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// ================= IMPLEMENTATION =================
type authRepo struct{}

func NewAuthRepository() AuthRepository {
	return &authRepo{}
}

// ================= METHODS =================

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
		pq.Array(&u.Permissions),
		&passwordHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.New("user tidak ditemukan")
		}
		return nil, "", errors.New("gagal mengambil user data")
	}

	return &u, passwordHash, nil
}

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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user tidak ditemukan")
		}
		return nil, errors.New("gagal mengambil user data")
	}

	return &u, nil
}

func (r *authRepo) BlacklistToken(ctx context.Context, jti string) error {
	mu.Lock()
	defer mu.Unlock()

	blacklist[jti] = true
	return nil
}

func (r *authRepo) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	mu.RLock()
	defer mu.RUnlock()

	_, found := blacklist[jti]
	return found, nil
}