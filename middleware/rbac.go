package middleware

import (
	"strings"

	"sistempelaporan/app/repository" // Perlu di-import untuk Blacklist

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	
)

// Gunakan key yang SAMA dengan di auth_service.go
// Idealnya ambil dari os.Getenv("JWT_SECRET")
var jwtKey = []byte("rahasia_negara_api")

// =======================================================
// 1. AuthMiddleware: Memvalidasi Token JWT + Blacklist Check
// =======================================================
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Ambil Token
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Missing token"})
		}
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		// 2. Parse & Validasi Tanda Tangan/Expiration
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
		}
		
		// --- TAMBAHAN PENTING: CEK BLACKLIST ---
		if repository.IsTokenBlacklisted(tokenString) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Token has been revoked (logged out)"})
		}
		// ----------------------------------------


		// 3. Ambil data (Claims) dari dalam token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid token claims"})
		}

		// 4. SIMPAN DATA USER KE CONTEXT (untuk service/controller)
		c.Locals("user_id", claims["user_id"].(string)) // Konversi ke string
		c.Locals("role", claims["role"])
		
		// Parsing permissions
		var permissions []string
		if permsRaw, ok := claims["permissions"].([]interface{}); ok {
			for _, p := range permsRaw {
				permissions = append(permissions, p.(string))
			}
		}
		c.Locals("permissions", permissions)

		return c.Next()
	}
}

// =======================================================
// 2. PermissionMiddleware: Mengecek hak akses (RBAC)
// =======================================================
func CheckPermission(requiredPerm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userPerms, ok := c.Locals("permissions").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "No permissions found"})
		}

		hasPermission := false
		for _, p := range userPerms {
			if p == requiredPerm {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"message": "Forbidden: You don't have permission " + requiredPerm,
			})
		}

		return c.Next()
	}
}
func CanAccessSelf() fiber.Handler {
    return func(c *fiber.Ctx) error {
        requestedID := c.Params("id") // ID dari URL (Lecturer ID: 23506891...)
        currentUserID := c.Locals("user_id").(string) // ID dari Token (User ID: 64d1bd23...)
        role := c.Locals("role").(string)

        // 1. Admin selalu bebas akses
        if strings.EqualFold(role, "Admin") {
            return c.Next()
        }

        // 2. Cek jika ID di URL sama dengan User ID (untuk Mahasiswa/User umum)
        if requestedID == currentUserID {
            return c.Next()
        }

        // 3. KHUSUS DOSEN WALI: Cek relasi User ID ke Lecturer ID
        if strings.EqualFold(role, "Dosen Wali") {
            // Kita cari tahu: "Siapa Lecturer ID milik User yang sedang login ini?"
            lecturerID, err := repository.GetLecturerIDByUserID(currentUserID)
            
            // Jika Lecturer ID hasil query sama dengan ID yang diminta di URL, beri akses
            if err == nil && requestedID == lecturerID {
                return c.Next()
            }
        }
        
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "message": "Akses ditolak: Anda hanya diperbolehkan mengakses data milik Anda sendiri",
        })
    }
}
// =======================================================
// 3B. AuthorizeResource: Middleware Otorisasi Relasional 
// =======================================================
func AuthorizeResource(mode string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        resourceID := c.Params("id")
        currentUserID := c.Locals("user_id").(string)
        role := c.Locals("role").(string)

        if strings.EqualFold(role, "Admin") { return c.Next() }

        var targetStudentID string
        ach, err := repository.FindAchievementByID(resourceID)
        if err == nil && ach != nil {
            targetStudentID = ach.StudentID.String()
        } else {
            targetStudentID = resourceID
        }

        if mode == "student_read" {
            // PERBAIKAN UNTUK DOSEN WALI
            if strings.EqualFold(role, "Dosen Wali") {
                // 1. Ambil ID Lecturer berdasarkan User ID yang sedang login
                lecturerID, err := repository.GetLecturerIDByUserID(currentUserID)
                if err != nil {
                    return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Akses ditolak: Data dosen tidak ditemukan"})
                }

                student, err := repository.GetStudentDetail(targetStudentID)
                if err == nil && student != nil {
                    // 2. Bandingkan ID Lecturer (bukan User ID)
                    if repository.ExtractAdvisorID(student) == lecturerID {
                        return c.Next()
                    }
                }
            }
            
            if strings.EqualFold(role, "Mahasiswa") {
                mhsID, _ := repository.GetStudentIDByUserID(currentUserID)
                if mhsID == targetStudentID {
                    return c.Next()
                }
            }
        }

        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "message": "Akses ditolak: Anda tidak memiliki hak untuk mengakses resource ini",
        })
    }
}