package service

import (
    "sistempelaporan/app/model"
    "sistempelaporan/app/repository"
    "sistempelaporan/helper"

    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "strings"
)

// --- FR-009: Create User (Admin) ---
// POST /api/v1/users
func CreateNewUser(c *fiber.Ctx) error {
	// 1. Parsing Request
	var req struct {
		User  model.User`json:"user"`
		Student *model.Student `json:"student"`
		Lecturer *model.Lecturer `json:"lecturer"`
		Password string `json:"password"` // Wajib diisi!
	}

	if err := c.BodyParser(&req); err != nil {
		return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
	}

	// 💡 FIX 1.1: Wajibkan Password Ada
	if req.Password == "" {
		return helper.Error(c, fiber.StatusBadRequest, "Password wajib diisi.", nil)
	}

	// 2. Logic Bisnis (Hash Password)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal hash password", err.Error())
	}
	req.User.PasswordHash = string(hashedPassword)

	// 3. Generate ID Baru (Jika belum ada di input)
	if req.User.ID == uuid.Nil {
		req.User.ID = uuid.New()
	}
	
	if req.Student != nil {
		if req.Student.ID == uuid.Nil {
			req.Student.ID = uuid.New()
		}
	}
	if req.Lecturer != nil {
		if req.Lecturer.ID == uuid.Nil {
			req.Lecturer.ID = uuid.New()
		}
	}

	// 4. Panggil Repository Transaction
	if err := repository.CreateUserWithProfile(&req.User, req.Student, req.Lecturer); err != nil {
		
        // 💡 FIX 1.2: Tangani error Foreign Key dan Conflict dari Repository
        if strings.Contains(err.Error(), "Role ID yang dimasukkan tidak ditemukan") {
            return helper.Error(c, fiber.StatusConflict, "Role ID tidak valid atau tidak ada di database.", nil)
        }
        if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
            return helper.Error(c, fiber.StatusConflict, "Username atau Email sudah digunakan.", nil)
        }
        
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal membuat user", err.Error())
	}

	return helper.Created(c, nil, "User berhasil dibuat")
}
// --- FR-009: Get All Users (Admin) ---
// GET /api/v1/users
// [cite: 733]
func GetAllUsers(c *fiber.Ctx) error {
    users, err := repository.FindAllUsers()
    if err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data user", err.Error())
    }
    return helper.Success(c, users, "Data semua user berhasil diambil")
}

// --- FR-009: Get User By ID (Admin) ---
// GET /api/v1/users/:id
// [cite: 734]
func GetUserByID(c *fiber.Ctx) error {
    id := c.Params("id")
    user, err := repository.FindUserByID(id)
    if err != nil {
        return helper.Error(c, fiber.StatusNotFound, "User tidak ditemukan", err.Error())
    }
    return helper.Success(c, user, "Detail user berhasil diambil")
}

// --- FR-009: Update User General (Admin) ---
// PUT /api/v1/users/:id
// [cite: 736]
func UpdateUser(c *fiber.Ctx) error {
    id := c.Params("id")

    var req struct {
        Username string `json:"username"`
        FullName string `json:"full_name"`
        Email    string `json:"email"`
    }

    if err := c.BodyParser(&req); err != nil {
        return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
    }

    if err := repository.UpdateUserGeneral(id, req.Username, req.FullName, req.Email); err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal update user", err.Error())
    }

    return helper.Success(c, nil, "User berhasil diupdate")
}

// --- FR-009: Update User Role (Admin) ---
// PUT /api/v1/users/:id/role
// [cite: 738]
func UpdateUserRole(c *fiber.Ctx) error {
    id := c.Params("id")

    var req struct {
        RoleID string `json:"role_id"`
    }

    if err := c.BodyParser(&req); err != nil {
        return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
    }

    if err := repository.UpdateUserRole(id, req.RoleID); err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal update role user", err.Error())
    }

    return helper.Success(c, nil, "Role user berhasil diupdate")
}

// --- FR-009: Delete User (Admin) ---
// DELETE /api/v1/users/:id
// [cite: 737]
func DeleteUser(c *fiber.Ctx) error {
    id := c.Params("id")

    if err := repository.DeleteUserByID(id); err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal menghapus user", err.Error())
    }

    return helper.Success(c, nil, "User berhasil dihapus (soft delete)")
}

// --- Helper: Assign Dosen Wali ---
// PUT /api/v1/students/:id/advisor
// [cite: 755]
func SetAdvisor(c *fiber.Ctx) error {
    studentID := c.Params("studentId")
    
    var req struct {
        AdvisorID string `json:"advisor_id"`
    }
    if err := c.BodyParser(&req); err != nil {
        return helper.Error(c, fiber.StatusBadRequest, "Input tidak valid", err.Error())
    }

    if err := repository.AssignAdvisorToStudent(studentID, req.AdvisorID); err != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal assign dosen wali", err.Error())
    }

    return helper.Success(c, nil, "Dosen wali berhasil diassign")
}