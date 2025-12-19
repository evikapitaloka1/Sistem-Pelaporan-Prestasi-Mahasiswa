package middleware

import (
	"strings"

	"sistempelaporan/app/repository" // Perlu di-import untuk Blacklist

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	
)

var jwtKey = []byte("rahasia_negara_api")

func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Missing token"})
		}
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
		}
		
		
		if repository.IsTokenBlacklisted(tokenString) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Token has been revoked (logged out)"})
		}

		
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid token claims"})
		}

		
		c.Locals("user_id", claims["user_id"].(string)) 
		c.Locals("role", claims["role"])
		
		
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
        requestedID := c.Params("id") 
        currentUserID := c.Locals("user_id").(string) 
        role := c.Locals("role").(string)

        if strings.EqualFold(role, "Admin") {
            return c.Next()
        }

       
        if requestedID == currentUserID {
            return c.Next()
        }

       
        if strings.EqualFold(role, "Dosen Wali") {
            
            lecturerID, err := repository.GetLecturerIDByUserID(currentUserID)
            
            
            if err == nil && requestedID == lecturerID {
                return c.Next()
            }
        }
        
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "message": "Akses ditolak: Anda hanya diperbolehkan mengakses data milik Anda sendiri",
        })
    }
}

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
          
            if strings.EqualFold(role, "Dosen Wali") {
                
                lecturerID, err := repository.GetLecturerIDByUserID(currentUserID)
                if err != nil {
                    return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Akses ditolak: Data dosen tidak ditemukan"})
                }

                student, err := repository.GetStudentDetail(targetStudentID)
                if err == nil && student != nil {
                 
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