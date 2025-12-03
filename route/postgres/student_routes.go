package routes

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
	rbacRead := mw.RBACMiddleware("student:read", authSvc)
	rbacUpdate := mw.RBACMiddleware("student:update", authSvc)

	// Grup Route menggunakan jwtMiddleware yang sudah di-inject
	students := api.Group("/students", jwtMiddleware) // Gunakan handler yang di-inject

	// =========================================================================================
	// 1. GET list (Hanya Admin yang boleh, menggunakan rbacRead yang dimodifikasi atau rbacManage)
	// =========================================================================================
	// Catatan: Jika student:read hanya untuk admin, gunakan rbacRead yang dimodifikasi
	// agar admin dengan user:manage bisa mengakses ini.
	students.Get("/", rbacRead, func(c *fiber.Ctx) error {
		result, err := studentSvc.ListStudents(c.Context())
		if err != nil {
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
	students.Get("/:id", rbacRead, func(c *fiber.Ctx) error {
		targetStudentID := c.Params("id")
		
		// Ambil data User ID dan Role dari Locals
		userIDStr, ok := c.Locals("userID").(string)
		if !ok || userIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID missing or invalid from token"})
		}
		
		userRole, _ := c.Locals("role").(string) // Ambil role untuk referensi

		// 🛑 LOGIKA PENGENDALIAN AKSES BERBASIS KEPEMILIKAN 🛑
		isOwner := targetStudentID == userIDStr
		
		// Admin/Superadmin diizinkan akses ke siapa pun (asumsi RBAC sudah mengecek izin read)
		// Kita butuh fungsi di service/middleware untuk cek apakah user adalah admin
		// Untuk sementara, kita mengandalkan bahwa jika user bukan owner,
		// maka RBAC middleware (rbacRead) harusnya sudah memastikan user adalah Admin.
		// Namun, cara yang lebih bersih adalah:

		// Jika pengguna mencoba melihat ID orang lain DAN dia bukan Admin/Dosen yang berhak, tolak.
		// Karena kita menggunakan rbacRead, kita berasumsi bahwa student:read sudah cukup.
		// Jika ini adalah data sendiri, kita biarkan lewat.
		
		if !isOwner && userRole != "Administrator" {
		    // Catatan: Jika userRole tidak membedakan Dosen dan Mahasiswa, logika ini harus diperluas.
		    // Jika bukan owner dan bukan Admin, tolak.
            return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "error": "Akses ditolak. Anda hanya dapat melihat data profil Anda sendiri.",
            })
		}
		// --------------------------------------------------------------------------

		result, err := studentSvc.GetStudentDetail(c.Context(), targetStudentID)
		if err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "success", "data": result})
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
	students.Get("/:id/achievements", rbacRead, func(c *fiber.Ctx) error {
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

		// 🛑 LOGIKA PENGENDALIAN AKSES BERBASIS KEPEMILIKAN 🛑
		// 1. Cek apakah pengguna adalah pemilik data (self-referential check)
		isOwner := targetStudentID == userIDStr
		
		// 2. Jika bukan pemilik data, kita perlu cek role.
		// Jika role adalah "Mahasiswa" dan target ID bukan ID-nya, tolak segera.
		// Dosen atau Admin akan diizinkan melalui logika di service (Poin 4).
		if !isOwner && userRole == "Mahasiswa" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Akses ditolak. Anda hanya dapat melihat data prestasi Anda sendiri.",
			})
		}
		// --------------------------------------------------------------------------

		// Parse string ID ke UUID untuk dikirim ke Service
		parsedUserID, err := uuid.Parse(userIDStr)
		if err != nil {
			// Ini seharusnya tidak terjadi jika token JWT valid
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user ID format in token"})
		}
		
		// 4. Panggil service untuk ambil achievements
		// Logika: Service akan memeriksa apakah pengguna adalah admin, atau dosen dari mahasiswa tersebut.
		result, err := achievementSvc.ListAchievementsByStudentID(c.Context(), targetStudentID, parsedUserID, userRole)
		if err != nil {
			if err.Error() == "forbidden: tidak memiliki hak akses untuk melihat prestasi mahasiswa ini" {
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