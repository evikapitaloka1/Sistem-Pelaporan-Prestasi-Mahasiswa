package routes

import (
	"net/http"
	"strings"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"log"
	mongo "uas/app/service/mongo"
	authService "uas/app/service/postgres"
	studentService "uas/app/service/postgres"
	mw "uas/middleware" // Asumsi ini adalah lokasi RBAC
)

// StudentRoutes setup semua route terkait student
func StudentRoutes(
	api fiber.Router,
	authSvc *authService.AuthService,
	studentSvc studentService.StudentService,
	achievementSvc mongo.AchievementService,
	jwtMiddleware fiber.Handler, // Menerima handler yang sudah siap
) {
	// Buat RBAC Middleware. PENTING: Middleware ini harus dimodifikasi untuk
	// memungkinkan Superadmin melewati pengecekan izin umum (akan dijelaskan di rbac.go)
	
	rbacUpdate := mw.RBACMiddleware("student:update", authSvc)

	// Grup Route menggunakan jwtMiddleware yang sudah di-inject
	students := api.Group("/students", jwtMiddleware) // Gunakan handler yang di-inject

	// =========================================================================================
	// 1. GET list (Hanya Admin yang boleh, menggunakan rbacRead yang dimodifikasi atau rbacManage)
	// =========================================================================================
	// Catatan: Jika student:read hanya untuk admin, gunakan rbacRead yang dimodifikasi
	// agar admin dengan user:manage bisa mengakses ini.
	students.Get("/", func(c *fiber.Ctx) error {

    // ===========================================================
    // Ambil userID & role dari JWT (Locals sudah diset di middleware)
    // ===========================================================
    userIDStr, ok := c.Locals("userID").(string)
    userRole, _ := c.Locals("role").(string)

    if !ok || strings.TrimSpace(userIDStr) == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "user ID missing or invalid from token",
        })
    }

    // Parse UUID
    requestingUserID, err := uuid.Parse(strings.TrimSpace(userIDStr))
    if err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "invalid user ID format in token",
        })
    }

        // 2. Panggil Service dengan 3 argumen baru
        result, err := studentSvc.ListStudents(c.Context(), requestingUserID, userRole) 
        // -------------------------------------------------------------------------------------------------------

        if err != nil {
            // Handle error otorisasi dari Service (misal: "unauthorized: only students...")
            if strings.Contains(err.Error(), "unauthorized") {
                return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
            }
            return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
        }
        return c.JSON(fiber.Map{"status": "success", "data": result})
    })

	// =========================================================================================
	// 2. GET detail /:id (HANYA UNTUK DIRI SENDIRI atau ADMIN)
	//    Logika pengecekan kepemilikan dipindahkan ke dalam handler.
	// =========================================================================================
	// Kita masih menggunakan rbacRead di sini, TAPI middleware RBAC harus mengizinkan
	// akses jika targetID (param) = UserID (locals) ATAU jika user punya user:manage.
	students.Get("/:id", func(c *fiber.Ctx) error {
    // Ambil student ID dari URL
    studentID := strings.TrimSpace(c.Params("id"))

    // Ambil userID dari JWT
    currentUserID, ok := c.Locals("userID").(string)
    if !ok || currentUserID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "User ID missing from token",
        })
    }

    // Ambil role dari JWT
    roleRaw := c.Locals("role")
    if roleRaw == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Role missing from token",
        })
    }
    role := strings.ToLower(roleRaw.(string))

    // ===========================
    // Ambil data student dari DB
    // ===========================
    student, err := studentSvc.GetStudentDetail(c.Context(), studentID)
    if err != nil {
        return c.Status(404).JSON(fiber.Map{
            "error": "Student tidak ditemukan",
        })
    }

    // ===========================
    // PERBANDINGAN AKSES
    // student.UserID → UUID user pemilik student
    // ===========================
    isOwner := strings.EqualFold(student.UserID.String(), currentUserID)
    isAdmin := role == "admin"

    log.Printf("STUDENT.USER_ID : '%s'", student.UserID.String())
    log.Printf("CURRENT USER    : '%s'", currentUserID)

    if !isOwner && !isAdmin {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Akses ditolak. Anda hanya bisa melihat data Anda sendiri.",
        })
    }

    // ===========================
    // Jika lolos, return data
    // ===========================
    return c.JSON(fiber.Map{
        "status": "success",
        "data": student,
    })
})

	// =========================================================================================
	// 3. PUT update advisor (Hanya Admin yang boleh, menggunakan rbacUpdate yang dimodifikasi)
	// =========================================================================================
	students.Put("/:id/advisor", rbacUpdate, func(c *fiber.Ctx) error {
		studentID := c.Params("id")

		body := struct {
			NewAdvisorID string `json:"new_advisor_id"`
		}{}

		if err := c.BodyParser(&body); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		role := c.Locals("role").(string)

		err := studentSvc.UpdateAdvisor(c.Context(), studentID, body.NewAdvisorID, role)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"status": "success"})
	})

	// =========================================================================================
	// 4. GET achievements by student ID (HANYA UNTUK DIRI SENDIRI atau ADMIN/DOSEN)
	// =========================================================================================
	// ... (Bagian atas kode Anda tetap sama)

// =========================================================================================
// 4. GET achievements by student ID (HANYA UNTUK DIRI SENDIRI atau ADMIN/DOSEN)
//    🛑 PENTING: Hapus rbacRead dari middleware list. Otorisasi dilakukan di dalam handler.
// =========================================================================================
students.Get("/:id/achievements", func(c *fiber.Ctx) error { // <- rbacRead DIHAPUS
    targetStudentID := c.Params("id")
    if targetStudentID == "" {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Student ID is required",
        })
    }

    // --- PENGAMBILAN DAN VALIDASI USER ID DARI TOKEN LOCALS ---
    userIDStr, ok := c.Locals("userID").(string)
    if !ok || userIDStr == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "User ID missing or invalid from token locals",
        })
    }
    
    // Ambil Role
    userRole, ok := c.Locals("role").(string)
    if !ok || userRole == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "User role missing or invalid",
        })
    }

    // 🛑 LOGIKA PENGENDALIAN AKSES BERBASIS KEPEMILIKAN DAN ROLE 🛑
    
    // 1. Cek apakah pengguna adalah pemilik data (self-referential check)
    isOwner := targetStudentID == userIDStr
    
    // 2. Jika bukan pemilik data (isOwner == false):
    //    Cek apakah dia Mahasiswa. Jika ya, akses ditolak karena hanya boleh self-access.
    if !isOwner && userRole == "Mahasiswa" {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "code": http.StatusForbidden, // Tambahkan code 403 untuk konsistensi output Anda sebelumnya
            "error": "Akses ditolak: Anda hanya dapat melihat data prestasi Anda sendiri.",
        })
    }
    // Catatan: Jika userRole adalah Admin/Dosen, logic akan dilanjutkan ke Service.
    // Dosen/Admin akan dicek lagi di Service (misalnya, apakah dosen tersebut adalah dosen walinya).
    // --------------------------------------------------------------------------

    // Parse string ID ke UUID untuk dikirim ke Service
    parsedUserID, err := uuid.Parse(userIDStr)
    if err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
    }
    
    // 4. Panggil service untuk ambil achievements
    // Service akan memeriksa apakah:
    // a) Role adalah Administrator/Dosen
    // b) Role Dosen harus terikat pada Mahasiswa ini (logic di Service layer)
    result, err := achievementSvc.ListAchievementsByStudentID(c.Context(), targetStudentID, parsedUserID, userRole)
    if err != nil {
        // Tangani error otorisasi/forbidden yang mungkin dilempar dari Service
        if strings.Contains(err.Error(), "forbidden") {
            return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "error": err.Error(),
            })
        }
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    // 5. Return hasil
    return c.JSON(fiber.Map{
        "status": "success",
        "data": result,
    })
})
}