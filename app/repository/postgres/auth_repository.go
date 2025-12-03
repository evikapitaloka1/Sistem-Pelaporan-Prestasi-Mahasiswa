package repository

import (
	"context"
	"database/sql" 
	"sync" // Tambahkan untuk memastikan akses yang aman ke map
	
	db "uas/database/postgres"
	model "uas/app/model/postgres"
	
	"errors"
	"github.com/lib/pq"
	"github.com/google/uuid"
)

// ================= IN-MEMORY BLACKLIST STORAGE =================
// Map untuk menyimpan JTI yang diblacklist (key=jti string, value=true)
var blacklist map[string]bool 
var mu sync.RWMutex // Mutex untuk melindungi akses ke map (wajib karena global)

func init() {
    blacklist = make(map[string]bool)
}
// ================= INTERFACE =================
type AuthRepository interface {
	GetByUsername(ctx context.Context, username string) (*model.UserData, string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserData, error)
	
	// 🚫 Hapus: Metode Update telah dihapus
	
	// ✅ METODE REVOCATION
	BlacklistToken(ctx context.Context, jti string) error
    
    // ✅ METODE PENCEKALAN
    IsBlacklisted(ctx context.Context, jti string) (bool, error)
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
		// Logika standar untuk error: user tidak ditemukan, dll.
		if errors.Is(err, sql.ErrNoRows) {
            return nil, "", errors.New("user tidak ditemukan")
        }
		return nil, "", errors.New("gagal mengambil user data")
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
		if errors.Is(err, sql.ErrNoRows) {
            return nil, errors.New("user tidak ditemukan")
        }
		return nil, errors.New("gagal mengambil user data")
	}

	return &u, nil
}

// 🚫 Hapus: Implementasi Update dihapus.

// ✅ Implementasi BlacklistToken (Menggunakan In-Memory Map)
func (r *authRepo) BlacklistToken(ctx context.Context, jti string) error {
	mu.Lock()
	defer mu.Unlock()
    
    // Simpan JTI ke map global
	blacklist[jti] = true
	return nil 
}

// ✅ Implementasi IsBlacklisted (Memeriksa In-Memory Map)
func (r *authRepo) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
    mu.RLock() // Gunakan RLock untuk read-only access
    defer mu.RUnlock()
    
    // Cek apakah JTI ada di map
    _, found := blacklist[jti]
    return found, nil
}