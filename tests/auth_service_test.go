package tests

import (
	"errors"
	"testing"
	"time"

	"sistempelaporan/app/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

/* ============================================================
   MOCK AUTH REPOSITORY
   ============================================================
*/

type MockAuthRepository struct {
	users     map[string]*model.User
	blacklist map[string]time.Time
}

func NewMockAuthRepository() *MockAuthRepository {
	return &MockAuthRepository{
		users:     make(map[string]*model.User),
		blacklist: make(map[string]time.Time),
	}
}

// Simulasi FindUserByUsername
func (m *MockAuthRepository) FindUserByUsername(username string) (*model.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

// Simulasi SetTokenBlacklist
func (m *MockAuthRepository) SetTokenBlacklist(token string, ttl time.Duration) error {
	m.blacklist[token] = time.Now().Add(ttl)
	return nil
}

/* ============================================================
   HELPER UNTUK TEST JWT
   ============================================================
*/

func getTestJWTSecret() []byte {
	return []byte("rahasia_negara_api")
}

/* ============================================================
   TEST CASES
   ============================================================
*/

func TestLogin_Logic(t *testing.T) {
	mockRepo := NewMockAuthRepository()
	
	// Setup: Buat user dummy dengan password ter-hash
	password := "rahasia123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	
	userID := uuid.New()
	mockRepo.users["evika"] = &model.User{
		ID:           userID,
		Username:     "evika",
		PasswordHash: string(hashedPassword),
		Role:         model.Role{Name: "Admin"},
	}

	t.Run("Success Login", func(t *testing.T) {
		// 1. Simulasi input
		inputUsername := "evika"
		inputPassword := "rahasia123"

		// 2. Logic: Cari User
		user, err := mockRepo.FindUserByUsername(inputUsername)
		if err != nil {
			t.Fatalf("Should find user, got error: %v", err)
		}

		// 3. Logic: Compare Password
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(inputPassword))
		if err != nil {
			t.Errorf("Password should match")
		}
	})

	t.Run("Wrong Password", func(t *testing.T) {
		inputPassword := "salah_password"
		user, _ := mockRepo.FindUserByUsername("evika")
		
		err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(inputPassword))
		if err == nil {
			t.Errorf("Should return error for wrong password")
		}
	})
}

func TestGenerateTokens_Logic(t *testing.T) {
	userID := uuid.New()
	user := &model.User{
		ID:   userID,
		Role: model.Role{Name: "Admin"},
	}
	perms := []string{"user:read_all", "report:create"}

	t.Run("Valid JWT Structure", func(t *testing.T) {
		// Simulasi generateTokens
		claims := jwt.MapClaims{
			"user_id":     user.ID.String(),
			"role":        user.Role.Name,
			"permissions": perms,
			"exp":         time.Now().Add(2 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(getTestJWTSecret())

		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}

		// Validasi isi token
		parsedToken, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return getTestJWTSecret(), nil
		})

		parsedClaims := parsedToken.Claims.(jwt.MapClaims)
		if parsedClaims["role"] != "Admin" {
			t.Errorf("Expected role Admin, got %v", parsedClaims["role"])
		}
	})
}

func TestLogout_BlacklistLogic(t *testing.T) {
	mockRepo := NewMockAuthRepository()
	dummyToken := "header.payload.signature"
	ttl := 1 * time.Hour

	t.Run("Add to Blacklist", func(t *testing.T) {
		err := mockRepo.SetTokenBlacklist(dummyToken, ttl)
		if err != nil {
			t.Errorf("Failed to blacklist token")
		}

		if _, ok := mockRepo.blacklist[dummyToken]; !ok {
			t.Errorf("Token should be in blacklist")
		}
	})
}