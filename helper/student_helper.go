package helper

import (
    "context"
    "errors"
    "strings"
	"net/http"

    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
)

//
// =============== GENERIC TOKEN / RESPONSE HELPER ===============
//

func ExtractUserContext(c *fiber.Ctx) (context.Context, uuid.UUID, string, error) {

    userIDStr, ok := c.Locals("userID").(string)
    if !ok || userIDStr == "" {
        return nil, uuid.Nil, "", errors.New("user ID token missing")
    }

    role, ok := c.Locals("role").(string)
    if !ok || role == "" {
        return nil, uuid.Nil, "", errors.New("user role token missing")
    }

    userID, err := uuid.Parse(userIDStr)
    if err != nil {
        return nil, uuid.Nil, "", errors.New("invalid user ID format")
    }

    return c.Context(), userID, role, nil
}

func ErrorResponse(c *fiber.Ctx, status int, err error) error {
    return c.Status(status).JSON(fiber.Map{
        "error": err.Error(),
    })
}

func ServiceResponse(c *fiber.Ctx, data interface{}, err error) error {
    if err != nil {
        if strings.Contains(err.Error(), "forbidden") {
            return ErrorResponse(c, fiber.StatusForbidden, err)
        }
        return ErrorResponse(c, fiber.StatusBadRequest, err)
    }
    return c.JSON(fiber.Map{"status": "success", "data": data})
}

//
// ===================== BUSINESS LOGIC HELPER =====================
//

// --- DETAIL STUDENT ---
func HandleStudentDetailAccess(c *fiber.Ctx, studentUserID uuid.UUID, requesterID uuid.UUID, requesterRole string) error {

	// Admin boleh akses semua
	if requesterRole == "admin" {
		return nil
	}

	// Mahasiswa hanya boleh akses dirinya sendiri
	if requesterRole == "student" {
		if requesterID != studentUserID {
			return JsonError(c, http.StatusForbidden, "forbidden: tidak boleh mengakses data mahasiswa lain")
		}
		return nil
	}

	// Dosen wali hanya boleh akses advisee
	if requesterRole == "lecturer" {
		// cek di service lecturer, jangan di helper
		return nil
	}

	return JsonError(c, http.StatusForbidden, "forbidden: role tidak dikenali")
}

// --- VALIDASI ACCES ACHIEVEMENT ---
func ValidateAchievementAccess(targetID string, requester uuid.UUID, role string) error {

    if targetID == requester.String() {
        return nil // Owner → OK
    }

    // mahasiswa TIDAK BOLEH akses punya orang lain
    if strings.EqualFold(role, "Mahasiswa") {
        return errors.New("akses ditolak: hanya boleh melihat prestasi sendiri")
    }

    // Admin/Dosen lanjut
    return nil
}
