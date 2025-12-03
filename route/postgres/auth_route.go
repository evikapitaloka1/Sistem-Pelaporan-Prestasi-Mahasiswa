package routes

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	
	// Import yang diperlukan berdasarkan penggunaan di body
	model "uas/app/model/postgres" 	
	service "uas/app/service/postgres" 
)

// SetupAuthRoutes mendefinisikan semua rute autentikasi.
// Menerima jwtMiddleware yang sudah siap (termasuk TokenBlacklistChecker) untuk rute yang dilindungi.
func SetupAuthRoutes(router fiber.Router, authService *service.AuthService, jwtMiddleware fiber.Handler) {

	auth := router.Group("/auth")

	// ================= LOGIN =================
	auth.Post("/login", func(c *fiber.Ctx) error {
		var req model.LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}

		resp, err := authService.Login(c.Context(), req.Username, req.Password)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"data": resp})
	})

	// ================= REFRESH =================
	auth.Post("/refresh", func(c *fiber.Ctx) error {
		var refreshToken string
		
		// 1. Coba ambil dari Authorization Header (Bearer Token)
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			refreshToken = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 2. Jika tidak ada di Header, coba ambil dari JSON Body
		if refreshToken == "" {
			var payload struct {
				RefreshToken string `json:"refresh_token"`
			}

			// Mengabaikan error parsing, karena token mungkin ada di header
			_ = c.BodyParser(&payload) 
			
			if payload.RefreshToken != "" {
				refreshToken = payload.RefreshToken
			}
		}
		
		// 3. Validasi akhir token
		if refreshToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token missing from header or body"})
		}
		
		token, err := authService.Refresh(c.Context(), refreshToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"token": token})
	})

	// ================= LOGOUT (DENGAN MIDDLEWARE) =================
	// ✅ Middleware jwtMiddleware WAJIB di sini agar JTI tersedia.
	auth.Post("/logout", jwtMiddleware, func(c *fiber.Ctx) error {
		
		// 1. Ambil JTI dari Locals (diset oleh JWTMiddleware)
		jti, ok := c.Locals("jti").(string)
		if !ok || jti == "" {
			// ✅ PERBAIKAN STATUS: Menggunakan 401 Unauthorized karena token tidak memiliki claim yang krusial (JTI)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "JTI not available in locals for logout. Token is missing JTI claim."})
		}
		
		// 2. Panggil service untuk mencabut token (memerlukan JTI)
		err := authService.Logout(c.Context(), jti) 
		if err != nil {
			// Asumsi error di service adalah internal atau masalah DB, kembalikan 500
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"message": "logout success"})
	}) 

	// ================= PROFILE (DENGAN MIDDLEWARE) =================
	auth.Get("/profile", jwtMiddleware, func(c *fiber.Ctx) error {

		// 1. Ambil userID dari Locals (sudah uuid.UUID berkat perbaikan middleware)
		userIDLocal := c.Locals("userID")
		
		uid, ok := userIDLocal.(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User ID missing or invalid type in token payload"})
		}

		// 2. Panggil Service
		user, err := authService.Profile(c.Context(), uid)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"data": user})
	})
}