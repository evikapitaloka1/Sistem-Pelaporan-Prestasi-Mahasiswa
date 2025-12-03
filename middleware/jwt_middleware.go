package middleware

import (
	"context" // ✅ Tambahkan context
	"fmt"
	"strings"
	"os" 
	"log" 
	
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// ✅ FIX: Update interface agar sesuai dengan implementasi repository (menerima context, mengembalikan error).
type TokenBlacklistChecker interface {
	// IsBlacklisted sekarang menerima context dan mengembalikan error
	IsBlacklisted(ctx context.Context, jti string) (bool, error) 
}

// CustomClaims mendefinisikan struktur claim yang diharapkan dalam JWT.
type CustomClaims struct {
	UserID string `json:"user_id"`
	Role string `json:"role"` 
	jwt.RegisteredClaims
}

// JWTMiddleware memverifikasi token JWT dari header Authorization.
// Menerima TokenBlacklistChecker untuk keamanan logout.
func JWTMiddleware(blacklistChecker TokenBlacklistChecker) fiber.Handler {
	
	jwtSecret := os.Getenv("SECRET_KEY")
	if jwtSecret == "" {
		panic("FATAL: SECRET_KEY environment variable not set.") 
	}
	secretKey := []byte(jwtSecret) 

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"error": "Akses ditolak: Missing Authorization Header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"error": "Akses ditolak: Format token tidak valid (Harus: Bearer <token>)",
			})
		}

		tokenString := parts[1]
		claims := &CustomClaims{}
		
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secretKey, nil 
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"error": fmt.Sprintf("Token tidak valid: %v", err),
			})
		}
		
		// --- LOGIKA PENCABUTAN TOKEN (BLACKLIST CHECK) ---
		
		jti := claims.RegisteredClaims.ID
		
		if jti == "" {
			log.Printf("DIAGNOSIS WARNING: JTI claim is empty in valid token for user: %s. Token cannot be blacklisted.", claims.UserID)
		}
		
		// 🛑 Pengecekan Blacklist (Wajib untuk Logout)
        // ✅ FIX: Panggil IsBlacklisted dengan Context dan tangani error
		if jti != "" {
            isBlacklisted, checkErr := blacklistChecker.IsBlacklisted(c.Context(), jti)
            
            if checkErr != nil {
                // Gagal menghubungi repository
                return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                    "code": 500,
                    "error": "Gagal memverifikasi status blacklist",
                })
            }
            
            if isBlacklisted {
                // ✅ JIKA DITEMUKAN DI BLACKLIST, AKSES DITOLAK
                return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                    "code": 401,
                    "error": "Token telah dicabut (Logged out)",
                })
            }
        }
		
		// 1. Validasi dan Konversi UserID (dari string ke uuid.UUID)
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"code": 401,
				"error": "User ID dalam token bukan format UUID yang valid",
			})
		}
		
		// 2. Role Mapping dan Normalisasi
		rawRole := strings.ToLower(claims.Role)
		var standardizedRole string
		
		if strings.Contains(rawRole, "admin") {
			standardizedRole = "admin"
		} else if strings.Contains(rawRole, "dosen wali") || strings.Contains(rawRole, "verifikator") {
			standardizedRole = "dosen wali"
		} else if strings.Contains(rawRole, "mahasiswa") || strings.Contains(rawRole, "pelapor") {
			standardizedRole = "mahasiswa"
		} else {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"code": 403,
				"error": fmt.Sprintf("Role '%s' tidak dikenali oleh sistem prestasi.", claims.Role),
			})
		}
		
		// 3. Simpan data yang sudah distandardisasi
		c.Locals("userID", userID) 
		c.Locals("role", standardizedRole) 
		c.Locals("jti", jti) // Simpan JTI untuk digunakan oleh endpoint /logout
		
		return c.Next()
	}
}