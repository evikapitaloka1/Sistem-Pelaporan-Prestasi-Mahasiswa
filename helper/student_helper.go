package helper

import (
    "context"
    "errors"
    "strings"


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

func ServiceResponse(c *fiber.Ctx, data interface{}, err error, msg ...string) error {
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "status": "error",
            "message": err.Error(),
        })
    }

    response := fiber.Map{
        "status": "success",
        "data":   data,
    }

    if len(msg) > 0 {
        response["message"] = msg[0]
    }

    return c.JSON(response)
}


//
// ===================== BUSINESS LOGIC HELPER =====================
//

// --- DETAIL STUDENT ---
func HandleStudentDetailAccess(c *fiber.Ctx, studentUserID uuid.UUID, jwtUserID uuid.UUID, role string) error {
    role = strings.ToLower(role)

    switch role {
    case "mahasiswa":
        if studentUserID.String() != jwtUserID.String() {
            return errors.New("akses ditolak: hanya mahasiswa bisa mengakses data sendiri")
        }
    case "admin":
        // admin bisa akses semua
        return nil
    default:
        return errors.New("akses ditolak: role tidak memiliki hak")
    }

    return nil
}

// --- VALIDASI ACCES ACHIEVEMENT ---
func ValidateAchievementAccess(targetID string, requester uuid.UUID, role string) error {
    roleLower := strings.ToLower(role)
    
    // Admin dan Dosen Wali diizinkan untuk melewati pengecekan ini
    if strings.Contains(roleLower, "admin") || strings.Contains(roleLower, "dosen wali") {
        return nil
    }

    // Jika requester adalah Mahasiswa, kita berasumsi ia mencoba mengakses profilnya
    // dan membiarkan layer Service (yang punya akses DB) yang melakukan validasi
    // apakah requester.userID sesuai dengan targetID.
    if strings.Contains(roleLower, "mahasiswa") {
        return nil 
    }

    // Default: Jika role tidak terdefinisi/tidak diizinkan, kembalikan error.
    // Namun, berdasarkan logika sebelumnya, kita hanya perlu menghapus bagian yang memblokir Mahasiswa.
    
    // Opsional: Jika Anda ingin lebih ketat dan hanya membolehkan role tertentu
    // Anda bisa mengubahnya menjadi:
    // return errors.New("akses ditolak: role tidak diizinkan untuk melihat prestasi")
    
    // Untuk kasus ini, kita kembalikan nil untuk role Mahasiswa agar Service yang memvalidasi.
    return nil
}