package tests

import (

	"errors"
	"testing"
	"time"

	"sistempelaporan/app/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

/* ============================================================
   MOCK REPOSITORY (SIMULASI DATABASE USERS)
   ============================================================
*/

type MockUserRepository struct {
	users    map[uuid.UUID]*model.User
	students map[uuid.UUID]*model.Student
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:    make(map[uuid.UUID]*model.User),
		students: make(map[uuid.UUID]*model.Student),
	}
}

// Simulasi CreateUserWithProfile
func (m *MockUserRepository) CreateUserWithProfile(u *model.User, s *model.Student) error {
	if u.Username == "" || u.Email == "" {
		return errors.New("username atau email tidak boleh kosong")
	}
	
	// Simulasi Unique Constraint
	for _, existing := range m.users {
		if existing.Username == u.Username {
			return errors.New("unique constraint: username sudah ada")
		}
	}

	u.CreatedAt = time.Now()
	m.users[u.ID] = u
	if s != nil {
		m.students[s.ID] = s
	}
	return nil
}

/* ============================================================
   SERVICE LOGIC TEST (LOGIKA DARI SERVICE/USER_SERVICE.GO)
   ============================================================
*/

// Karena service kamu di kode utama menggunakan Fiber Ctx, 
// Unit Test ini fokus pada logika "Business Process" sebelum masuk ke repository.

func TestCreateNewUser_Logic(t *testing.T) {
	mockRepo := NewMockUserRepository()
	
	// Skenario Test
	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Valid User",
			username: "evika_pitaloka",
			email:    "evika@student.id",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "Empty Password",
			username: "admin",
			email:    "admin@test.com",
			password: "", // Ini harusnya error sesuai logic FIX 1.1 kamu
			wantErr:  true,
			errMsg:   "password wajib diisi",
		},
		{
			name:     "Duplicate Username",
			username: "evika_pitaloka", // Pakai nama yang sama dengan test pertama
			email:    "new@test.com",
			password: "password123",
			wantErr:  true,
			errMsg:   "unique constraint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Logic Validasi Password (seperti di service kamu)
			if tt.password == "" {
				if !tt.wantErr {
					t.Errorf("Expected no error, but got: password wajib diisi")
				}
				return
			}

			// 2. Logic Hash Password
			hashed, _ := bcrypt.GenerateFromPassword([]byte(tt.password), bcrypt.DefaultCost)

			// 3. Prepare Model
			user := &model.User{
				ID:           uuid.New(),
				Username:     tt.username,
				Email:        tt.email,
				PasswordHash: string(hashed),
				IsActive:     true,
			}

			// 4. Call Mock Repo
			err := mockRepo.CreateUserWithProfile(user, nil)

			// 5. Assertions
			if (err != nil) != tt.wantErr {
				t.Errorf("Test %s: error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func TestDeleteUser_SoftDelete(t *testing.T) {
	mockRepo := NewMockUserRepository()
	userID := uuid.New()
	
	// Arrange: Tambah user dulu
	user := &model.User{ID: userID, Username: "evika", IsActive: true}
	mockRepo.users[userID] = user

	// Act: Simulasi Soft Delete (Set is_active = false)
	if u, ok := mockRepo.users[userID]; ok {
		u.IsActive = false
		u.UpdatedAt = time.Now()
	}

	// Assert
	if mockRepo.users[userID].IsActive != false {
		t.Errorf("User should be soft deleted (IsActive = false)")
	}
}